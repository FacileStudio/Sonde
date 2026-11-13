package fleet

import (
	"net/http"

	"github.com/FacileStudio/tronc/middleware"
	"github.com/go-chi/chi/v5"
)

const (
	ingestPerSecond = 1.0 / 60
	ingestBurst     = 10
)

// ingestLimit bounds accepted deliveries to a burst of ten, refilled at one a
// minute, in a single bucket. It runs after the token check on purpose: a
// refused request costs a hash and nothing else, so limiting strangers would
// only let them spend the runner's budget, while an accepted one opens a
// transaction over thousands of rows.
func ingestLimit() func(http.Handler) http.Handler {
	return middleware.Ratelimit(middleware.RatelimitConfig{
		Limit:   ingestPerSecond,
		Burst:   ingestBurst,
		KeyFunc: func(*http.Request) string { return "fleet-ingest" },
	})
}

// RegisterRoutes mounts the fleet API. The report and progress routes sit
// outside the session gate on purpose, since their caller is a scheduled job
// holding a token;
// every other route goes through Gate.
func RegisterRoutes(router chi.Router, store *Store, requireAuth func(http.Handler) http.Handler, cfg Config) {
	h := NewHandler(store, cfg.logger())
	h.dispatcher = cfg.Dispatch
	gate := Gate(requireAuth, cfg.Admins)

	router.Route("/fleet", func(r chi.Router) {
		r.With(IngestAuth(cfg.Token, cfg.logger()), ingestLimit()).Post("/reports", h.Ingest)
		r.With(IngestAuth(cfg.Token, cfg.logger()), progressLimit()).Post("/scans/progress", h.ScanProgress)

		r.Group(func(r chi.Router) {
			r.Use(gate)
			r.Get("/actions", h.Actions)
			r.Get("/drift", h.Drift)
			r.Get("/graph", h.Graph)
			r.Get("/conformance", h.Conformance)
			r.Get("/projects", h.Projects)
			r.Patch("/projects/{id}", h.UpdateProject)
			r.Get("/runs", h.Runs)
			r.With(rescanLimit()).Post("/scans", h.Rescan)
			r.Get("/scans/current", h.CurrentScan)
			r.Get("/policies", h.ListPolicies)
			r.Post("/policies", h.CreatePolicy)
			r.Delete("/policies/{id}", h.DeletePolicy)
		})
	})
}
