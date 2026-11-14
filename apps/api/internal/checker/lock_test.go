package checker

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type lockReplica struct {
	hits   atomic.Int32
	cancel func()
}

func startLockReplica(t *testing.T, db *sql.DB) *lockReplica {
	t.Helper()
	r := &lockReplica{}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		r.hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(site.Close)

	source := &fakeSource{}
	source.set(httpMonitor(site.URL, time.Second))
	engine := New(source, &fakeRecorder{}, WithLogger(quietLogger()), WithoutTransitions())
	ctx, cancel := contextWithCancel(t)
	r.cancel = cancel
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = engine.RunExclusive(ctx, db)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return r
}

func TestOnlyTheLockHolderProbes(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	a := startLockReplica(t, db)
	b := startLockReplica(t, db)
	waitFor(t, 20*time.Second, func() bool { return a.hits.Load()+b.hits.Load() >= 2 })
	time.Sleep(1500 * time.Millisecond)
	holder, waiter := a, b
	if a.hits.Load() == 0 {
		holder, waiter = b, a
	}
	if got := waiter.hits.Load(); got != 0 {
		t.Fatalf("the replica without the lock probed %d times", got)
	}

	holder.cancel()
	waitFor(t, 30*time.Second, func() bool { return waiter.hits.Load() >= 2 })
}

func TestARunCanFollowAnother(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer site.Close()
	source := &fakeSource{}
	source.set(httpMonitor(site.URL, time.Second))
	recorder := &fakeRecorder{}
	engine := New(source, recorder, WithLogger(quietLogger()), WithoutTransitions())

	for range 2 {
		ctx, cancel := contextWithCancel(t)
		finished := make(chan error, 1)
		go func() { finished <- engine.Run(ctx) }()
		before := recorder.snapshot()
		waitFor(t, 10*time.Second, func() bool { return recorder.snapshot() > before })
		cancel()
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	}
}

func contextWithCancel(t *testing.T) (context.Context, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	return ctx, cancel
}

func (r *fakeRecorder) snapshot() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.checks
}
