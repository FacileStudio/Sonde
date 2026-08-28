package events

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

type fakeOwners struct {
	email string
	err   error
}

func (o fakeOwners) OwnerEmail(context.Context, int64) (string, error) { return o.email, o.err }

type fakeEmitter struct {
	mu       sync.Mutex
	err      error
	channels []string
	payloads []any
}

func (e *fakeEmitter) EmitNow(channel string, payload any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.channels = append(e.channels, channel)
	e.payloads = append(e.payloads, payload)
	return e.err
}

func (e *fakeEmitter) Disconnect() {}

func (e *fakeEmitter) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.channels)
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakePool is an Antenne instance that counts registrations, so a test can tell
// "the publisher dialed" from "the publisher did nothing".
type fakePool struct {
	server        *httptest.Server
	mu            sync.Mutex
	registrations int
}

func newFakePool(t *testing.T) *fakePool {
	t.Helper()
	pool := &fakePool{}
	upgrader := websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/antenne/register", func(w http.ResponseWriter, _ *http.Request) {
		pool.mu.Lock()
		pool.registrations++
		pool.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "token", "app_id": "sonde"})
	})
	mux.HandleFunc("/api/antenne/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	pool.server = httptest.NewServer(mux)
	t.Cleanup(pool.server.Close)
	return pool
}

func (p *fakePool) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.registrations
}

func newPublisher(emitter emitter, owners Owners) *Publisher {
	return &Publisher{emitter: emitter, owners: owners, logger: quietLogger()}
}

// wireEvent is the emitted envelope as a consumer in another language sees it:
// decoded from JSON by key, never from the Go types that produced it.
type wireEvent struct {
	Version int    `json:"version"`
	App     string `json:"app"`
	Object  string `json:"object"`
	Action  string `json:"action"`
	Payload struct {
		MonitorID  string  `json:"monitor_id"`
		FacileID   string  `json:"facile_id"`
		Slug       string  `json:"slug"`
		Name       string  `json:"name"`
		Status     string  `json:"status"`
		LatencyMS  int64   `json:"latency_ms"`
		Error      *string `json:"error"`
		ActorEmail string  `json:"actor_email"`
	} `json:"payload"`
	IdempotencyKey string `json:"idempotency_key"`
}

func decodeEvent(t *testing.T, payload any) wireEvent {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshalling the event: %v", err)
	}
	var event wireEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("the event is not JSON: %v", err)
	}
	return event
}
