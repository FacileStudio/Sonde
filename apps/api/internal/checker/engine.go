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
	"sync/atomic"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	FailureThreshold = 3

	PushGraceFactor = 2

	maxTransitionsBuffer  = 256
	defaultWorkers        = 10
	defaultSweepInterval  = 15 * time.Second
	defaultReloadInterval = 15 * time.Second
)

// Transition is one monitor crossing between up and down. IncidentAt is when
// the outage it belongs to started, and is carried on the recovery as well as
// on the outage so both ends of an incident share one stable identity.
type Transition struct {
	MonitorID  int64
	Slug       string
	Name       string
	Up         bool
	LatencyMS  int64
	Error      string
	IncidentAt time.Time
}

type Recorder interface {
	RecordCheck(ctx context.Context, monitorID int64, ok bool, latencyMS int64, cause string, tlsExpiresAt *time.Time) error
	OpenIncident(ctx context.Context, monitorID int64, cause string) error
	ResolveIncidents(ctx context.Context, monitorID int64) error
}

type Source interface {
	ListEnabled(ctx context.Context) ([]schemas.Monitor, error)
}

// monitorState is everything the engine remembers about one monitor. stop is
// closed to retire the watcher goroutine bound to it, which is how a disabled,
// deleted or rescheduled monitor stops being probed.
type monitorState struct {
	monitor       schemas.Monitor
	up            bool
	failures      int
	lastHeartbeat time.Time
	downSince     time.Time
	stop          chan struct{}
	inFlight      atomic.Bool
}

// watchStart pairs a state with the stop channel its watcher must observe.
// Reading state.stop again outside the lock would race a reconciliation that
// has already replaced it.
type watchStart struct {
	state *monitorState
	stop  <-chan struct{}
}

type Engine struct {
	source      Source
	recorder    Recorder
	logger      *slog.Logger
	client      *http.Client
	dialer      *net.Dialer
	workers     int
	sweep       time.Duration
	reloadEvery time.Duration

	transitions chan Transition

	mu      sync.Mutex
	states  map[int64]*monitorState
	byToken map[string]*monitorState

	jobs chan *monitorState
	// Two groups, not one: every watcher has to be gone before jobs is
	// closed, or a watcher still in submit sends on a closed channel and
	// takes the process down.
	watcherWG sync.WaitGroup
	workerWG  sync.WaitGroup
}

type Option func(*Engine)

func WithWorkers(n int) Option { return func(e *Engine) { e.workers = n } }

func WithSweepInterval(d time.Duration) Option { return func(e *Engine) { e.sweep = d } }

// WithReloadInterval sets how often the engine re-reads the enabled monitors
// and converges on them.
func WithReloadInterval(d time.Duration) Option { return func(e *Engine) { e.reloadEvery = d } }

// WithoutTransitions drops state changes instead of queueing them. The channel
// is bounded and nothing drains it when no publisher is configured, which is
// the production default, so without this every change past the first 256 logs
// a warning about a queue nobody is meant to read.
func WithoutTransitions() Option { return func(e *Engine) { e.transitions = nil } }

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
		reloadEvery: defaultReloadInterval,
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

// Run probes until ctx ends. The monitor set is re-read from the source every
// reload interval rather than only at boot, so a monitor created, edited,
// disabled or deleted through the API takes effect without a restart.
func (e *Engine) Run(ctx context.Context) error {
	starts, err := e.reload(ctx)
	if err != nil {
		return err
	}

	for range e.workers {
		e.workerWG.Add(1)
		go func() {
			defer e.workerWG.Done()
			for state := range e.jobs {
				e.probe(ctx, state)
				state.inFlight.Store(false)
			}
		}()
	}

	e.startWatchers(ctx, starts)

	sweeper := time.NewTicker(e.sweep)
	defer sweeper.Stop()
	reloader := time.NewTicker(e.reloadEvery)
	defer reloader.Stop()

	for {
		select {
		case <-ctx.Done():
			e.watcherWG.Wait()
			close(e.jobs)
			e.workerWG.Wait()
			return nil
		case <-sweeper.C:
			e.sweepPush(ctx)
		case <-reloader.C:
			starts, err := e.reload(ctx)
			if err != nil {
				e.logger.Error("reloading monitors", slog.Any("error", err))
				continue
			}
			e.startWatchers(ctx, starts)
		}
	}
}

