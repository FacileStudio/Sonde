package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"

	"gorm.io/gorm"
)

// The status pages and the heartbeat endpoint are public by design: a visitor
// reads a status page and a cron job pushes a heartbeat, and neither has a
// session or a CSRF header. Sign-in through Registre must not change that.
func TestPublicRoutesStayReachableWithoutASession(t *testing.T) {
	router, db := liveRouter(t, configureIssuer)
	owner := signIn(t, db, "owner@facile.studio")

	created := call(t, router, http.MethodPost, "/api/monitors", csrfHeaders(owner.Token), map[string]any{
		"slug": "cron-job", "name": "Cron job", "type": "push",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body)
	}
	var monitor struct {
		ID        int64  `json:"id"`
		PushToken string `json:"push_token"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &monitor); err != nil || monitor.PushToken == "" {
		t.Fatalf("push creation: %s (%v)", created.Body, err)
	}
	page := call(t, router, http.MethodPost, "/api/status-pages", csrfHeaders(owner.Token), map[string]any{
		"slug": "public-page", "title": "Public page", "monitor_ids": []int64{monitor.ID},
	})
	if page.Code != http.StatusCreated {
		t.Fatalf("status page create = %d: %s", page.Code, page.Body)
	}

	if status := call(t, router, http.MethodGet, "/api/public/status/public-page", nil, nil); status.Code != http.StatusOK {
		t.Errorf("public status page = %d, want 200: %s", status.Code, status.Body)
	}
	if missing := call(t, router, http.MethodGet, "/api/public/status/nope", nil, nil); missing.Code != http.StatusNotFound {
		t.Errorf("unknown status page = %d, want 404", missing.Code)
	}
	for _, path := range []string{"/health", "/version"} {
		if probe := call(t, router, http.MethodGet, path, nil, nil); probe.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, probe.Code)
		}
	}

	assertHeartbeatNeedsNoSession(t, db, monitor.PushToken)
}

// assertHeartbeatNeedsNoSession runs a real engine so the push token is
// registered, then pushes with no cookie and no CSRF header.
func assertHeartbeatNeedsNoSession(t *testing.T, db *gorm.DB, token string) {
	t.Helper()
	engine := checker.NewGorm(db, checker.WithoutTransitions(), checker.WithLogger(slog.New(slog.DiscardHandler)))
	router, err := buildRouter(routerDeps{DB: db, SQL: rawSQL(t, db), Engine: engine, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("buildRouter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = engine.Run(ctx)
	}()
	defer func() {
		cancel()
		<-done
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		response := call(t, router, http.MethodPost, "/api/push/"+token, nil, nil)
		if response.Code == http.StatusOK {
			return
		}
		if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden {
			t.Fatalf("the heartbeat demanded a session: %d", response.Code)
		}
		if time.Now().After(deadline) {
			t.Fatalf("heartbeat = %d, want 200: %s", response.Code, response.Body)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
