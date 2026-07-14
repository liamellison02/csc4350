package reconcile

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"testing"

	"github.com/liamellison02/csc4350/control-plane/internal/store"
)

// fakeStore serves canned desired configs and agent states, answers rollout
// lookups from latest (keyed "uid/versionID"), and records CreateRollout
// calls. reads counts DesiredConfigs+AgentStates calls. error fields force
// the corresponding method to fail without recording.
type fakeStore struct {
	desired []store.DesiredConfig
	states  map[string]store.AgentState
	latest  map[string]string

	rollouts []rolloutCall
	reads    int

	desiredErr, statesErr, latestErr, createErr error
}

type rolloutCall struct {
	versionID int
	uid       string
}

func (f *fakeStore) DesiredConfigs(context.Context) ([]store.DesiredConfig, error) {
	f.reads++
	if f.desiredErr != nil {
		return nil, f.desiredErr
	}
	return f.desired, nil
}

func (f *fakeStore) AgentStates(context.Context, []string) (map[string]store.AgentState, error) {
	f.reads++
	if f.statesErr != nil {
		return nil, f.statesErr
	}
	return f.states, nil
}

func (f *fakeStore) LatestRolloutStatus(_ context.Context, uid string, versionID int) (string, bool, error) {
	if f.latestErr != nil {
		return "", false, f.latestErr
	}
	status, ok := f.latest[fmt.Sprintf("%s/%d", uid, versionID)]
	return status, ok, nil
}

func (f *fakeStore) CreateRollout(_ context.Context, versionID int, uid string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.rollouts = append(f.rollouts, rolloutCall{versionID: versionID, uid: uid})
	return nil
}

// fakeSender serves canned connected uids and records successful SendConfig
// calls; sendErr forces every send to fail without recording.
type fakeSender struct {
	uids    []string
	sends   []sendCall
	sendErr error
}

type sendCall struct {
	uid  string
	yaml string
	hash string
}

func (f *fakeSender) ConnectedUIDs() []string { return f.uids }

func (f *fakeSender) SendConfig(_ context.Context, uid string, yamlBody, hash []byte) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sends = append(f.sends, sendCall{uid: uid, yaml: string(yamlBody), hash: hex.EncodeToString(hash)})
	return nil
}

func dc(id int, sel string, verID, verNo int, hash, yaml string) store.DesiredConfig {
	return store.DesiredConfig{ConfigID: id, Selector: sel, VersionID: verID, VersionNo: verNo, Hash: hash, YAML: yaml}
}