// reload converges the in-memory state on the enabled monitors the source
// reports, and returns the watchers the caller must start. Starting them is
// left to the caller because it happens outside the lock, and only the Run
// goroutine may do it: nothing may spawn a watcher after Run has begun
// draining them.
func (e *Engine) reload(ctx context.Context) ([]watchStart, error) {
	latest, err := e.source.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	var starts []watchStart
	live := make(map[int64]struct{}, len(latest))

	for i := range latest {
		monitor := latest[i]
		live[monitor.ID] = struct{}{}

		state, known := e.states[monitor.ID]
		if !known {
			state = &monitorState{
				monitor:       monitor,
				up:            true,
				lastHeartbeat: now,
				stop:          make(chan struct{}),
			}
			e.states[monitor.ID] = state
			if monitor.PushToken != nil {
				e.byToken[*monitor.PushToken] = state
			}
			if monitor.Type != monitors.TypePush {
				starts = append(starts, watchStart{state: state, stop: state.stop})
			}
			continue
		}

		previous := state.monitor
		state.monitor = monitor
		if previous.PushToken != nil {
			delete(e.byToken, *previous.PushToken)
		}
		if monitor.PushToken != nil {
			e.byToken[*monitor.PushToken] = state
		}

		if !needsRewatch(previous, monitor) {
			continue
		}
		close(state.stop)
		state.stop = make(chan struct{})
		if monitor.Type != monitors.TypePush {
			starts = append(starts, watchStart{state: state, stop: state.stop})
		}
	}

	for id, state := range e.states {
		if _, ok := live[id]; ok {
			continue
		}
		close(state.stop)
		delete(e.states, id)
		if state.monitor.PushToken != nil {
			delete(e.byToken, *state.monitor.PushToken)
		}
	}

	return starts, nil
}

// needsRewatch reports whether a change forces a new watcher goroutine. Only
// these two do: a watcher's ticker is fixed at the interval it started with,
// and a push monitor has no watcher at all. Every other field a probe reads is
// taken from the state at probe time, so a new target or expectation applies on
// the next tick by itself.
func needsRewatch(before, after schemas.Monitor) bool {
	return before.Type != after.Type || before.IntervalSeconds != after.IntervalSeconds
}

func (e *Engine) startWatchers(ctx context.Context, starts []watchStart) {
	for _, start := range starts {
		e.watcherWG.Add(1)
		go e.watch(ctx, start.state, start.stop)
	}
}

func (e *Engine) watch(ctx context.Context, state *monitorState, stop <-chan struct{}) {
	defer e.watcherWG.Done()

	e.mu.Lock()
	interval := time.Duration(state.monitor.IntervalSeconds) * time.Second
	e.mu.Unlock()
	// time.NewTicker panics on a non-positive interval, and a row written
	// outside the validated store is enough to reach one.
	if interval <= 0 {
		interval = time.Duration(monitors.DefaultIntervalSeconds) * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.submit(ctx, state)

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			e.submit(ctx, state)
		}
	}
}

func (e *Engine) submit(ctx context.Context, state *monitorState) {
	if ctx.Err() != nil || !state.inFlight.CompareAndSwap(false, true) {
		return
	}
	select {
	case e.jobs <- state:
	default:
		state.inFlight.Store(false)
	}
}

func (e *Engine) emit(transition Transition) {
	if e.transitions == nil {
		return
	}
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
			state.downSince = time.Now()
			opened = true
		}
	}
	monitor := state.monitor
	incidentAt := state.downSince
	e.mu.Unlock()

	if err := e.recorder.RecordCheck(ctx, monitor.ID, result.ok, result.latencyMS, result.errText, result.tlsExpiresAt); err != nil {
		e.logger.Error("recording check", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
	}

	if result.ok {
		if err := e.recorder.ResolveIncidents(ctx, monitor.ID); err != nil {
			e.logger.Error("resolving incidents", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
		}
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
		e.emit(Transition{MonitorID: monitor.ID, Slug: monitor.Slug, Name: monitor.Name, Up: false, LatencyMS: result.latencyMS, Error: result.errText, IncidentAt: incidentAt})
	case resolved:
		e.emit(Transition{MonitorID: monitor.ID, Slug: monitor.Slug, Name: monitor.Name, Up: true, LatencyMS: result.latencyMS, IncidentAt: incidentAt})
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
	incidentAt := state.downSince
	e.mu.Unlock()

	if err := e.recorder.RecordCheck(ctx, monitor.ID, true, 0, "", nil); err != nil {
		e.logger.Error("recording heartbeat", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
	}
	if !wasUp {
		if err := e.recorder.ResolveIncidents(ctx, monitor.ID); err != nil {
			e.logger.Error("resolving incidents", slog.Any("error", err), slog.Int64("monitor_id", monitor.ID))
		}
		e.emit(Transition{MonitorID: monitor.ID, Slug: monitor.Slug, Name: monitor.Name, Up: true, IncidentAt: incidentAt})
	}
	return nil
}

func (e *Engine) sweepPush(ctx context.Context) {
	now := time.Now()

	e.mu.Lock()
	type openRequest struct {
		monitor    schemas.Monitor
		silent     time.Duration
		incidentAt time.Time
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
			state.downSince = now
			opens = append(opens, openRequest{
				monitor:    state.monitor,
				silent:     silent.Truncate(time.Second),
				incidentAt: now,
			})
		}
	}
	e.mu.Unlock()

	for _, request := range opens {
		grace := time.Duration(request.monitor.IntervalSeconds*PushGraceFactor) * time.Second
		cause := "no push heartbeat for " + request.silent.String() + " (grace " + grace.String() + ")"
		if err := e.recorder.OpenIncident(ctx, request.monitor.ID, cause); err != nil {
			e.logger.Error("opening incident", slog.Any("error", err), slog.Int64("monitor_id", request.monitor.ID))
		}
		e.emit(Transition{MonitorID: request.monitor.ID, Slug: request.monitor.Slug, Name: request.monitor.Name, Up: false, Error: cause, IncidentAt: request.incidentAt})
	}
}
