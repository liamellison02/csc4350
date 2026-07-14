// Package reconcile pushes desired config versions to drifted agents.
package reconcile

import (
	"context"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/liamellison02/csc4350/control-plane/internal/selector"
	"github.com/liamellison02/csc4350/control-plane/internal/store"
)

// Store is the persistence surface the reconciler reads and writes.
type Store interface {
	DesiredConfigs(ctx context.Context) ([]store.DesiredConfig, error)
	AgentStates(ctx context.Context, uids []string) (map[string]store.AgentState, error)
	LatestRolloutStatus(ctx context.Context, uid string, versionID int) (string, bool, error)
	CreateRollout(ctx context.Context, versionID int, uid string) error
}

// Sender pushes configs to connected agents.
type Sender interface {
	ConnectedUIDs() []string
	SendConfig(ctx context.Context, uid string, yamlBody, hash []byte) error
}

// Reconciler compares desired vs effective config per connected agent
// and pushes only on mismatch.
type Reconciler struct {
	store  Store
	sender Sender
	log    *log.Logger
}

// New builds a reconciler.
func New(st Store, sender Sender, logger *log.Logger) *Reconciler {
	return &Reconciler{store: st, sender: sender, log: logger}
}

// Run ticks until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.Tick(ctx); err != nil {
				r.log.Printf("ERROR reconcile tick: %v", err)
			}
		}
	}
}

// Tick runs one reconcile pass. per-agent problems log and continue;
// only whole-pass failures (db reads) return an error.
func (r *Reconciler) Tick(ctx context.Context) error {
	uids := r.sender.ConnectedUIDs()
	if len(uids) == 0 {
		return nil
	}
	desired, err := r.store.DesiredConfigs(ctx)
	if err != nil {
		return err
	}
	if len(desired) == 0 {
		return nil
	}
	states, err := r.store.AgentStates(ctx, uids)
	if err != nil {
		return err
	}
	for _, uid := range uids {
		st, ok := states[uid]
		if !ok {
			continue
		}
		d, ok := r.pick(desired, st.Labels)
		if !ok {
			continue
		}
		// effective hashes are lowercase hex; normalize the stored side so a
		// case difference cannot defeat convergence.
		d.Hash = strings.ToLower(d.Hash)
		if st.EffectiveHash == d.Hash {
			continue
		}
		status, found, err := r.store.LatestRolloutStatus(ctx, uid, d.VersionID)
		if err != nil {
			r.log.Printf("ERROR rollout lookup %s: %v", uid, err)
			continue
		}
		// pending: push in flight. failed: wait for a new version.
		if found && (status == "pending" || status == "failed") {
			continue
		}
		hashBytes, err := hex.DecodeString(d.Hash)
		if err != nil {
			r.log.Printf("skip config %d for %s: hash %q is not hex", d.ConfigID, uid, d.Hash)
			continue
		}
		if err := r.sender.SendConfig(ctx, uid, []byte(d.YAML), hashBytes); err != nil {
			r.log.Printf("ERROR push config %d to %s: %v", d.ConfigID, uid, err)
			continue
		}
		if err := r.store.CreateRollout(ctx, d.VersionID, uid); err != nil {
			r.log.Printf("ERROR record rollout for %s: %v", uid, err)
			continue
		}
		r.log.Printf("pushed config %d version %d to agent %s", d.ConfigID, d.VersionNo, uid)
	}
	return nil
}

// pick returns the most specific matching config: most selector pairs
// wins, ties go to the earliest (desired is ordered by config id).
func (r *Reconciler) pick(desired []store.DesiredConfig, labels map[string]string) (store.DesiredConfig, bool) {
	var best store.DesiredConfig
	bestPairs := -1
	for _, d := range desired {
		sel, err := selector.Parse(d.Selector)
		if err != nil {
			r.log.Printf("skip configuration %d: %v", d.ConfigID, err)
			continue
		}
		if !selector.Matches(sel, labels) {
			continue
		}
		if len(sel) > bestPairs {
			best, bestPairs = d, len(sel)
		}
	}
	return best, bestPairs >= 0
}
