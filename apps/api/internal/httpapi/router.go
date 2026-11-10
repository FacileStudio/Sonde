// Package httpapi exposes the Sonde domain over HTTP: monitor CRUD,
// incidents, uptime windows, the config export and status pages, both the
// admin-managed side and the unauthenticated public renderer feed.
package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/retention"
	"github.com/FacileStudio/Sonde/apps/api/internal/stats"
	"github.com/FacileStudio/Sonde/apps/api/internal/uptime"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// Bus reports what the event bus is doing. *events.Publisher satisfies it, and
// a nil publisher is a valid one: it answers "disabled", which is the
// unconfigured instance and the production default.
type Bus interface {
	State(ctx context.Context) events.State
}

type Service struct {
	db        *gorm.DB
	monitors  *monitors.Store
	uptime    *uptime.Service
	stats     *stats.Service
	bus       Bus
	journal   string
	retention int
}

// Deps is what the service reads beyond the database. It is a struct because
// the constructor was already three positional arguments and settings needs
// three more, none of which a caller should have to remember the order of.
type Deps struct {
	Monitors         *monitors.Store
	Uptime           *uptime.Service
	Stats            *stats.Service
	Bus              Bus
	JournalURL       string
	RawRetentionDays int
}

func New(db *gorm.DB, deps Deps) *Service {
	return &Service{
		db:        db,
		monitors:  deps.Monitors,
		uptime:    deps.Uptime,
		stats:     deps.Stats,
		bus:       deps.Bus,
		journal:   deps.JournalURL,
		retention: deps.RawRetentionDays,
	}
}

// integrationsSnapshot reads the instance's outbound wiring. The bus half
// tolerates an absent Bus so a router built without one — which is what the
// route-table test does — reports disabled instead of panicking.
func (s *Service) integrationsSnapshot(ctx context.Context) IntegrationsResponse {
	state := events.State{State: events.StateDisabled, Channels: events.Channels()}
	if s.bus != nil {
		state = s.bus.State(ctx)
	}

	var lastEvent *string
	if state.LastEventAt != nil {
		at := state.LastEventAt.UTC().Format(time.RFC3339)
		lastEvent = &at
	}

	return IntegrationsResponse{
		Bus: BusResponse{
			Configured:  state.Configured,
			State:       state.State,
			Instance:    state.Instance,
			App:         state.App,
			InstanceID:  state.InstanceID,
			Channels:    state.Channels,
			Published:   state.Published,
			Pending:     state.Pending,
			LastEventAt: lastEvent,
			LastError:   state.LastError,
		},
		Journal: JournalResponse{Configured: s.journal != "", URL: s.journal},
		Retention: RetentionResponse{
			RawDays:       s.retention,
			HourlyMonths:  retention.HourlyRetentionMonths,
			FailThreshold: checker.FailureThreshold,
		},
	}
}

func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	requireAdmin := adminGate(requireAuth)

	router.With(requireAuth).Get("/monitors", service.listMonitors)
	router.With(requireAuth).Post("/monitors", service.createMonitor)
	// Static before the {id} pattern in the source, though chi does not need
	// the ordering: its trie tries static children ahead of parameter ones, so
	// /monitors/overview can never be read as a monitor called "overview".
	router.With(requireAuth).Get("/monitors/overview", service.monitorsOverview)
	router.With(requireAuth).Get("/incidents", service.listIncidents)
	router.With(requireAuth).Get("/incidents/stats", service.incidentStats)
	router.With(requireAuth).Get("/integrations", service.integrations)
	router.Route("/monitors/{id}", func(r chi.Router) {
		r.With(requireAuth).Get("/", service.getMonitor)
		r.With(requireAuth).Patch("/", service.updateMonitor)
		r.With(requireAuth).Delete("/", service.deleteMonitor)
		r.With(requireAuth).Get("/incidents", service.monitorIncidents)
		r.With(requireAuth).Get("/uptime", service.monitorUptime)
		r.With(requireAuth).Get("/uptime/summary", service.monitorUptimeSummary)
		r.With(requireAuth).Get("/checks", service.monitorChecks)
		r.With(requireAuth).Get("/history", service.monitorHistory)
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
			identity, ok := kit.FromContext(r.Context())
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
