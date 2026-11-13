package fleet

import (
	"net/http"
	"time"

	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/middleware"
)

const (
	progressPerSecond = 2
	progressBurst     = 20
)

// progressLimit bounds progress updates in their own bucket, so a chatty
// runner can never spend the budget of its report delivery. Like ingest, it
// sits after the token check.
func progressLimit() func(http.Handler) http.Handler {
	return middleware.Ratelimit(middleware.RatelimitConfig{
		Limit:   progressPerSecond,
		Burst:   progressBurst,
		KeyFunc: func(*http.Request) string { return "fleet-progress" },
	})
}

// ScanProgress keeps the runner's latest update in memory.
func (h *Handler) ScanProgress(w http.ResponseWriter, r *http.Request) {
	var update ProgressUpdate
	if err := httpjson.DecodeJSON(w, r, &update); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := update.Validate(); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	h.scans.record(update, h.now())
	w.WriteHeader(http.StatusNoContent)
}

// CurrentScan answers the state of the newest scan: GitHub's run and steps,
// the runner's progress, and the last report.
func (h *Handler) CurrentScan(w http.ResponseWriter, r *http.Request) {
	now := h.now()
	obs, last := h.scans.observe(now, func() (observation, *time.Time) { return h.fetchScan(r.Context()) })
	progress, dispatchedAt, delivery := h.scans.snapshot()
	status := scanView{obs: obs, progress: progress, dispatchedAt: dispatchedAt, now: now}.status()
	status.LastReportAt, status.LastDelivery = last, delivery
	httpjson.WriteJSON(w, http.StatusOK, status)
}
