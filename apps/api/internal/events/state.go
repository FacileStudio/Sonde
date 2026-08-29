package events

import "time"

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

// AppName is the pool identity, defaulted.
func (c Config) AppName() string {
	if c.App == "" {
		return DefaultApp
	}
	return c.App
}

// State is what the bus is doing right now, for the settings page and for
// anything else that has to answer "are alerts actually leaving this box".
// The secret is never part of it.
type State struct {
	Configured  bool       `json:"configured"`
	State       string     `json:"state"`
	Instance    string     `json:"instance"`
	App         string     `json:"app"`
	InstanceID  string     `json:"instance_id,omitempty"`
	Channels    []string   `json:"channels"`
	Published   int64      `json:"published"`
	Dropped     int64      `json:"dropped"`
	LastEventAt *time.Time `json:"last_event_at"`
	LastError   string     `json:"last_error,omitempty"`
}

// State reports what the bus is doing. Safe on a nil publisher, which is the
// unconfigured instance and the production default.
func (p *Publisher) State() State {
	if p == nil {
		return State{State: StateDisabled, Channels: []string{ChannelDown, ChannelUp}}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state := StateConnecting
	switch {
	case p.emitter.IsConnected():
		state = StateConnected
	case p.dialed:
		state = StateReconnecting
	}

	return State{
		Configured:  true,
		State:       state,
		Instance:    p.config.Instance,
		App:         p.config.AppName(),
		InstanceID:  p.config.InstanceID,
		Channels:    []string{ChannelDown, ChannelUp},
		Published:   p.published,
		Dropped:     p.dropped,
		LastEventAt: p.lastEventAt,
		LastError:   p.lastError,
	}
}
