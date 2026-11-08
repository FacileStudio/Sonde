package fleet

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the fleet API. The report route sits outside the
// session gate on purpose, since its caller is a scheduled job holding a token;
// every other route goes through Gate.
func RegisterRoutes(router chi.Router, store *Store, requireAuth func(http.Handler) http.Handler, cfg Config) {
	h := NewHandler(store)
	gate := Gate(requireAuth, cfg.Admins)

	router.Route("/fleet", func(r chi.Router) {
		r.With(IngestAuth(cfg.Token)).Post("/reports", h.Ingest)

		r.Group(func(r chi.Router) {
			r.Use(gate)
			r.Get("/actions", h.Actions)
			r.Get("/drift", h.Drift)
			r.Get("/projects", h.Projects)
			r.Patch("/projects/{id}", h.UpdateProject)
			r.Get("/runs", h.Runs)
			r.Get("/policies", h.ListPolicies)
			r.Post("/policies", h.CreatePolicy)
			r.Delete("/policies/{id}", h.DeletePolicy)
		})
	})
}
