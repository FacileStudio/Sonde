package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
)

// The event contract keys on an actor email and a probe has no human behind it,
// so the owner join is what stands between a monitor event and an unattributed
// one. It runs against the real schema because it is a join, not a rule.
func TestOwnerEmailResolvesTheAccountBehindAMonitor(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, router)

	created := call(t, router, http.MethodPost, "/api/monitors", csrfHeaders(session.Token), map[string]any{
		"slug": "website", "name": "Website", "type": "http", "target": "https://facile.studio",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body)
	}
	var monitor struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &monitor); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	store := monitors.NewStore(db)
	email, err := store.OwnerEmail(context.Background(), monitor.ID)
	if err != nil {
		t.Fatalf("OwnerEmail: %v", err)
	}
	if email != session.User.Email {
		t.Errorf("OwnerEmail = %q, want %q", email, session.User.Email)
	}

	missing, err := store.OwnerEmail(context.Background(), monitor.ID+1000)
	if err != nil {
		t.Fatalf("OwnerEmail on an unknown monitor: %v", err)
	}
	if missing != "" {
		t.Errorf("OwnerEmail on an unknown monitor = %q, want empty", missing)
	}
}
