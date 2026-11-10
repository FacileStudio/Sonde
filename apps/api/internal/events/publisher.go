package events

import (
	"context"
	"database/sql"
	stderrors "errors"
	"log/slog"
	"math"
	"sync"
	"time"

	antenne "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/antenne-client/go/outbox"
)

const (
	connectTimeout    = 10 * time.Second
	reconnectAttempts = math.MaxInt32
	dialBackoffMin    = 5 * time.Second
	dialBackoffMax    = 60 * time.Second
	drainMaxBackoff   = 5 * time.Second
)

type emitter interface {
	Connect(ctx context.Context) error
	EmitNow(channel string, payload any) error
	Disconnect()
	IsConnected() bool
}

// Publisher drains the outbox to the bus. It owns the live client and dials it
// itself, because antenne-client gives up after a bounded number of reconnects
// and never retries the first dial: Sonde and Antenne restart together on a
// suite deploy, and whoever loses the race would publish nothing until a
// restart. The reconnect budget is unbounded for the same reason.
type Publisher struct {
	config  *antenne.Config
	emitter emitter
	drainer *outbox.Drainer
	db      *sql.DB
	logger  *slog.Logger

	mu          sync.Mutex
	dialed      bool
	published   int64
	lastEventAt *time.Time
	lastError   string
}

// New returns a publisher, or nil when config is nil. Every method is a no-op
// on a nil publisher. New does not dial: Start does, off the boot path.
func New(config *antenne.Config, db *sql.DB, logger *slog.Logger) *Publisher {
	if config == nil {
		return nil
	}
	client := antenne.NewClient(config,
		antenne.WithMaxReconnect(reconnectAttempts),
		antenne.WithOnError(func(err error) { logger.Warn("antenne connection", slog.Any("error", err)) }))
	return newPublisher(config, client, db, logger)
}

func newPublisher(config *antenne.Config, client emitter, db *sql.DB, logger *slog.Logger) *Publisher {
	p := &Publisher{config: config, emitter: client, db: db, logger: logger}
	p.drainer = outbox.NewDrainer(db, p.emit, outbox.WithMaxBackoff(drainMaxBackoff), outbox.WithOnError(p.drainFailed))
	return p
}

// Start dials the bus and drains the outbox in the background until ctx ends.
func (p *Publisher) Start(ctx context.Context) {
	if p == nil {
		return
	}
	go p.dial(ctx)
	go p.drainer.Run(ctx)
}

// Close drops the connection. Queued events stay in the outbox.
func (p *Publisher) Close() {
	if p == nil {
		return
	}
	p.emitter.Disconnect()
}

func (p *Publisher) dial(ctx context.Context) {
	for backoff := dialBackoffMin; ; backoff = min(backoff*2, dialBackoffMax) {
		attempt, cancel := context.WithTimeout(ctx, connectTimeout)
		err := p.emitter.Connect(attempt)
		cancel()
		p.recordDial(err)
		if err == nil {
			p.logger.Info("publishing monitor events", slog.String("instance", p.config.Instance), slog.String("app", p.config.App))
			return
		}
		p.logger.Warn("antenne unreachable, retrying", slog.Any("error", err), slog.Duration("in", backoff))
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func (p *Publisher) recordDial(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		p.dialed, p.lastError = true, ""
		return
	}
	p.lastError = err.Error()
}

func (p *Publisher) emit(channel string, payload any) error {
	if err := p.emitter.EmitNow(channel, payload); err != nil {
		return err
	}
	now := time.Now().UTC()
	p.mu.Lock()
	p.published++
	p.lastEventAt = &now
	p.mu.Unlock()
	return nil
}

func (p *Publisher) drainFailed(err error) {
	if stderrors.Is(err, antenne.ErrNotConnected) {
		return
	}
	p.logger.Warn("draining the antenne outbox", slog.Any("error", err))
	p.mu.Lock()
	p.lastError = err.Error()
	p.mu.Unlock()
}
