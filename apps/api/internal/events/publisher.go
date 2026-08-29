// Package events publishes checker state transitions to the Antenne event bus,
// wrapped in the enveloppe envelope every suite app shares. Fan-out to humans
// is Antenne's job; Sonde only reports what changed.
package events

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"sync"
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
	// no timeout, so a host that drops packets never returns.
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

	// That budget only covers reconnection, which the library starts counting
	// after a first successful dial. The first dial is ours to retry, and it
	// is the one that matters most: Sonde and Antenne restart together on a
	// suite deploy, so whoever loses the race used to lose events until
	// somebody noticed and restarted Sonde by hand.
	dialBackoffMin = 5 * time.Second
	dialBackoffMax = 60 * time.Second
)

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
	Connect(ctx context.Context) error
	EmitNow(channel string, payload any) error
	Disconnect()
	IsConnected() bool
}

// Publisher turns checker transitions into events on the bus.
type Publisher struct {
	config  Config
	emitter emitter
	owners  Owners
	logger  *slog.Logger

	mu          sync.Mutex
	dialed      bool
	published   int64
	dropped     int64
	lastEventAt *time.Time
	lastError   string
}

// New returns a publisher, or nil when events are not configured. A nil
// publisher is usable: every method on it is a no-op and State reports
// disabled.
//
// New does not dial. Dialing is Start's job, off the boot path, because a bus
// that is slow or down must not hold up the server that reports on it.
func New(config Config, owners Owners, logger *slog.Logger) *Publisher {
	if !config.Enabled() {
		return nil
	}

	client := antenne.NewClient(&antenne.Config{
		App:        config.AppName(),
		Instance:   config.Instance,
		Secret:     config.Secret,
		InstanceID: config.InstanceID,
		Events:     antenne.EventConfig{Emit: []string{ChannelDown, ChannelUp}},
	},
		antenne.WithMaxReconnect(reconnectAttempts),
		antenne.WithOnError(func(err error) {
			logger.Warn("antenne connection", slog.Any("error", err))
		}))

	return &Publisher{config: config, emitter: client, owners: owners, logger: logger}
}

// Start dials the bus in the background and keeps retrying until it answers or
// ctx ends. It returns immediately.
func (p *Publisher) Start(ctx context.Context) {
	if p == nil {
		return
	}
	go p.dial(ctx)
}

func (p *Publisher) dial(ctx context.Context) {
	backoff := dialBackoffMin
	for {
		attempt, cancel := context.WithTimeout(ctx, connectTimeout)
		err := p.emitter.Connect(attempt)
		cancel()
		if err == nil {
			p.mu.Lock()
			p.dialed = true
			p.lastError = ""
			p.mu.Unlock()
			p.logger.Info("publishing monitor events",
				slog.String("instance", p.config.Instance),
				slog.String("app", p.config.AppName()))
			return
		}

		p.mu.Lock()
		p.lastError = err.Error()
		p.mu.Unlock()
		p.logger.Warn("antenne unreachable, retrying",
			slog.Any("error", err),
			slog.Duration("in", backoff))

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > dialBackoffMax {
			backoff = dialBackoffMax
		}
	}
}

// Close drops the connection. Safe on a nil publisher.
func (p *Publisher) Close() {
	if p == nil {
		return
	}
	p.emitter.Disconnect()
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

	err := p.emitter.EmitNow(channel, event)

	p.mu.Lock()
	if err == nil {
		p.published++
		p.lastEventAt = &now
	} else {
		p.dropped++
		p.lastError = err.Error()
	}
	p.mu.Unlock()

	if err != nil {
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
