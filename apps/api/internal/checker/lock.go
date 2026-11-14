package checker

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/FacileStudio/tronc/joblock"
)

const (
	lockKey         = "sonde-checker"
	runRetryBackoff = 5 * time.Second
)

// reset readies the engine for a fresh Run. A Run that ended closed the jobs
// channel and left watchers' stop channels behind, so a second Run on the same
// engine, after the lock was lost and won back, must start from a clean slate.
func (e *Engine) reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, state := range e.states {
		select {
		case <-state.stop:
		default:
			close(state.stop)
		}
	}
	e.states = make(map[int64]*monitorState)
	e.byToken = make(map[string]*monitorState)
	e.jobs = make(chan *monitorState, defaultWorkers*2)
}

// Load reads the enabled monitors without probing them, so a replica that
// does not hold the checker lock still knows its push tokens and answers a
// heartbeat instead of a 404.
func (e *Engine) Load(ctx context.Context) error {
	_, err := e.reload(ctx)
	return err
}

// RunExclusive probes only while this replica holds the Postgres advisory
// lock "sonde-checker" in db, so two replicas never probe, open incidents or
// emit transitions at once. It returns when ctx ends.
func (e *Engine) RunExclusive(ctx context.Context, db *sql.DB) error {
	if err := e.Load(ctx); err != nil {
		e.logger.Warn("loading monitors before taking the checker lock", slog.Any("error", err))
	}
	return joblock.Hold(ctx, db, lockKey, e.runHeld, joblock.WithOnError(func(err error) {
		e.logger.Warn("checker lock", slog.Any("error", err))
	}))
}

func (e *Engine) runHeld(held context.Context) {
	for held.Err() == nil {
		err := e.Run(held)
		if err == nil {
			return
		}
		e.logger.Error("checker stopped", slog.Any("error", err))
		select {
		case <-held.Done():
		case <-time.After(runRetryBackoff):
		}
	}
}
