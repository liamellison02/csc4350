package opamp

import (
	"context"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
)

// fakeStore records calls without touching a database. labels are kept in a
// parallel slice so upsertCall stays comparable for the == assertions below.
type fakeStore struct {
	upserts      []upsertCall
	labels       []map[string]string
	disconnected []string
	resolves     []resolveCall
	effHashes    []effHashCall
}

type upsertCall struct {
	uid, hostname, agentType, version, hash string
}

type resolveCall struct {
	uid, hash, status, errMsg string
}

type effHashCall struct {
	uid, hash string
}

func (f *fakeStore) UpsertAgent(_ context.Context, uid, hostname, agentType, version, hash string, labels map[string]string) error {
	f.upserts = append(f.upserts, upsertCall{uid, hostname, agentType, version, hash})
	f.labels = append(f.labels, labels)
	return nil
}

func (f *fakeStore) MarkDisconnected(_ context.Context, uid string) error {
	f.disconnected = append(f.disconnected, uid)
	return nil
}

func (f *fakeStore) ResolveRollouts(_ context.Context, uid, hash, status, errMsg string) error {
	f.resolves = append(f.resolves, resolveCall{uid, hash, status, errMsg})
	return nil
}

func (f *fakeStore) SetEffectiveConfigHash(_ context.Context, uid, hash string) error {
	f.effHashes = append(f.effHashes, effHashCall{uid, hash})
	return nil
}

// fakeConn is a comparable Connection usable as a map key. it records the
// last message pushed through Send.
type fakeConn struct {
	lastSent *protobufs.ServerToAgent
}

func (*fakeConn) Connection() net.Conn { return nil }

func (c *fakeConn) Send(_ context.Context, msg *protobufs.ServerToAgent) error {
	c.lastSent = msg
	return nil
}

func (*fakeConn) Disconnect() error { return nil }

// fakeConn must satisfy the opamp Connection interface used as a map key.
var _ types.Connection = (*fakeConn)(nil)

func newTestServer(store AgentStore) *Server {
	return NewServer(store, ":0", log.New(io.Discard, "", 0))
}

var testUID = []byte{0x01, 0x93, 0xd2, 0x4a, 0x2b, 0xc4, 0x7e, 0x0a, 0x9f, 0x11, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

func descMsg() *protobufs.AgentToServer {
	return &protobufs.AgentToServer{
		InstanceUid: testUID,
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				strKV(attrHostName, "collector-prod-01"),
				strKV(attrServiceName, "io.opentelemetry.collector"),
				strKV(attrServiceVersion, "0.147.0"),
			},
		},
	}
}

func TestOnConnectingAccepts(t *testing.T) {
	s := newTestServer(&fakeStore{})
	resp := s.onConnecting(&http.Request{})
	if !resp.Accept {
		t.Fatal("onConnecting did not accept the connection")
	}
	if resp.ConnectionCallbacks.OnMessage == nil {
		t.Error("OnMessage callback not wired")
	}
	if resp.ConnectionCallbacks.OnConnectionClose == nil {
		t.Error("OnConnectionClose callback not wired")
	}
}

func TestOnMessageUpsertsWithDescription(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)
	conn := &fakeConn{}

	resp := s.onMessage(context.Background(), conn, descMsg())

	if len(store.upserts) != 1 {
		t.Fatalf("got %d upserts, want 1", len(store.upserts))
	}
	got := store.upserts[0]
	want := upsertCall{InstanceUID(testUID), "collector-prod-01", "io.opentelemetry.collector", "0.147.0", ""}
	if got != want {
		t.Errorf("upsert = %+v, want %+v", got, want)
	}
	if string(resp.GetInstanceUid()) != string(testUID) {
		t.Errorf("response uid = %x, want %x", resp.GetInstanceUid(), testUID)
	}
}

func TestOnMessageRecordsLabels(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)
	msg := descMsg()
	msg.AgentDescription.NonIdentifyingAttributes = []*protobufs.KeyValue{
		strKV("env", "prod"),
		strKV("region", "us-east"),
	}

	s.onMessage(context.Background(), &fakeConn{}, msg)

	if len(store.labels) != 1 {
		t.Fatalf("got %d label records, want 1", len(store.labels))
	}
	want := map[string]string{"env": "prod", "region": "us-east"}
	if !maps.Equal(store.labels[0], want) {
		t.Errorf("labels = %v, want %v", store.labels[0], want)
	}
}

