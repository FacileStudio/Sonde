// Package httpapi exposes the Sonde domain over HTTP: monitor CRUD,
// incidents, uptime windows, the config export and status pages, both the
// admin-managed side and the unauthenticated public renderer feed.
package httpapi

import (
	"fmt"
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/uptime"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	monitors *monitors.Store
	uptime   *uptime.Service
}

func New(db *gorm.DB, monitorsStore *monitors.Store, uptimeService *uptime.Service) *Service {
	return &Service{db: db, monitors: monitorsStore, uptime: uptimeService}
}

func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	requireAdmin := adminGate(requireAuth)

	router.With(requireAuth).Get("/monitors", service.listMonitors)
	router.With(requireAuth).Post("/monitors", service.createMonitor)
	router.With(requireAuth).Get("/incidents", service.listIncidents)
	router.Route("/monitors/{id}", func(r chi.Router) {
		r.With(requireAuth).Get("/", service.getMonitor)
		r.With(requireAuth).Patch("/", service.updateMonitor)
		r.With(requireAuth).Delete("/", service.deleteMonitor)
		r.With(requireAuth).Get("/incidents", service.monitorIncidents)
		r.With(requireAuth).Get("/uptime", service.monitorUptime)
		r.With(requireAuth).Get("/checks", service.monitorChecks)
	})

	router.With(requireAdmin).Get("/config/export", service.exportConfig)
	router.Route("/status-pages", func(r chi.Router) {
		r.With(requireAdmin).Post("/", service.createStatusPage)
		r.With(requireAdmin).Get("/", service.listStatusPages)
		r.Route("/{id}", func(r chi.Router) {
			r.With(requireAdmin).Get("/", service.getStatusPage)
			r.With(requireAdmin).Patch("/", service.updateStatusPage)
			r.With(requireAdmin).Delete("/", service.deleteStatusPage)
		})
	})

	router.Get("/public/status/{slug}", service.publicStatusPage)
}

func adminGate(requireAuth func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := authcontext.IdentityFromContext(r.Context())
			if !ok || !identity.IsAdmin {
				httpjson.WriteError(w, errors.Forbidden("administrator role required"))
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func monitorID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "id")
	var id int64
	if _, err := fmt.Sscan(raw, &id); err != nil || id < 1 {
		return 0, errors.Invalid("id must be a positive integer")
	}
	return id, nil
}
