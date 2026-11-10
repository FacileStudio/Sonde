package events

import (
	"context"
	"database/sql"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/antenne-client/go/catalog"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"github.com/FacileStudio/antenne-client/go/outbox"
)

const (
	statusUp   = "up"
	statusDown = "down"
)

// Payload is the contract's monitor object plus the actor the suite keys events
// on. A probe has no human behind it, so the monitor's owner is the actor of
// every transition it produces. envelope.Monitor carries no such field, so
// Sonde adds one instead of redefining the shared object.
type Payload struct {
	envelope.Monitor
	ActorEmail string `json:"actor_email,omitempty"`
}

// Outbox queues transitions in antenne_outbox. It satisfies checker.Events.
type Outbox struct{}

// Queue is where the recorder queues transitions: the outbox when the bus is
// configured, nothing on a nil publisher, so an unconfigured instance never
// fills a table nobody drains.
func (p *Publisher) Queue() checker.Events {
	if p == nil {
		return nil
	}
	return Outbox{}
}

// Enqueue queues one transition inside tx, the transaction that writes the
// incident. The idempotency key names the incident, not the moment: the outage
// and its recovery share one incident timestamp, so a second queueing of either
// is a no-op rather than a duplicate on the bus.
func (Outbox) Enqueue(ctx context.Context, tx *sql.Tx, transition checker.Transition) error {
	channel, status := catalog.MonitorDown, statusDown
	if transition.Up {
		channel, status = catalog.MonitorUp, statusUp
	}
	owner, err := ownerEmail(ctx, tx, transition.MonitorID)
	if err != nil {
		return err
	}
	event := Event(transition, status, owner, time.Now().UTC())
	err = outbox.Enqueue(ctx, tx, channel, event, event.IdempotencyKey)
	if stderrors.Is(err, outbox.ErrDuplicate) {
		return nil
	}
	return err
}

// Event builds the envelope for one transition.
func Event(transition checker.Transition, status, owner string, now time.Time) envelope.Event[Payload] {
	var failure *string
	if transition.Error != "" {
		failure = &transition.Error
	}
	monitorID := strconv.FormatInt(transition.MonitorID, 10)
	return envelope.Event[Payload]{
		Version:  envelope.EventVersion,
		App:      envelope.AppSonde,
		Object:   envelope.ObjectMonitor,
		Action:   envelope.ActionUpdated,
		FacileID: transition.Slug,
		Payload: Payload{
			Monitor: envelope.Monitor{
				MonitorID: monitorID, FacileID: transition.Slug, Slug: transition.Slug, Name: transition.Name,
				Status: status, LatencyMS: transition.LatencyMS, Error: failure,
			},
			ActorEmail: owner,
		},
		Timestamp:      now.Format(time.RFC3339),
		IdempotencyKey: monitorID + ":" + status + ":" + strconv.FormatInt(transition.IncidentAt.UnixMilli(), 10),
	}
}

func ownerEmail(ctx context.Context, tx *sql.Tx, monitorID int64) (string, error) {
	var email string
	err := tx.QueryRowContext(ctx,
		`SELECT users.email FROM monitors JOIN users ON users.id = monitors.owner_id WHERE monitors.id = $1`,
		monitorID).Scan(&email)
	if stderrors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return email, err
}