func TestOnMessageEncodesEffectiveConfigHash(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)
	msg := descMsg()
	msg.RemoteConfigStatus = &protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: []byte{0xde, 0xad, 0xbe, 0xef},
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
	}

	s.onMessage(context.Background(), &fakeConn{}, msg)

	if len(store.upserts) != 1 {
		t.Fatalf("got %d upserts, want 1", len(store.upserts))
	}
	if store.upserts[0].hash != "deadbeef" {
		t.Errorf("hash = %q, want %q", store.upserts[0].hash, "deadbeef")
	}
}

// a reconnect after a failed apply must not record the failed config's hash
// as effective; the blank hash keeps the previously stored one.
func TestOnMessageFailedApplyHashNotEffective(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)
	msg := descMsg()
	msg.RemoteConfigStatus = &protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: []byte{0xde, 0xad, 0xbe, 0xef},
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
	}

	s.onMessage(context.Background(), &fakeConn{}, msg)

	if len(store.upserts) != 1 {
		t.Fatalf("got %d upserts, want 1", len(store.upserts))
	}
	if store.upserts[0].hash != "" {
		t.Errorf("hash = %q, want empty for a failed apply", store.upserts[0].hash)
	}
}

func TestOnMessageWithoutDescriptionDoesNotUpsert(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)

	resp := s.onMessage(context.Background(), &fakeConn{}, &protobufs.AgentToServer{InstanceUid: testUID})

	if len(store.upserts) != 0 {
		t.Errorf("got %d upserts, want 0 for a description-less message", len(store.upserts))
	}
	if string(resp.GetInstanceUid()) != string(testUID) {
		t.Errorf("response uid = %x, want %x", resp.GetInstanceUid(), testUID)
	}
}

func TestConnectionCloseMarksDisconnected(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)
	conn := &fakeConn{}

	// a message first so the server learns the connection's uid.
	s.onMessage(context.Background(), conn, descMsg())
	s.onConnectionClose(conn)

	if len(store.disconnected) != 1 || store.disconnected[0] != InstanceUID(testUID) {
		t.Errorf("disconnected = %v, want [%s]", store.disconnected, InstanceUID(testUID))
	}
}

func TestConnectionCloseUntrackedIsNoop(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)

	s.onConnectionClose(&fakeConn{})

	if len(store.disconnected) != 0 {
		t.Errorf("disconnected = %v, want none for an untracked connection", store.disconnected)
	}
}

func rolloutStatusMsg(status protobufs.RemoteConfigStatuses, errMsg string) *protobufs.AgentToServer {
	return &protobufs.AgentToServer{
		InstanceUid: testUID,
		RemoteConfigStatus: &protobufs.RemoteConfigStatus{
			LastRemoteConfigHash: []byte{0xde, 0xad, 0xbe, 0xef},
			Status:               status,
			ErrorMessage:         errMsg,
		},
	}
}

func TestRemoteConfigStatusAppliedResolvesRollouts(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)

	s.onMessage(context.Background(), &fakeConn{}, rolloutStatusMsg(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED, ""))

	if len(store.resolves) != 1 {
		t.Fatalf("got %d resolves, want 1", len(store.resolves))
	}
	want := resolveCall{InstanceUID(testUID), "deadbeef", "applied", ""}
	if store.resolves[0] != want {
		t.Errorf("resolve = %+v, want %+v", store.resolves[0], want)
	}
	// applied acks carry no description, so the hash must be persisted here.
	wantHash := effHashCall{InstanceUID(testUID), "deadbeef"}
	if len(store.effHashes) != 1 || store.effHashes[0] != wantHash {
		t.Errorf("effective hashes = %+v, want [%+v]", store.effHashes, wantHash)
	}
}

func TestRemoteConfigStatusFailedResolvesRollouts(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer(store)

	s.onMessage(context.Background(), &fakeConn{}, rolloutStatusMsg(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED, "invalid yaml"))

	if len(store.resolves) != 1 {
		t.Fatalf("got %d resolves, want 1", len(store.resolves))
	}
	want := resolveCall{InstanceUID(testUID), "deadbeef", "failed", "invalid yaml"}
	if store.resolves[0] != want {
		t.Errorf("resolve = %+v, want %+v", store.resolves[0], want)
	}
	if len(store.effHashes) != 0 {
		t.Errorf("effective hashes = %+v, want none for a failed apply", store.effHashes)
	}
}