func TestTick(t *testing.T) {
	cases := []struct {
		name    string
		uids    []string
		desired []store.DesiredConfig
		states  map[string]store.AgentState
		latest  map[string]string
		sendErr error

		wantSends    []sendCall
		wantRollouts []rolloutCall
		wantReads    int
	}{
		{
			name:    "no connected agents",
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "aa01", "yaml-1")},
			states:  map[string]store.AgentState{},
		},
		{
			name:    "hash already matches",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "aa01", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: "aa01"},
			},
			wantReads: 2,
		},
		{
			name:    "uppercase desired hash matches lowercase effective",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "ABCD12", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: "abcd12"},
			},
			wantReads: 2,
		},
		{
			name:    "mismatch pushes and records",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "bb02", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: "aa01"},
			},
			wantSends:    []sendCall{{uid: "u1", yaml: "yaml-1", hash: "bb02"}},
			wantRollouts: []rolloutCall{{versionID: 11, uid: "u1"}},
			wantReads:    2,
		},
		{
			name:    "pending rollout skips",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "bb02", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: "aa01"},
			},
			latest:    map[string]string{"u1/11": "pending"},
			wantReads: 2,
		},
		{
			name:    "failed rollout blocks same version",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "bb02", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: "aa01"},
			},
			latest:    map[string]string{"u1/11": "failed"},
			wantReads: 2,
		},
		{
			name: "most specific selector wins",
			uids: []string{"u1"},
			desired: []store.DesiredConfig{
				dc(1, "", 11, 1, "aa01", "yaml-1"),
				dc(2, "env=prod", 22, 1, "bb02", "yaml-2"),
			},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{"env": "prod", "region": "e"}, EffectiveHash: ""},
			},
			wantSends:    []sendCall{{uid: "u1", yaml: "yaml-2", hash: "bb02"}},
			wantRollouts: []rolloutCall{{versionID: 22, uid: "u1"}},
			wantReads:    2,
		},
		{
			name: "tie goes to lowest id",
			uids: []string{"u1"},
			desired: []store.DesiredConfig{
				dc(1, "env=prod", 11, 1, "aa01", "yaml-1"),
				dc(2, "env=prod", 22, 1, "bb02", "yaml-2"),
			},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{"env": "prod"}, EffectiveHash: ""},
			},
			wantSends:    []sendCall{{uid: "u1", yaml: "yaml-1", hash: "aa01"}},
			wantRollouts: []rolloutCall{{versionID: 11, uid: "u1"}},
			wantReads:    2,
		},
		{
			name: "malformed selector skipped",
			uids: []string{"u1"},
			desired: []store.DesiredConfig{
				dc(1, "oops", 11, 1, "aa01", "yaml-1"),
				dc(2, "", 22, 1, "bb02", "yaml-2"),
			},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: ""},
			},
			wantSends:    []sendCall{{uid: "u1", yaml: "yaml-2", hash: "bb02"}},
			wantRollouts: []rolloutCall{{versionID: 22, uid: "u1"}},
			wantReads:    2,
		},
		{
			name:    "non-hex hash skipped",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "hash-prod-v1", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: ""},
			},
			wantReads: 2,
		},
		{
			name:    "send failure records nothing",
			uids:    []string{"u1"},
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "aa01", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: ""},
			},
			sendErr:   errors.New("boom"),
			wantReads: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakeStore{desired: tc.desired, states: tc.states, latest: tc.latest}
			snd := &fakeSender{uids: tc.uids, sendErr: tc.sendErr}
			r := New(st, snd, log.New(io.Discard, "", 0))

			if err := r.Tick(context.Background()); err != nil {
				t.Fatalf("Tick() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(snd.sends, tc.wantSends) {
				t.Errorf("sends = %+v, want %+v", snd.sends, tc.wantSends)
			}
			if !reflect.DeepEqual(st.rollouts, tc.wantRollouts) {
				t.Errorf("rollouts = %+v, want %+v", st.rollouts, tc.wantRollouts)
			}
			if st.reads != tc.wantReads {
				t.Errorf("store reads = %d, want %d", st.reads, tc.wantReads)
			}
		})
	}
}

// whole-pass store read failures must fail the tick; per-agent store
// problems must log and continue instead.
func TestTickStoreErrors(t *testing.T) {
	boom := errors.New("boom")
	base := func() *fakeStore {
		return &fakeStore{
			desired: []store.DesiredConfig{dc(1, "", 11, 1, "aa01", "yaml-1")},
			states: map[string]store.AgentState{
				"u1": {Labels: map[string]string{}, EffectiveHash: ""},
			},
		}
	}
	cases := []struct {
		name    string
		mutate  func(*fakeStore)
		wantErr bool

		wantSends    []sendCall
		wantRollouts []rolloutCall
	}{
		{
			name:    "desired configs error fails tick",
			mutate:  func(f *fakeStore) { f.desiredErr = boom },
			wantErr: true,
		},
		{
			name:    "agent states error fails tick",
			mutate:  func(f *fakeStore) { f.statesErr = boom },
			wantErr: true,
		},
		{
			name:   "rollout lookup error continues",
			mutate: func(f *fakeStore) { f.latestErr = boom },
		},
		{
			name:      "create rollout error continues",
			mutate:    func(f *fakeStore) { f.createErr = boom },
			wantSends: []sendCall{{uid: "u1", yaml: "yaml-1", hash: "aa01"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := base()
			tc.mutate(st)
			snd := &fakeSender{uids: []string{"u1"}}
			r := New(st, snd, log.New(io.Discard, "", 0))

			err := r.Tick(context.Background())
			if tc.wantErr && err == nil {
				t.Fatal("Tick() error = nil, want non-nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Tick() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(snd.sends, tc.wantSends) {
				t.Errorf("sends = %+v, want %+v", snd.sends, tc.wantSends)
			}
			if !reflect.DeepEqual(st.rollouts, tc.wantRollouts) {
				t.Errorf("rollouts = %+v, want %+v", st.rollouts, tc.wantRollouts)
			}
		})
	}
}
