package checker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
)

type fakeRecorder struct {
	mu       sync.Mutex
	opened   int
	resolved int
	checks   int
}

func (r *fakeRecorder) RecordCheck(_ context.Context, _ int64, _ bool, _ int64, _ string, _ *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks++
	return nil
}

func (r *fakeRecorder) OpenIncident(_ context.Context, _ int64, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opened++
	return nil
}

func (r *fakeRecorder) ResolveIncidents(_ context.Context, _ int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolved++
	return nil
}

type fakeSource struct {
	monitors []schemas.Monitor
}

func (s *fakeSource) ListEnabled(context.Context) ([]schemas.Monitor, error) {
	return s.monitors, nil
}

func httpMonitor(target string, interval time.Duration) schemas.Monitor {
	return schemas.Monitor{
		ID:     1,
		Slug:   "web",
		Name:   "Web",
		Type:   monitors.TypeHTTP,
		Target: target,
		MonitorSchedule: schemas.MonitorSchedule{
			IntervalSeconds: int(interval / time.Second),
			TimeoutSeconds:  1,
		},
		MonitorExpectation: schemas.MonitorExpectation{ExpectedStatus: 200},
		Enabled:            true,
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// waitFor polls until collect returns true or the deadline passes.
func waitFor(t *testing.T, timeout time.Duration, collect func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if collect() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

func TestFlappingServerOpensThenResolvesIncident(t *testing.T) {
	var requests int
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		requests++
		n := requests
		mu.Unlock()
		if n <= FailureThreshold {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	recorder := &fakeRecorder{}
	engine := New(
		&fakeSource{monitors: []schemas.Monitor{httpMonitor(server.URL, time.Second)}},
		recorder,
		WithLogger(quietLogger()),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()

	sawDown := false
	sawUp := false
	waitFor(t, 15*time.Second, func() bool {
		for {
			select {
			case transition := <-engine.Transitions():
				if !transition.Up {
					sawDown = true
				} else if sawDown {
					sawUp = true
				}
			default:
				return sawDown && sawUp
			}
		}
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("engine.Run() = %v", err)
	}

	if recorder.opened != 1 || recorder.resolved != 1 {
		t.Fatalf("opened=%d resolved=%d, want 1 and 1", recorder.opened, recorder.resolved)
	}
	if recorder.checks < FailureThreshold+1 {
		t.Fatalf("checks=%d, want at least %d", recorder.checks, FailureThreshold+1)
	}
}

func TestHangingProbeDoesNotLeakGoroutines(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()

	before := runtime.NumGoroutine()

	engine := New(
		&fakeSource{monitors: []schemas.Monitor{
			httpMonitor(server.URL, time.Second),
			httpMonitor(server.URL, time.Second),
		}},
		&fakeRecorder{},
		WithWorkers(4),
		WithSweepInterval(time.Second),
		WithLogger(quietLogger()),
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()

	time.Sleep(2500 * time.Millisecond)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("engine.Run() = %v", err)
	}

	waitFor(t, 10*time.Second, func() bool { return runtime.NumGoroutine() <= before+2 })
}

func TestHeartbeatRecordsAndResolves(t *testing.T) {
	token := "tok-123"
	push := schemas.Monitor{
		ID: 7, Slug: "cron", Name: "Cron", Type: monitors.TypePush,
		MonitorSchedule: schemas.MonitorSchedule{IntervalSeconds: 60},
		Enabled:         true, PushToken: &token,
	}
	recorder := &fakeRecorder{}
	engine := New(&fakeSource{monitors: []schemas.Monitor{push}}, recorder, WithSweepInterval(50*time.Millisecond), WithLogger(quietLogger()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()
	waitFor(t, 5*time.Second, func() bool { return len(engine.states) == 1 })

	if err := engine.Heartbeat(ctx, token); err != nil {
		t.Fatalf("Heartbeat() = %v, want nil", err)
	}
	if err := engine.Heartbeat(ctx, "wrong"); err == nil {
		t.Fatal("Heartbeat(wrong token) = nil, want not found")
	}

	e := engine
	e.mu.Lock()
	state := e.states[7]
	state.lastHeartbeat = time.Now().Add(-10 * time.Minute)
	e.mu.Unlock()
	for i := 0; i < FailureThreshold; i++ {
		e.sweepPush(ctx)
	}

	waitFor(t, 2*time.Second, func() bool {
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		return recorder.opened == 1
	})

	if err := engine.Heartbeat(ctx, token); err != nil {
		t.Fatalf("recovery Heartbeat() = %v", err)
	}
	waitFor(t, 2*time.Second, func() bool {
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		return recorder.resolved == 1
	})
	cancel()
	<-done

	downSeen, upSeen := false, false
	for drained := true; drained; {
		select {
		case transition := <-engine.transitions:
			if !transition.Up {
				downSeen = true
			} else {
				upSeen = true
			}
		default:
			drained = false
		}
	}
	if !downSeen || !upSeen {
		t.Fatalf("transitions down=%v up=%v, want both", downSeen, upSeen)
	}
}

func TestTransitionChannelOverflowIsNotFatal(t *testing.T) {
	engine := New(&fakeSource{}, &fakeRecorder{}, WithLogger(quietLogger()))
	for i := 0; i < maxTransitionsBuffer*3; i++ {
		engine.emit(Transition{MonitorID: int64(i)})
	}
	if len(engine.transitions) != cap(engine.transitions) {
		t.Fatalf("channel holds %d, want full buffer of %d", len(engine.transitions), cap(engine.transitions))
	}
}
