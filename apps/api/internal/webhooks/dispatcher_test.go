package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
)

type mockWebhookStore struct {
	webhooks []schemas.Webhook
}

func (m *mockWebhookStore) ListEnabledForOwner(ctx context.Context, ownerID int64) ([]schemas.Webhook, error) {
	return m.webhooks, nil
}

type mockMonitorStore struct {
	ownerID int64
	email   string
}

func (m *mockMonitorStore) OwnerID(ctx context.Context, monitorID int64) (int64, error) {
	return m.ownerID, nil
}

func (m *mockMonitorStore) OwnerEmail(ctx context.Context, monitorID int64) (string, error) {
	return m.email, nil
}

func TestDispatcherDeliversSignedTransition(t *testing.T) {
	var (
		mu         sync.Mutex
		received   envelope.Event[Payload]
		authHeader string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		authHeader = r.Header.Get("X-Custom-Signature")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secret := "test-secret-123"
	webhooks := &mockWebhookStore{
		webhooks: []schemas.Webhook{
			{
				ID:           1,
				OwnerID:      42,
				URL:          server.URL,
				Secret:       secret,
				SecretHeader: "X-Custom-Signature",
				Enabled:      true,
			},
		},
	}
	monitors := &mockMonitorStore{ownerID: 42, email: "admin@sonde.local"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispatcher := NewDispatcher(webhooks, monitors, logger)

	transition := checker.Transition{
		MonitorID:  99,
		Slug:       "api-gateway",
		Name:       "API Gateway",
		Up:         false,
		LatencyMS:  120,
		Error:      "status 502 Bad Gateway",
		IncidentAt: time.Date(2026, 8, 29, 4, 0, 0, 0, time.UTC),
	}

	dispatcher.Dispatch(context.Background(), transition)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if received.Payload.MonitorID != "99" {
		t.Fatalf("expected monitor ID 99, got %s", received.Payload.MonitorID)
	}
	if received.Payload.Status != "down" {
		t.Errorf("expected status down, got %s", received.Payload.Status)
	}
	if received.Payload.ActorEmail != "admin@sonde.local" {
		t.Errorf("expected actor email admin@sonde.local, got %s", received.Payload.ActorEmail)
	}
	if authHeader == "" {
		t.Error("expected HMAC signature header, got empty")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	rawBody, _ := json.Marshal(received)
	mac.Write(rawBody)
	// The prefix is the half that was missing. Antenne compares the whole
	// header against "sha256=" + digest, so a test that checked the digest
	// alone passed while every signed delivery to Antenne was refused 401.
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if authHeader != expectedSig {
		t.Errorf("signature header %q did not match expected %q", authHeader, expectedSig)
	}
}

func TestDispatcherTestEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispatcher := NewDispatcher(nil, nil, logger)

	result, err := dispatcher.Test(context.Background(), schemas.Webhook{
		URL: server.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Errorf("expected OK true, got false (%s)", result.Error)
	}
	if result.StatusCode != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", result.StatusCode)
	}
}
