// Package events publishes checker state transitions to the Antenne event bus,
// wrapped in the enveloppe envelope every suite app shares. Fan-out to humans
// is Antenne's job; Sonde only reports what changed.
package events

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"

	antenne "github.com/FacileStudio/antenne-client/go"
	enveloppe "github.com/FacileStudio/enveloppe/go"
)

const (
	// ChannelDown carries a monitor that just went down, ChannelUp one that
	// recovered. Two channels rather than one so a delivery target can
	// subscribe to outages without also taking every recovery.
	ChannelDown = "monitor.down"
	ChannelUp   = "monitor.up"

	// DefaultApp is the pool identity Sonde registers under.
	DefaultApp = "sonde"

	statusUp   = "up"
	statusDown = "down"

	ownerLookupTimeout = 3 * time.Second

	// antenne-client registers over HTTP with http.DefaultClient, which has
	// no timeout, so a host that drops packets never returns. That call is
	// on the boot path ahead of the router, so an unbounded wait means the
	// API never listens and never answers /health.
	connectTimeout = 10 * time.Second

	// antenne-client stops reconnecting for good once its attempt budget is
	// spent: it reports "max reconnection attempts reached" and schedules
	// nothing further, with no way back short of a new client. Its default of
	// 20 is about 7.5 minutes of backoff, so a routine Antenne redeploy would
	// end event publishing for the life of this process. A monitor has to keep
	// reporting when the things around it restart, so the budget is the
	// practical equivalent of unbounded: backoff caps at 30s, which is a retry
	// every half minute for longer than any deployment will live.
	reconnectAttempts = math.MaxInt32
)

// Config is the Antenne connection, read from the environment. An empty
// Instance disables events entirely: no client is built and nothing dials.
type Config struct {
	Instance   string
	App        string
	Secret     string
	InstanceID string
}

// Enabled reports whether events are configured at all.
func (c Config) Enabled() bool { return c.Instance != "" }

// Owners resolves the email of the account a monitor belongs to.
type Owners interface {
	OwnerEmail(ctx context.Context, monitorID int64) (string, error)
}

// Payload is the contract's monitor object plus the actor the suite keys events
// on. A probe has no human behind it, so the monitor's owner is the actor of
// every transition it produces. enveloppe.Monitor carries no such field, so the
// publisher adds one instead of redefining the shared object.
type Payload struct {
	enveloppe.Monitor
	ActorEmail string `json:"actor_email,omitempty"`
}

type emitter interface {
	EmitNow(channel string, payload any) error
	Disconnect()
}

// Publisher turns checker transitions into events on the bus.
type Publisher struct {
	emitter emitter
	owners  Owners
	logger  *slog.Logger
}

// New dials Antenne and returns a publisher, or nil when events are not
// configured or the bus refuses the connection. A nil publisher is usable: Run
// returns immediately. Monitoring has to keep working when everything else is
// broken, so an unreachable bus is a warning and never a failed boot.
func New(ctx context.Context, config Config, owners Owners, logger *slog.Logger) *Publisher {
	if !config.Enabled() {
		return nil
	}

	app := config.App
	if app == "" {
		app = DefaultApp
	}

	client := antenne.NewClient(&antenne.Config{
		App:        app,
		Instance:   config.Instance,
		Secret:     config.Secret,
		InstanceID: config.InstanceID,
		Events:     antenne.EventConfig{Emit: []string{ChannelDown, ChannelUp}},
	},
		antenne.WithMaxReconnect(reconnectAttempts),
		antenne.WithOnError(func(err error) {
			logger.Warn("antenne connection", slog.Any("error", err))
		}))

	dial, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := client.Connect(dial); err != nil {
		client.Disconnect()
		logger.Warn("antenne unreachable, monitor events are off", slog.Any("error", err))
		return nil
	}

	logger.Info("publishing monitor events", slog.String("instance", config.Instance), slog.String("app", app))
	return &Publisher{emitter: client, owners: owners, logger: logger}
}

// Run consumes transitions until ctx ends or the channel closes. Emission
// failures are logged at warn and dropped, so a dead bus never stalls the
// checker behind it.
func (p *Publisher) Run(ctx context.Context, transitions <-chan checker.Transition) {
	if p == nil {
		return
	}
	defer p.emitter.Disconnect()

	for {
		select {
		case <-ctx.Done():
			return
		case transition, open := <-transitions:
			if !open {
				return
			}
			p.publish(ctx, transition)
		}
	}
}

// Publish emits a single transition to the bus.
func (p *Publisher) Publish(ctx context.Context, transition checker.Transition) {
	if p == nil {
		return
	}
	p.publish(ctx, transition)
}

func (p *Publisher) publish(ctx context.Context, transition checker.Transition) {
	channel, status := ChannelDown, statusDown
	if transition.Up {
		channel, status = ChannelUp, statusUp
	}

	payload := p.payload(ctx, transition, status)
	now := time.Now().UTC()
	// Keyed on the incident, not on the moment of the call. A wall clock in
	// the key is unique by construction and deduplicates nothing, which is
	// worse than no key at all because the field promises otherwise. The
	// outage and its recovery share one incident timestamp, so a redelivery
	// of either collapses onto the first.
	event := enveloppe.Event[Payload]{
		Version:        enveloppe.EventVersion,
		App:            enveloppe.AppSonde,
		Object:         enveloppe.ObjectMonitor,
		Action:         enveloppe.ActionUpdated,
		FacileID:       payload.FacileID,
		Payload:        payload,
		Timestamp:      now.Format(time.RFC3339),
		IdempotencyKey: payload.MonitorID + ":" + status + ":" + strconv.FormatInt(transition.IncidentAt.UnixMilli(), 10),
	}

	if err := p.emitter.EmitNow(channel, event); err != nil {
		p.logger.Warn("dropping monitor event",
			slog.Any("error", err),
			slog.String("channel", channel),
			slog.Int64("monitor_id", transition.MonitorID))
	}
}

func (p *Publisher) payload(ctx context.Context, transition checker.Transition, status string) Payload {
	var failure *string
	if transition.Error != "" {
		failure = &transition.Error
	}

	return Payload{
		Monitor: enveloppe.Monitor{
			MonitorID: strconv.FormatInt(transition.MonitorID, 10),
			FacileID:  transition.Slug,
			Slug:      transition.Slug,
			Name:      transition.Name,
			Status:    status,
			LatencyMS: transition.LatencyMS,
			Error:     failure,
		},
		ActorEmail: p.ownerEmail(ctx, transition.MonitorID),
	}
}

func (p *Publisher) ownerEmail(ctx context.Context, monitorID int64) string {
	if p.owners == nil {
		return ""
	}

	lookup, cancel := context.WithTimeout(ctx, ownerLookupTimeout)
	defer cancel()

	email, err := p.owners.OwnerEmail(lookup, monitorID)
	if err != nil {
		p.logger.Warn("resolving the monitor owner",
			slog.Any("error", err),
			slog.Int64("monitor_id", monitorID))
		return ""
	}
	return email
}
