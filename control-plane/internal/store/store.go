// Package store persists agent state to postgres via pgxpool.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgx connection pool. column names mirror
// database/schema.sql (agents table).
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// on first sight insert the row; on reconnect refresh the reported fields
// and mark the agent healthy with a fresh last_seen. a blank incoming hash
// keeps the previously recorded one instead of clobbering it.
const upsertAgentSQL = `
INSERT INTO agents (instance_uid, hostname, agent_type, version, status, last_seen, effective_config_hash, labels)
VALUES ($1, $2, $3, $4, 'healthy', now(), NULLIF($5, ''), $6::jsonb)
ON CONFLICT (instance_uid) DO UPDATE SET
  hostname = EXCLUDED.hostname,
  agent_type = EXCLUDED.agent_type,
  version = EXCLUDED.version,
  status = 'healthy',
  last_seen = now(),
  effective_config_hash = COALESCE(NULLIF(EXCLUDED.effective_config_hash, ''), agents.effective_config_hash),
  labels = EXCLUDED.labels`

// UpsertAgent records an agent as healthy, inserting or refreshing its row.
func (s *Store) UpsertAgent(ctx context.Context, uid, hostname, agentType, version, effectiveHash string, labels map[string]string) error {
	if labels == nil {
		labels = map[string]string{}
	}
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return fmt.Errorf("marshal labels for agent %s: %w", uid, err)
	}
	_, err = s.pool.Exec(ctx, upsertAgentSQL, uid, hostname, agentType, version, effectiveHash, labelsJSON)
	if err != nil {
		return fmt.Errorf("upsert agent %s: %w", uid, err)
	}
	return nil
}

const markDisconnectedSQL = `
UPDATE agents SET status = 'disconnected', last_seen = now() WHERE instance_uid = $1`

// MarkDisconnected flips an agent to disconnected when its connection closes.
func (s *Store) MarkDisconnected(ctx context.Context, uid string) error {
	_, err := s.pool.Exec(ctx, markDisconnectedSQL, uid)
	if err != nil {
		return fmt.Errorf("mark agent %s disconnected: %w", uid, err)
	}
	return nil
}

// DesiredConfig is a configuration's current version, the reconciler's
// desired state.
type DesiredConfig struct {
	ConfigID  int
	Selector  string
	VersionID int
	VersionNo int
	Hash      string
	YAML      string
}

// AgentState is the observed side the reconciler compares against.
type AgentState struct {
	Labels        map[string]string
	EffectiveHash string
}

// ordered by config id so selector ties resolve to the lowest id.
const desiredConfigsSQL = `
SELECT c.id, COALESCE(c.label_selector, ''), v.id, v.version_no, v.hash, v.yaml
FROM configurations c
JOIN config_versions v ON v.id = c.current_version_id
ORDER BY c.id`

// DesiredConfigs returns every configuration that has a current version.
func (s *Store) DesiredConfigs(ctx context.Context) ([]DesiredConfig, error) {
	rows, err := s.pool.Query(ctx, desiredConfigsSQL)
	if err != nil {
		return nil, fmt.Errorf("query desired configs: %w", err)
	}
	defer rows.Close()
	var out []DesiredConfig
	for rows.Next() {
		var d DesiredConfig
		if err := rows.Scan(&d.ConfigID, &d.Selector, &d.VersionID, &d.VersionNo, &d.Hash, &d.YAML); err != nil {
			return nil, fmt.Errorf("scan desired config: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

const agentStatesSQL = `
SELECT instance_uid, labels, COALESCE(effective_config_hash, '')
FROM agents WHERE instance_uid = ANY($1)`

// AgentStates returns labels and effective hash for the given uids.
// unparseable labels degrade to empty maps rather than failing the tick.
func (s *Store) AgentStates(ctx context.Context, uids []string) (map[string]AgentState, error) {
	rows, err := s.pool.Query(ctx, agentStatesSQL, uids)
	if err != nil {
		return nil, fmt.Errorf("query agent states: %w", err)
	}
	defer rows.Close()
	out := make(map[string]AgentState, len(uids))
	for rows.Next() {
		var uid string
		var raw []byte
		var st AgentState
		if err := rows.Scan(&uid, &raw, &st.EffectiveHash); err != nil {
			return nil, fmt.Errorf("scan agent state: %w", err)
		}
		if err := json.Unmarshal(raw, &st.Labels); err != nil {
			st.Labels = map[string]string{}
		}
		out[uid] = st
	}
	return out, rows.Err()
}

const latestRolloutSQL = `
SELECT status FROM rollouts
WHERE agent_instance_uid = $1 AND config_version_id = $2
ORDER BY id DESC LIMIT 1`

// LatestRolloutStatus returns the newest rollout status for the pair;
// found is false when none exists.
func (s *Store) LatestRolloutStatus(ctx context.Context, uid string, versionID int) (string, bool, error) {
	var status string
	err := s.pool.QueryRow(ctx, latestRolloutSQL, uid, versionID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("latest rollout for %s v%d: %w", uid, versionID, err)
	}
	return status, true, nil
}

const createRolloutSQL = `
INSERT INTO rollouts (config_version_id, agent_instance_uid, status)
VALUES ($1, $2, 'pending')`

// CreateRollout records that a config version was pushed to an agent.
func (s *Store) CreateRollout(ctx context.Context, versionID int, uid string) error {
	if _, err := s.pool.Exec(ctx, createRolloutSQL, versionID, uid); err != nil {
		return fmt.Errorf("create rollout for %s v%d: %w", uid, versionID, err)
	}
	return nil
}

// error column is varchar(255); truncate before writing.
const resolveRolloutsSQL = `
UPDATE rollouts SET
  status = $3,
  applied_at = CASE WHEN $3 = 'applied' THEN now() ELSE applied_at END,
  error = NULLIF($4, '')
WHERE agent_instance_uid = $1 AND status = 'pending'
  AND config_version_id IN (SELECT id FROM config_versions WHERE hash = $2)`

// ResolveRollouts settles the agent's pending rollouts whose version
// matches the acknowledged config hash. status is applied or failed.
func (s *Store) ResolveRollouts(ctx context.Context, uid, hash, status, errMsg string) error {
	if len(errMsg) > 255 {
		errMsg = errMsg[:255]
	}
	if _, err := s.pool.Exec(ctx, resolveRolloutsSQL, uid, hash, status, errMsg); err != nil {
		return fmt.Errorf("resolve rollouts for %s hash %s: %w", uid, hash, err)
	}
	return nil
}
