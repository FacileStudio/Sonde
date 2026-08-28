package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
)

// An unconfigured instance is the production default, so "off" has to mean no
// client, no dial and no error rather than a client pointing at nothing.
func TestNoInstanceMeansNothingDials(t *testing.T) {
	pool := newFakePool(t)

	publisher := New(context.Background(), Config{Secret: "shared"}, fakeOwners{}, quietLogger())
	if publisher != nil {
		t.Fatal("New returned a publisher without an instance")
	}
	if pool.count() != 0 {
		t.Errorf("registrations = %d, want 0", pool.count())
	}

	transitions := make(chan checker.Transition, 1)
	transitions <- checker.Transition{MonitorID: 1}
	done := make(chan struct{})
	go func() {
		publisher.Run(context.Background(), transitions)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run on a disabled publisher did not return")
	}
}

func TestAConfiguredInstanceIsDialed(t *testing.T) {
	pool := newFakePool(t)

	config := Config{Instance: pool.server.URL, Secret: "shared"}
	publisher := New(context.Background(), config, fakeOwners{}, quietLogger())
	if publisher == nil {
		t.Fatal("New returned no publisher for a reachable instance")
	}
	defer publisher.emitter.Disconnect()

	if pool.count() != 1 {
		t.Errorf("registrations = %d, want 1", pool.count())
	}
}

// An unreachable bus must cost a warning, not a boot: the point of a monitor is
// that it keeps working when other things do not.
func TestAnUnreachableBusDisablesEventsWithoutFailing(t *testing.T) {
	pool := newFakePool(t)
	address := pool.server.URL
	pool.server.Close()

	config := Config{Instance: address, Secret: "shared"}
	if publisher := New(context.Background(), config, fakeOwners{}, quietLogger()); publisher != nil {
		t.Fatal("New returned a publisher for an unreachable instance")
	}
}

func TestATransitionCarriesTheMonitorPayloadAndItsOwner(t *testing.T) {
	emitter := &fakeEmitter{}
	publisher := newPublisher(emitter, fakeOwners{email: "owner@facile.studio"})

	publisher.publish(context.Background(), checker.Transition{
		MonitorID: 7,
		Slug:      "web",
		Name:      "Web",
		Up:        false,
		LatencyMS: 0,
		Error:     "connection refused",
	})

	if emitter.count() != 1 {
		t.Fatalf("emitted %d events, want 1", emitter.count())
	}
	if emitter.channels[0] != ChannelDown {
		t.Errorf("channel = %q, want %q", emitter.channels[0], ChannelDown)
	}

	event := decodeEvent(t, emitter.payloads[0])

	switch {
	case event.Version != 1:
		t.Errorf("version = %d, want 1", event.Version)
	case event.App != "Sonde":
		t.Errorf("app = %q, want Sonde", event.App)
	case event.Object != "monitor":
		t.Errorf("object = %q, want monitor", event.Object)
	case event.Action != "updated":
		t.Errorf("action = %q, want updated", event.Action)
	case event.Payload.MonitorID != "7":
		t.Errorf("monitor_id = %q, want 7", event.Payload.MonitorID)
	case event.Payload.FacileID != "web" || event.Payload.Slug != "web":
		t.Errorf("facile_id/slug = %q/%q, want web/web", event.Payload.FacileID, event.Payload.Slug)
	case event.Payload.Name != "Web":
		t.Errorf("name = %q, want Web", event.Payload.Name)
	case event.Payload.Status != statusDown:
		t.Errorf("status = %q, want %q", event.Payload.Status, statusDown)
	case event.Payload.Error == nil || *event.Payload.Error != "connection refused":
		t.Errorf("error = %v, want connection refused", event.Payload.Error)
	case event.Payload.ActorEmail != "owner@facile.studio":
		t.Errorf("actor_email = %q, want owner@facile.studio", event.Payload.ActorEmail)
	case event.IdempotencyKey == "":
		t.Error("idempotency_key is empty")
	}
}

func TestARecoveryIsNullErrorOnTheUpChannel(t *testing.T) {
	emitter := &fakeEmitter{}
	publisher := newPublisher(emitter, fakeOwners{email: "owner@facile.studio"})

	publisher.publish(context.Background(), checker.Transition{
		MonitorID: 7, Slug: "web", Name: "Web", Up: true, LatencyMS: 42,
	})

	if emitter.channels[0] != ChannelUp {
		t.Fatalf("channel = %q, want %q", emitter.channels[0], ChannelUp)
	}

	event := decodeEvent(t, emitter.payloads[0])
	switch {
	case event.Payload.Status != statusUp:
		t.Errorf("status = %q, want %q", event.Payload.Status, statusUp)
	case event.Payload.Error != nil:
		t.Errorf("error = %q, want null", *event.Payload.Error)
	case event.Payload.LatencyMS != 42:
		t.Errorf("latency_ms = %d, want 42", event.Payload.LatencyMS)
	}
}

// A failing emit, or an owner lookup that errors, must not stop the consumer:
// the checker keeps producing transitions and nothing may back up behind them.
func TestAFailingEmitDoesNotStallTheConsumer(t *testing.T) {
	emitter := &fakeEmitter{err: errors.New("not connected")}
	publisher := newPublisher(emitter, fakeOwners{err: errors.New("no such monitor")})

	transitions := make(chan checker.Transition)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		publisher.Run(ctx, transitions)
		close(done)
	}()

	for i := range 5 {
		select {
		case transitions <- checker.Transition{MonitorID: int64(i), Slug: "web", Name: "Web"}:
		case <-time.After(time.Second):
			t.Fatalf("the consumer stalled on transition %d", i)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return when the context was cancelled")
	}
	if emitter.count() != 5 {
		t.Errorf("attempted %d emits, want 5", emitter.count())
	}
}
