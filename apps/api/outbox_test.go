package main

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/antenne-client/go/outbox"
	"gorm.io/gorm"
)

type failingEvents struct{}

func (failingEvents) Enqueue(context.Context, *sql.Tx, checker.Transition) error {
	return stderrors.New("the queue refused")
}

type queuedEvent struct {
	Channel string
	Key     string
	Actor   string
	Status  string
}

func outboxMonitor(t *testing.T) (*gorm.DB, int64, string) {
	t.Helper()
	router, db := liveAPI(t)
	session := adminSession(t, db)
	created := call(t, router, http.MethodPost, "/api/monitors", csrfHeaders(session.Token), map[string]any{
		"slug": "website", "name": "Website", "type": "http", "target": "https://facile.studio",
	})
	var monitor struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &monitor); err != nil || created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body)
	}
	return db, monitor.ID, session.User.Email
}

func queued(t *testing.T, db *gorm.DB) []queuedEvent {
	t.Helper()
	var out []queuedEvent
	err := db.Raw(`SELECT channel, idempotency_key AS key, payload->'payload'->>'actor_email' AS actor,
		payload->'payload'->>'status' AS status FROM antenne_outbox ORDER BY id`).Scan(&out).Error
	if err != nil {
		t.Fatalf("read the outbox: %v", err)
	}
	return out
}

func countOf(t *testing.T, db *gorm.DB, query string) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query).Scan(&n).Error; err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// The migration is the library's, copied under Sonde's next number. A copy
// that drifts from the library is a schema the drainer was never tested on.
func TestTheOutboxMigrationIsTheLibrarysVerbatim(t *testing.T) {
	ours, err := os.ReadFile("migrations/00008_antenne_outbox.sql")
	if err != nil {
		t.Fatalf("read the migration: %v", err)
	}
	if string(ours) != outbox.Migration {
		t.Fatal("migrations/00008_antenne_outbox.sql differs from outbox.Migration")
	}
}

// An outage and its recovery are queued in the transaction that writes the
// incident, under one incident key, and attributed to the monitor's owner.
func TestAnIncidentQueuesItsEventsInTheSameTransaction(t *testing.T) {
	db, monitorID, owner := outboxMonitor(t)
	recorder := checker.NewGormRecorder(db, events.Outbox{})
	ctx := context.Background()
	at := time.UnixMilli(1_700_000_000_000)

	down := checker.Transition{MonitorID: monitorID, Slug: "website", Name: "Website", Error: "timeout", IncidentAt: at}
	if err := recorder.OpenIncident(ctx, down, "timeout"); err != nil {
		t.Fatalf("OpenIncident: %v", err)
	}
	if err := recorder.OpenIncident(ctx, down, "timeout"); err != nil {
		t.Fatalf("a second queueing of the same incident must be a no-op: %v", err)
	}
	up := down
	up.Up, up.Error = true, ""
	if err := recorder.ResolveIncidents(ctx, monitorID, &up); err != nil {
		t.Fatalf("ResolveIncidents: %v", err)
	}

	got := queued(t, db)
	if len(got) != 2 || got[0].Channel != "monitor.down" || got[1].Channel != "monitor.up" {
		t.Fatalf("queued = %+v, want monitor.down then monitor.up", got)
	}
	wantKey := strconv.FormatInt(monitorID, 10) + ":down:1700000000000"
	if got[0].Actor != owner || got[0].Status != "down" || got[0].Key != wantKey {
		t.Errorf("down event = %+v, want actor %s, status down, key %s", got[0], owner, wantKey)
	}
}

// A refused enqueue rolls the incident back with it: the event and the state
// change commit together or not at all.
func TestARefusedEnqueueRollsTheIncidentBack(t *testing.T) {
	db, monitorID, _ := outboxMonitor(t)
	recorder := checker.NewGormRecorder(db, failingEvents{})

	down := checker.Transition{MonitorID: monitorID, IncidentAt: time.Now()}
	if err := recorder.OpenIncident(context.Background(), down, "boom"); err == nil {
		t.Fatal("OpenIncident succeeded although the queue refused")
	}
	if n := countOf(t, db, `SELECT count(*) FROM incidents`); n != 0 {
		t.Errorf("incidents = %d, want 0 after a refused enqueue", n)
	}
}

// Without a bus nothing is queued, so an unconfigured instance never fills a
// table nobody drains.
func TestAnUnconfiguredBusQueuesNothing(t *testing.T) {
	db, monitorID, _ := outboxMonitor(t)
	var publisher *events.Publisher
	recorder := checker.NewGormRecorder(db, publisher.Queue())

	if err := recorder.OpenIncident(context.Background(), checker.Transition{MonitorID: monitorID, IncidentAt: time.Now()}, "boom"); err != nil {
		t.Fatalf("OpenIncident: %v", err)
	}
	if n := countOf(t, db, `SELECT count(*) FROM antenne_outbox`); n != 0 {
		t.Errorf("outbox rows = %d, want 0 without a bus", n)
	}
	if n := countOf(t, db, `SELECT count(*) FROM incidents`); n != 1 {
		t.Errorf("incidents = %d, want 1", n)
	}
}
