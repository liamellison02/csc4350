// Package opamp wires an opamp-go server to persist collector agent state.
package opamp

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"
)

// dbTimeout bounds store writes made from connection-close callbacks,
// which carry no request context.
const dbTimeout = 5 * time.Second

// AgentStore is the persistence surface the server needs.
type AgentStore interface {
	UpsertAgent(ctx context.Context, uid, hostname, agentType, version, effectiveHash string) error
	MarkDisconnected(ctx context.Context, uid string) error
}

// Server adapts an opamp-go server to the helmsman store. it accepts every
// connection, upserts agents when they report a description, and marks them
// disconnected on close.
type Server struct {
	opamp  server.OpAMPServer
	store  AgentStore
	listen string
	log    *log.Logger

	// the opamp Connection carries no instance uid on close, so we remember
	// the uid learned from the first message keyed by connection identity.
	mu    sync.Mutex
	conns map[types.Connection]string
}

// NewServer builds a control-plane opamp server bound to store and listen.
func NewServer(store AgentStore, listen string, logger *log.Logger) *Server {
	return &Server{
		opamp:  server.New(opampLogger{logger}),
		store:  store,
		listen: listen,
		log:    logger,
		conns:  make(map[types.Connection]string),
	}
}

// Start begins accepting opamp connections; blocks until the listener is ready.
func (s *Server) Start() error {
	return s.opamp.Start(server.StartSettings{
		ListenEndpoint: s.listen,
		Settings: server.Settings{
			Callbacks: types.Callbacks{
				OnConnecting: s.onConnecting,
			},
		},
	})
}

// Stop closes the listener and all live connections.
func (s *Server) Stop(ctx context.Context) error {
	return s.opamp.Stop(ctx)
}

// onConnecting accepts every connection and attaches the per-connection
// message and close handlers.
func (s *Server) onConnecting(*http.Request) types.ConnectionResponse {
	return types.ConnectionResponse{
		Accept: true,
		ConnectionCallbacks: types.ConnectionCallbacks{
			OnMessage:         s.onMessage,
			OnConnectionClose: s.onConnectionClose,
		},
	}
}

// onMessage upserts the agent when it carries a description and echoes the
// instance uid back to the agent.
func (s *Server) onMessage(ctx context.Context, conn types.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	uid := InstanceUID(msg.GetInstanceUid())
	if uid != "" {
		s.track(conn, uid)
	}

	// a description arrives on first connect and whenever it changes; other
	// messages are health/config heartbeats that must not clobber the row.
	if desc := msg.GetAgentDescription(); desc != nil {
		hostname, agentType, version := Identify(desc)
		hash := effectiveConfigHash(msg)
		if err := s.store.UpsertAgent(ctx, uid, hostname, agentType, version, hash); err != nil {
			s.log.Printf("ERROR upsert agent %s: %v", uid, err)
		} else {
			s.log.Printf("agent upserted: uid=%s host=%s type=%s version=%s", uid, hostname, agentType, version)
		}
	}

	return &protobufs.ServerToAgent{InstanceUid: msg.GetInstanceUid()}
}

// onConnectionClose marks the agent disconnected if we know its uid.
func (s *Server) onConnectionClose(conn types.Connection) {
	uid := s.untrack(conn)
	if uid == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	if err := s.store.MarkDisconnected(ctx, uid); err != nil {
		s.log.Printf("ERROR mark agent %s disconnected: %v", uid, err)
		return
	}
	s.log.Printf("agent disconnected: uid=%s", uid)
}

func (s *Server) track(conn types.Connection, uid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[conn] = uid
}

func (s *Server) untrack(conn types.Connection) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	uid := s.conns[conn]
	delete(s.conns, conn)
	return uid
}

// effectiveConfigHash reads the hash of the last remote config the agent
// acknowledged, hex-encoded; empty when the agent has none yet.
func effectiveConfigHash(msg *protobufs.AgentToServer) string {
	if rcs := msg.GetRemoteConfigStatus(); rcs != nil {
		if h := rcs.GetLastRemoteConfigHash(); len(h) > 0 {
			return hex.EncodeToString(h)
		}
	}
	return ""
}

// opampLogger adapts *log.Logger to the opamp-go client/types.Logger interface.
type opampLogger struct {
	l *log.Logger
}

func (o opampLogger) Debugf(_ context.Context, format string, v ...any) {
	o.l.Printf("DEBUG "+format, v...)
}

func (o opampLogger) Errorf(_ context.Context, format string, v ...any) {
	o.l.Printf("ERROR "+format, v...)
}
