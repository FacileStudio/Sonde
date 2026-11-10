package events

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/antenne-client/go/catalog"
	"github.com/FacileStudio/antenne-client/go/outbox"
)

// An unconfigured bus is the default, so "off" has to mean no client, no
// queue and no error, and every method has to be safe on the nil publisher.
func TestANilConfigMeansNoPublisherAndNoQueue(t *testing.T) {
	publisher := New(nil, nil, nil)
	if publisher != nil {
		t.Fatal("New returned a publisher without a config")
	}
	publisher.Start(context.Background())
	publisher.Close()
	if publisher.Queue() != nil {
		t.Error("a nil publisher hands the recorder a queue")
	}
	if state := publisher.State(context.Background()); state.Configured || state.State != StateDisabled {
		t.Errorf("state = %+v, want an unconfigured disabled bus", state)
	}
}

func TestTheEventCarriesTheMonitorItsOwnerAndTheIncidentKey(t *testing.T) {
	at := time.UnixMilli(1_700_000_000_000)
	down := Event(checker.Transition{MonitorID: 7, Slug: "web", Name: "Web", LatencyMS: 12, Error: "timeout", IncidentAt: at}, statusDown, "owner@facile.studio", at)
	up := Event(checker.Transition{MonitorID: 7, Slug: "web", Name: "Web", Up: true, IncidentAt: at}, statusUp, "", at)

	raw, err := json.Marshal(down)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"version":1,"app":"Sonde","object":"monitor","action":"updated","facile_id":"web","payload":{"monitor_id":"7","facile_id":"web","slug":"web","name":"Web","status":"down","latency_ms":12,"error":"timeout","actor_email":"owner@facile.studio"},"timestamp":"` + at.Format(time.RFC3339) + `","idempotency_key":"7:down:1700000000000"}`
	if string(raw) != want {
		t.Errorf("down event =\n%s\nwant\n%s", raw, want)
	}
	if up.Payload.Error != nil || up.IdempotencyKey != "7:up:1700000000000" {
		t.Errorf("recovery = %+v, want a null error and the incident key", up)
	}
}

func TestTheDrainerSendsTheQueueInOrderAndCounts(t *testing.T) {
	emitter := &fakeEmitter{connected: true}
	publisher := testPublisher(t, emitter)
	ctx := context.Background()
	enqueue(t, publisher, catalog.MonitorDown, "a")
	enqueue(t, publisher, catalog.MonitorUp, "b")

	if state := publisher.State(ctx); state.Pending != 2 {
		t.Fatalf("pending = %d before draining, want 2", state.Pending)
	}
	if _, err := publisher.drainer.Drain(ctx); err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if got := emitter.sent(); !slices.Equal(got, []string{"monitor.down", "monitor.up"}) {
		t.Errorf("sent = %v, want down then up", got)
	}
	state := publisher.State(ctx)
	if state.Pending != 0 || state.Published != 2 || state.LastEventAt == nil {
		t.Errorf("state = %+v, want 0 pending and 2 published", state)
	}
}

// A bus that refuses keeps the event queued and counts nothing as published.
func TestARefusedEmitLeavesTheEventQueued(t *testing.T) {
	emitter := &fakeEmitter{emitErr: errors.New("not connected")}
	publisher := testPublisher(t, emitter)
	enqueue(t, publisher, catalog.MonitorDown, "a")

	if _, err := publisher.drainer.Drain(context.Background()); err == nil {
		t.Fatal("Drain succeeded although the emit failed")
	}
	if state := publisher.State(context.Background()); state.Pending != 1 || state.Published != 0 {
		t.Errorf("state = %+v, want the event still pending", state)
	}
}

// Sonde and Antenne restart together on a suite deploy, so a refused first dial
// is retried and reads as connecting, and a lost connection reads as
// reconnecting, which has a different fix.
func TestTheDialIsRetriedAndTheStatesStayDistinct(t *testing.T) {
	emitter := &fakeEmitter{dialErr: errors.New("connection refused")}
	publisher := testPublisher(t, emitter)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go publisher.dial(ctx)

	waitFor(t, func() bool { return publisher.State(ctx).LastError != "" })
	if state := publisher.State(ctx); state.State != StateConnecting || !state.Configured {
		t.Errorf("state = %+v, want a configured bus still connecting", state)
	}

	cancel()
	publisher.recordDial(nil)
	emitter.mu.Lock()
	emitter.connected = true
	emitter.mu.Unlock()
	if state := publisher.State(ctx).State; state != StateConnected {
		t.Errorf("state = %q, want connected", state)
	}
	emitter.Disconnect()
	if state := publisher.State(ctx).State; state != StateReconnecting {
		t.Errorf("state = %q, want reconnecting", state)
	}
}

func enqueue(t *testing.T, publisher *Publisher, channel catalog.Channel, key string) {
	t.Helper()
	tx, err := publisher.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := outbox.Enqueue(context.Background(), tx, channel, map[string]string{"k": key}, key); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}