func TestRemoteConfigStatusIgnoredCases(t *testing.T) {
	cases := map[string]*protobufs.AgentToServer{
		"no status": {InstanceUid: testUID},
		"empty hash": {
			InstanceUid:        testUID,
			RemoteConfigStatus: &protobufs.RemoteConfigStatus{Status: protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED},
		},
		"applying": rolloutStatusMsg(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLYING, ""),
		"unset":    rolloutStatusMsg(protobufs.RemoteConfigStatuses_RemoteConfigStatuses_UNSET, ""),
	}
	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			store := &fakeStore{}
			s := newTestServer(store)

			s.onMessage(context.Background(), &fakeConn{}, msg)

			if len(store.resolves) != 0 {
				t.Errorf("resolves = %+v, want none", store.resolves)
			}
			if len(store.effHashes) != 0 {
				t.Errorf("effective hashes = %+v, want none", store.effHashes)
			}
		})
	}
}

func TestConnectedUIDsTracksLifecycle(t *testing.T) {
	s := newTestServer(&fakeStore{})
	conn := &fakeConn{}

	s.onMessage(context.Background(), conn, descMsg())
	if got := s.ConnectedUIDs(); len(got) != 1 || got[0] != InstanceUID(testUID) {
		t.Fatalf("ConnectedUIDs = %v, want [%s]", got, InstanceUID(testUID))
	}

	s.onConnectionClose(conn)
	if got := s.ConnectedUIDs(); len(got) != 0 {
		t.Errorf("ConnectedUIDs after close = %v, want empty", got)
	}
}

func TestSendConfigUnknownUIDErrors(t *testing.T) {
	s := newTestServer(&fakeStore{})

	err := s.SendConfig(context.Background(), "no-such-agent", []byte("x"), []byte{0x01})

	if err == nil {
		t.Fatal("SendConfig to unknown uid returned nil error")
	}
	if !strings.Contains(err.Error(), "no-such-agent") {
		t.Errorf("error %q does not mention the uid", err)
	}
}

func TestSendConfigDeliversRemoteConfig(t *testing.T) {
	s := newTestServer(&fakeStore{})
	conn := &fakeConn{}
	rawUID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	s.onMessage(context.Background(), conn, &protobufs.AgentToServer{InstanceUid: rawUID})

	yamlBody := []byte("receivers: {}\n")
	hash := []byte{0xab, 0xcd}
	if err := s.SendConfig(context.Background(), InstanceUID(rawUID), yamlBody, hash); err != nil {
		t.Fatalf("SendConfig: %v", err)
	}

	sent := conn.lastSent
	if sent == nil {
		t.Fatal("no message sent on the tracked connection")
	}
	if string(sent.InstanceUid) != string(rawUID) {
		t.Fatalf("instance uid = %x, want %x", sent.InstanceUid, rawUID)
	}
	rc := sent.GetRemoteConfig()
	if string(rc.GetConfigHash()) != string(hash) {
		t.Fatalf("config hash = %x, want %x", rc.GetConfigHash(), hash)
	}
	body := rc.GetConfig().GetConfigMap()[""].GetBody()
	if string(body) != string(yamlBody) {
		t.Fatalf("config body = %q, want %q", body, yamlBody)
	}
	if ct := rc.GetConfig().GetConfigMap()[""].GetContentType(); ct != "text/yaml" {
		t.Fatalf("content type = %q, want text/yaml", ct)
	}
}

func TestReconnectKeepsNewConnection(t *testing.T) {
	s := newTestServer(&fakeStore{})
	old, next := &fakeConn{}, &fakeConn{}

	s.onMessage(context.Background(), old, descMsg())
	s.onMessage(context.Background(), next, descMsg())
	// the stale connection closes after the agent already reconnected.
	s.onConnectionClose(old)

	if got := s.ConnectedUIDs(); len(got) != 1 || got[0] != InstanceUID(testUID) {
		t.Fatalf("ConnectedUIDs = %v, want [%s]", got, InstanceUID(testUID))
	}
	if err := s.SendConfig(context.Background(), InstanceUID(testUID), []byte("x"), []byte{0x01}); err != nil {
		t.Fatalf("SendConfig after reconnect: %v", err)
	}
	if old.lastSent != nil {
		t.Error("config sent to the stale connection")
	}
	if next.lastSent == nil {
		t.Error("config not sent to the live connection")
	}
}

func TestOnMessageAdvertisesCapabilities(t *testing.T) {
	s := newTestServer(&fakeStore{})

	resp := s.onMessage(context.Background(), &fakeConn{}, &protobufs.AgentToServer{InstanceUid: testUID})

	want := uint64(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus | protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig)
	if resp.GetCapabilities() != want {
		t.Errorf("capabilities = %d, want %d", resp.GetCapabilities(), want)
	}
}
