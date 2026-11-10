package events

import (
	"context"
	"time"
)

// Bus states, as an operator reads them in settings. There is deliberately no
// "gave up": the publisher retries the first dial forever, so an unreachable
// bus is always either Connecting or Reconnecting and never a terminal state
// that only a restart can leave.
const (
	StateDisabled     = "disabled"
	StateConnecting   = "connecting"
	StateConnected    = "connected"
	StateReconnecting = "reconnecting"
)

// State is what the bus is doing right now, for the settings page and for
// anything else that has to answer "are alerts actually leaving this box".
// Pending counts queued events not yet sent. The secret is never part of it.
type State struct {
	Configured  bool       `json:"configured"`
	State       string     `json:"state"`
	Instance    string     `json:"instance"`
	App         string     `json:"app"`
	InstanceID  string     `json:"instance_id,omitempty"`
	Channels    []string   `json:"channels"`
	Published   int64      `json:"published"`
	Pending     int64      `json:"pending"`
	LastEventAt *time.Time `json:"last_event_at"`
	LastError   string     `json:"last_error,omitempty"`
}

// State reports what the bus is doing. Safe on a nil publisher, which is the
// unconfigured instance.
func (p *Publisher) State(ctx context.Context) State {
	if p == nil {
		return State{State: StateDisabled, Channels: Channels()}
	}
	pending, lastError := p.pending(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()
	if lastError == "" {
		lastError = p.lastError
	}
	return State{
		Configured:  true,
		State:       p.connection(),
		Instance:    p.config.Instance,
		App:         p.config.App,
		InstanceID:  p.config.InstanceID,
		Channels:    Channels(),
		Published:   p.published,
		Pending:     pending,
		LastEventAt: p.lastEventAt,
		LastError:   lastError,
	}
}

func (p *Publisher) connection() string {
	switch {
	case p.emitter.IsConnected():
		return StateConnected
	case p.dialed:
		return StateReconnecting
	}
	return StateConnecting
}

func (p *Publisher) pending(ctx context.Context) (int64, string) {
	var n int64
	err := p.db.QueryRowContext(ctx, `SELECT count(*) FROM antenne_outbox WHERE sent_at IS NULL`).Scan(&n)
	if err != nil {
		return 0, err.Error()
	}
	return n, ""
}
