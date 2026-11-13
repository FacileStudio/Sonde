package fleet

import (
	"context"
	"sync"
	"time"
)

const observeTTL = 2500 * time.Millisecond

// scanTracker keeps what this instance knows of the scan in flight: the
// runner's latest progress, the last dispatch and delivery, and a short-lived
// copy of what GitHub said, so many viewers polling cost one lookup every few
// seconds. It lives in memory; a restart forgets it, which costs only the
// progress bar until the runner's next update.
type scanTracker struct {
	refresh      sync.Mutex
	mu           sync.Mutex
	progress     tracked
	dispatchedAt time.Time
	delivery     *Delivery
	obs          observation
	lastReport   *time.Time
	observedAt   time.Time
}

func (t *scanTracker) record(update ProgressUpdate, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progress = tracked{update: update, at: now}
}

func (t *scanTracker) dispatched(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dispatchedAt, t.observedAt = now, time.Time{}
}

func (t *scanTracker) delivered(delivery Delivery) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.delivery, t.observedAt = &delivery, time.Time{}
}

func (t *scanTracker) snapshot() (tracked, time.Time, *Delivery) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.progress, t.dispatchedAt, t.delivery
}

// observe returns the cached lookup, refreshing it once it is older than
// observeTTL. Concurrent callers wait for one refresh instead of each calling
// GitHub.
func (t *scanTracker) observe(now time.Time, fetch func() (observation, *time.Time)) (observation, *time.Time) {
	t.refresh.Lock()
	defer t.refresh.Unlock()
	t.mu.Lock()
	obs, last, fresh := t.obs, t.lastReport, now.Sub(t.observedAt) < observeTTL
	t.mu.Unlock()
	if fresh {
		return obs, last
	}
	obs, last = fetch()
	t.mu.Lock()
	t.obs, t.lastReport, t.observedAt = obs, last, now
	t.mu.Unlock()
	return obs, last
}

// fetchScan asks GitHub for the newest run and the store for the newest
// report. A failed lookup is logged and answered from the runner's progress
// alone, so a GitHub outage never hides a scan that is reporting.
func (h *Handler) fetchScan(ctx context.Context) (observation, *time.Time) {
	obs, err := h.dispatcher.Observe(ctx)
	if err != nil {
		h.logger.Warn("fleet scan lookup failed", "error", err)
	}
	if h.store == nil {
		return obs, nil
	}
	runs, err := h.store.Runs(ctx, 1)
	if err != nil || len(runs) == 0 {
		return obs, nil
	}
	at := runs[0].CreatedAt
	return obs, &at
}
