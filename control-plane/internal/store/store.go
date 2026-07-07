// Package store persists agent state to postgres via pgxpool.
package store

import (
	"context"
	"fmt"

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
INSERT INTO agents (instance_uid, hostname, agent_type, version, status, last_seen, effective_config_hash)
VALUES ($1, $2, $3, $4, 'healthy', now(), NULLIF($5, ''))
ON CONFLICT (instance_uid) DO UPDATE SET
  hostname = EXCLUDED.hostname,
  agent_type = EXCLUDED.agent_type,
  version = EXCLUDED.version,
  status = 'healthy',
  last_seen = now(),
  effective_config_hash = COALESCE(NULLIF(EXCLUDED.effective_config_hash, ''), agents.effective_config_hash)`

// UpsertAgent records an agent as healthy, inserting or refreshing its row.
func (s *Store) UpsertAgent(ctx context.Context, uid, hostname, agentType, version, effectiveHash string) error {
	_, err := s.pool.Exec(ctx, upsertAgentSQL, uid, hostname, agentType, version, effectiveHash)
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
