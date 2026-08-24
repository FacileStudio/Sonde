// Package checker runs the probe engine: one ticker per enabled monitor, a
// bounded worker pool executing each probe under a hard context deadline, raw
// check persistence, incident state transitions and the push-heartbeat route.
package checker

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	FailureThreshold = 3

	PushGraceFactor = 2

	maxTransitionsBuffer = 256
	defaultWorkers       = 10
	defaultSweepInterval = 15 * time.Second
)

type Transition struct {
	MonitorID int64
	Slug      string
	Name      string
	Up        bool
	LatencyMS int64
	Error     string
}

type Recorder interface {
	RecordCheck(ctx context.Context, monitorID int64, ok bool, latencyMS int64, cause string, tlsExpiresAt *time.Time) error
	OpenIncident(ctx context.Context, monitorID int64, cause string) error
	ResolveIncidents(ctx context.Context, monitorID int64) error
}

type Source interface {
	ListEnabled(ctx context.Context) ([]schemas.Monitor, error)
}

type monitorState struct {
	monitor       schemas.Monitor
	up            bool
	failures      int
	lastHeartbeat time.Time
	inFlight      bool
}

type Engine struct {
	source   Source
	recorder Recorder
	logger   *slog.Logger
	client   *http.Client
	dialer   *net.Dialer
	workers  int
	sweep    time.Duration

	transitions chan Transition

	mu      sync.Mutex
	states  map[int64]*monitorState
	byToken map[string]*monitorState

	jobs chan *monitorState
	wg   sync.WaitGroup
}

type Option func(*Engine)

func WithWorkers(n int) Option { return func(e *Engine) { e.workers = n } }

func WithSweepInterval(d time.Duration) Option { return func(e *Engine) { e.sweep = d } }

func WithLogger(logger *slog.Logger) Option { return func(e *Engine) { e.logger = logger } }

func WithHTTPClient(client *http.Client) Option { return func(e *Engine) { e.client = client } }

func New(source Source, recorder Recorder, options ...Option) *Engine {
	engine := &Engine{
		source:      source,
		recorder:    recorder,
		logger:      slog.Default(),
		client:      &http.Client{},
		dialer:      &net.Dialer{},
		workers:     defaultWorkers,
		sweep:       defaultSweepInterval,
		transitions: make(chan Transition, maxTransitionsBuffer),
		states:      make(map[int64]*monitorState),
		byToken:     make(map[string]*monitorState),
		jobs:        make(chan *monitorState, defaultWorkers*2),
	}
	for _, option := range options {
		option(engine)
	}
	return engine
}

func NewGorm(db *gorm.DB, options ...Option) *Engine {
	store := monitors.NewStore(db)
	return New(store, &GormRecorder{db: db}, options...)
}

func (e *Engine) Transitions() <-chan Transition { return e.transitions }

func (e *Engine) Run(ctx context.Context) error {
	initial, err := e.source.ListEnabled(ctx)
	if err != nil {
		return err
	}

	e.mu.Lock()
	for i := range initial {
		state := &monitorState{monitor: initial[i], up: true, lastHeartbeat: time.Now()}
		e.states[initial[i].ID] = state
		if initial[i].PushToken != nil {
			e.byToken[*initial[i].PushToken] = state
		}
	}
	e.mu.Unlock()

	for range e.workers {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			for state := range e.jobs {
				e.probe(ctx, state)
				state.inFlight = false
			}
		}()
	}

	for _, state := range e.snapshot() {
		if state.monitor.Type == monitors.TypePush {
			continue
		}
		e.wg.Add(1)
		go e.watch(ctx, state)
	}

	sweeper := time.NewTicker(e.sweep)
	defer sweeper.Stop()

	for {
		select {
		case <-ctx.Done():
			close(e.jobs)
			e.wg.Wait()
			return nil
		case <-sweeper.C:
			e.sweepPush(ctx)
		}
	}
}

func (e *Engine) snapshot() []*monitorState {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*monitorState, 0, len(e.states))
	for _, state := range e.states {
		out = append(out, state)
	}
	return out
}

func (e *Engine) watch(ctx context.Context, state *monitorState) {
	defer e.wg.Done()

	interval := time.Duration(state.monitor.IntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.submit(ctx, state)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.submit(ctx, state)
		}
	}
}

