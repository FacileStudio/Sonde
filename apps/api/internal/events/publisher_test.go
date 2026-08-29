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

	publisher := New(Config{Secret: "shared"}, fakeOwners{}, quietLogger())
	if publisher != nil {
		t.Fatal("New returned a publisher without an instance")
	}

	// Every method has to be safe on the nil publisher, because that is the
	// value main holds on an unconfigured instance.
	publisher.Start(context.Background())
	publisher.Publish(context.Background(), checker.Transition{MonitorID: 1})
	publisher.Close()

	if state := publisher.State(); state.Configured || state.State != StateDisabled {
		t.Errorf("state = %+v, want an unconfigured disabled bus", state)
	}
	if pool.count() != 0 {
		t.Errorf("registrations = %d, want 0", pool.count())
	}
}

func TestAConfiguredInstanceIsDialed(t *testing.T) {
	pool := newFakePool(t)

	config := Config{Instance: pool.server.URL, Secret: "shared"}
	publisher := New(config, fakeOwners{}, quietLogger())
	if publisher == nil {
		t.Fatal("New returned no publisher for a reachable instance")
	}
	defer publisher.Close()

	// New must not dial: the boot path cannot wait on the bus.
	if pool.count() != 0 {
		t.Fatalf("New dialled %d times, want 0", pool.count())
	}

	publisher.Start(context.Background())
	waitFor(t, func() bool { return publisher.State().State == StateConnected })

	if pool.count() != 1 {
		t.Errorf("registrations = %d, want 1", pool.count())
	}
	state := publisher.State()
	if !state.Configured || state.Instance != config.Instance || state.App != DefaultApp {
		t.Errorf("state = %+v, want a configured bus on %s", state, config.Instance)
	}
}

// An unreachable bus must cost a warning, not a boot: the point of a monitor is
// that it keeps working when other things do not.
//
// It must also not cost the connection for the life of the process. Sonde and
// Antenne restart together on a suite deploy, and a publisher that gave up on
// its first failed dial meant whichever lost the race published nothing until
// somebody restarted it by hand.
func TestAnUnreachableBusIsRetriedRatherThanAbandoned(t *testing.T) {
	emitter := &fakeEmitter{dialErr: errors.New("connection refused")}
	publisher := newPublisher(emitter, fakeOwners{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	publisher.Start(ctx)

	waitFor(t, func() bool { return emitter.dialCount() >= 1 })

	state := publisher.State()
	if state.State != StateConnecting {
		t.Errorf("state = %q, want %q", state.State, StateConnecting)
	}
	if state.LastError == "" {
		t.Error("a refused dial left no last_error for the settings page to show")
	}
	if !state.Configured {
		t.Error("a refused dial reported the bus as unconfigured")
	}
}

// Losing an established connection is a different state from never having had
// one, and it has a different fix. Collapsing both into "not connected" is what
// the settings page must not do.
func TestALostConnectionReadsAsReconnecting(t *testing.T) {
	emitter := &fakeEmitter{}
	publisher := newPublisher(emitter, fakeOwners{})

	publisher.Start(context.Background())
	waitFor(t, func() bool { return publisher.State().State == StateConnected })

	emitter.Disconnect()
	if state := publisher.State().State; state != StateReconnecting {
		t.Errorf("state = %q, want %q", state, StateReconnecting)
	}
}

// The counters are the only evidence an operator has that events are leaving
// the box, so a dropped event has to be counted as dropped and not silently
// folded into the published total.
func TestPublishedAndDroppedAreCountedSeparately(t *testing.T) {
	emitter := &fakeEmitter{}
	publisher := newPublisher(emitter, fakeOwners{})

	publisher.publish(context.Background(), checker.Transition{MonitorID: 1, Slug: "web", Name: "Web"})
	emitter.err = errors.New("not connected")
	publisher.publish(context.Background(), checker.Transition{MonitorID: 1, Slug: "web", Name: "Web"})

	state := publisher.State()
	if state.Published != 1 {
		t.Errorf("published = %d, want 1", state.Published)
	}
	if state.Dropped != 1 {
		t.Errorf("dropped = %d, want 1", state.Dropped)
	}
	if state.LastEventAt == nil {
		t.Error("last_event_at is nil after a successful emit")
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

// The key has to identify the incident, not the instant the publisher happened
// to run: a key that is unique per call deduplicates nothing while the field
// promises it does. Both ends of one outage carry it, and a later outage of the
// same monitor gets its own.
func TestTheIdempotencyKeyIdentifiesTheIncident(t *testing.T) {
	emitter := &fakeEmitter{}
	publisher := newPublisher(emitter, fakeOwners{})

	outage := time.Now().Add(-time.Hour)
	down := checker.Transition{MonitorID: 7, Slug: "web", Name: "Web", IncidentAt: outage}
	up := checker.Transition{MonitorID: 7, Slug: "web", Name: "Web", Up: true, IncidentAt: outage}
	later := checker.Transition{MonitorID: 7, Slug: "web", Name: "Web", IncidentAt: time.Now()}

	publisher.publish(context.Background(), down)
	publisher.publish(context.Background(), down)
	publisher.publish(context.Background(), up)
	publisher.publish(context.Background(), later)

	keys := make([]string, 0, 4)
	for _, payload := range emitter.payloads {
		keys = append(keys, decodeEvent(t, payload).IdempotencyKey)
	}

	if keys[0] != keys[1] {
		t.Errorf("a redelivered outage changed key: %q then %q", keys[0], keys[1])
	}
	if keys[2] == keys[0] {
		t.Errorf("the recovery reused the outage key %q", keys[0])
	}
	if keys[3] == keys[0] {
		t.Errorf("a second outage reused the first outage's key %q", keys[0])
	}
}

// A failing emit, or an owner lookup that errors, must not stop the caller:
// the checker keeps producing transitions and nothing may back up behind them.
func TestAFailingEmitDoesNotStallTheCaller(t *testing.T) {
	emitter := &fakeEmitter{err: errors.New("not connected")}
	publisher := newPublisher(emitter, fakeOwners{err: errors.New("no such monitor")})

	done := make(chan struct{})
	go func() {
		for i := range 5 {
			publisher.Publish(context.Background(), checker.Transition{
				MonitorID: int64(i), Slug: "web", Name: "Web",
			})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publishing stalled on a dead bus")
	}
	if emitter.count() != 5 {
		t.Errorf("attempted %d emits, want 5", emitter.count())
	}
}

// waitFor polls a condition rather than sleeping a guessed interval, so the
// test is neither flaky on a loaded machine nor slower than it has to be.
func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met within 2s")
}
