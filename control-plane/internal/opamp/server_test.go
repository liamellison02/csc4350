package opamp

import (
	"context"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
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
}

type upsertCall struct {
	uid, hostname, agentType, version, hash string
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

// fakeConn is a comparable no-op Connection usable as a map key.
type fakeConn struct{}

func (*fakeConn) Connection() net.Conn                                 { return nil }
func (*fakeConn) Send(context.Context, *protobufs.ServerToAgent) error { return nil }
func (*fakeConn) Disconnect() error                                    { return nil }

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
	}

	s.onMessage(context.Background(), &fakeConn{}, msg)

	if len(store.upserts) != 1 {
		t.Fatalf("got %d upserts, want 1", len(store.upserts))
	}
	if store.upserts[0].hash != "deadbeef" {
		t.Errorf("hash = %q, want %q", store.upserts[0].hash, "deadbeef")
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