func (e *Engine) submit(ctx context.Context, state *monitorState) {
	if ctx.Err() != nil || state.inFlight {
		return
	}
	select {
	case e.jobs <- state:
		state.inFlight = true
	default:
	}
}

func (e *Engine) emit(transition Transition) {
	select {
	case e.transitions <- transition:
	default:
		e.logger.Warn("transition channel full, dropping event",
			slog.Int64("monitor_id", transition.MonitorID))
	}
}

func (e *Engine) applyResult(ctx context.Context, state *monitorState, result probeResult) {
	var opened bool
	var resolved bool

	e.mu.Lock()
	wasUp := state.up
	if result.ok {
		state.failures = 0
		state.up = true
		resolved = !wasUp
	} else {
		state.failures++
		if wasUp && state.failures >= FailureThreshold {
			state.up = false
			opened = true
		}
	}
	monitor := state.monitor
	e.mu.Unlock()

	if err := e.recorder.RecordCheck(ctx, monitor.ID, result.ok, result.latencyMS, result.errText, result.tlsExpiresAt); err != nil {
		e.logger.Error("recording check", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
	}

	switch {
	case opened:
		cause := result.errText
		if len(cause) > 500 {
			cause = cause[:500]
		}
		if err := e.recorder.OpenIncident(ctx, monitor.ID, cause); err != nil {
			e.logger.Error("opening incident", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
		}
		e.emit(Transition{MonitorID: monitor.ID, Slug: monitor.Slug, Name: monitor.Name, Up: false, LatencyMS: result.latencyMS, Error: result.errText})
	case resolved:
		if err := e.recorder.ResolveIncidents(ctx, monitor.ID); err != nil {
			e.logger.Error("resolving incidents", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
		}
		e.emit(Transition{MonitorID: monitor.ID, Slug: monitor.Slug, Name: monitor.Name, Up: true, LatencyMS: result.latencyMS})
	}
}

func (e *Engine) Heartbeat(ctx context.Context, token string) error {
	e.mu.Lock()
	state, ok := e.byToken[token]
	if !ok {
		e.mu.Unlock()
		return errors.NotFound("unknown push token")
	}
	wasUp := state.up
	state.failures = 0
	state.lastHeartbeat = time.Now()
	state.up = true
	monitor := state.monitor
	e.mu.Unlock()

	if err := e.recorder.RecordCheck(ctx, monitor.ID, true, 0, "", nil); err != nil {
		e.logger.Error("recording heartbeat", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
	}
	if !wasUp {
		if err := e.recorder.ResolveIncidents(ctx, monitor.ID); err != nil {
			e.logger.Error("resolving incidents", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
		}
		e.emit(Transition{MonitorID: monitor.ID, Slug: monitor.Slug, Name: monitor.Name, Up: true})
	}
	return nil
}

func (e *Engine) sweepPush(ctx context.Context) {
	now := time.Now()

	e.mu.Lock()
	type openRequest struct {
		monitor schemas.Monitor
		silent  time.Duration
	}
	var opens []openRequest
	for _, state := range e.states {
		if state.monitor.Type != monitors.TypePush || !state.up {
			continue
		}
		grace := time.Duration(state.monitor.IntervalSeconds*PushGraceFactor) * time.Second
		silent := now.Sub(state.lastHeartbeat)
		if silent <= grace {
			continue
		}
		state.failures++
		if state.failures >= FailureThreshold {
			state.up = false
			opens = append(opens, openRequest{monitor: state.monitor, silent: silent.Truncate(time.Second)})
		}
	}
	e.mu.Unlock()

	for _, request := range opens {
		grace := time.Duration(request.monitor.IntervalSeconds*PushGraceFactor) * time.Second
		cause := "no push heartbeat for " + request.silent.String() + " (grace " + grace.String() + ")"
		if err := e.recorder.OpenIncident(ctx, request.monitor.ID, cause); err != nil {
			e.logger.Error("opening incident", slog.Any("error", err), slog.Int64("monitor_id", request.monitor.ID))
		}
		e.emit(Transition{MonitorID: request.monitor.ID, Slug: request.monitor.Slug, Name: request.monitor.Name, Up: false, Error: cause})
	}
}
