// Router assembly, kept out of main.go so the file that owns the process
// lifecycle is not also the file that owns the route table.
package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
	"github.com/FacileStudio/Sonde/apps/api/internal/httpapi"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/stats"
	"github.com/FacileStudio/Sonde/apps/api/internal/uptime"
	"github.com/FacileStudio/Sonde/apps/api/internal/webhooks"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"
	"github.com/FacileStudio/Sonde/apps/api/modules/users"

	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/httpx"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// apiTitle names this API in the reference page and the OpenAPI document.
// scripts/rename.sh rewrites this line when a fork is retargeted, so keep it a
// single const on one line.
const apiTitle = "Sonde API"

const apiDescription = "Uptime monitoring: probes, incidents, push heartbeats."

//go:embed migrations/*.sql
var migrationsFS embed.FS

func registry() documentation.Response {
	return documentation.Response{
		Modules: []documentation.Module{
			auth.Documentation,
			checker.Documentation,
			fleet.Documentation,
			httpapi.Documentation,
			httpapi.StatusPagesDocumentation,
			users.Documentation,
			webhooks.Documentation,
		},
	}
}

func refConfig() apiref.Config {
	return apiref.Config{
		Title:       apiTitle,
		Description: apiDescription,
		Servers:     []string{"/api"},
		Registry:    registry(),
	}
}

// routerDeps is what the router is assembled from. The engine and the
// publisher are built by run before the router exists — the checker has to be
// probing whether or not anything ever calls the API — so they arrive here
// rather than being constructed in place. Every field is optional: the route
// table test builds a router over nothing.
type routerDeps struct {
	DB        *gorm.DB
	SQL       *sql.DB
	Users     *kit.Store
	Engine    *checker.Engine
	Publisher *events.Publisher
	Deletions *directory.Deletions
	Env       env.Config
	Logger    *slog.Logger
}

// buildRouter assembles the router the server serves. It is a function rather
// than inline in main so a test can walk the real route table; it tolerates a
// nil database because no service touches the handle until a request arrives.
//
// One session manager owns the sessions table; the OIDC kit issues through it.
// Sign-in goes through Registre only, so no password kit is built.
func buildRouter(deps routerDeps) (chi.Router, error) {
	identity, err := buildIdentity(deps)
	if err != nil {
		return nil, err
	}
	clientDir := spa.DirFromEnv()
	router := httpx.NewRouter(httpx.Config{
		Logger:   deps.Logger,
		Security: securityConfig(clientDir, deps.Logger),
		CORS:     troncmiddleware.CORSConfig{AllowedOrigins: deps.Env.CORSAllowedOrigins},
	})
	health.Mount(router, health.DB(deps.SQL))
	router.Get("/version", versionHandler)
	apiref.Mount(router, refConfig())
	var mountErr error
	router.Route("/api", func(r chi.Router) { mountErr = mountAPI(r, deps, identity) })
	if mountErr != nil {
		return nil, mountErr
	}
	if spa.Available(clientDir) {
		router.Handle("/*", spa.Handler(spa.Config{Dir: clientDir}))
		deps.Logger.Info("serving client", slog.String("dir", clientDir))
	}
	return router, nil
}

// identity is the sign-in machinery the API routes are mounted behind.
type identity struct {
	sessions    *session.Manager
	kit         *oidc.Kit
	users       *kit.Store
	authService *auth.Service
}

func buildIdentity(deps routerDeps) (identity, error) {
	identityStore, err := newPorteStore(deps.SQL)
	if err != nil {
		return identity{}, fmt.Errorf("failed to open the porte store: %w", err)
	}
	resealInBackground(context.Background(), identityStore, deps.Logger)
	sessions, err := session.New(deps.Env.Porte, session.Deps{Sessions: identityStore.Sessions(), Logger: deps.Logger})
	if err != nil {
		return identity{}, fmt.Errorf("failed to configure sessions: %w", err)
	}
	users := deps.Users
	oidcKit, err := oidc.New(context.Background(), deps.Env.Porte, oidc.Deps{
		Users:      users,
		Identities: identityStore.Identities(),
		Codes:      identityStore.LoginCodes(),
		Sessions:   sessions,
		Erased:     erasedGuard(deps.Deletions),
		Logger:     deps.Logger,
	})
	if err != nil {
		return identity{}, fmt.Errorf("failed to configure authentication: %w", err)
	}
	if oidcKit.Enabled() {
		deps.Logger.Info("single sign-on enabled", slog.String("issuer", deps.Env.Porte.Issuer))
	}
	return identity{sessions: sessions, kit: oidcKit, users: users, authService: auth.NewService(deps.DB)}, nil
}

// mountAPI registers every /api route. The SPA catch-all is mounted after it,
// so nothing here can be shadowed.
func mountAPI(r chi.Router, deps routerDeps, id identity) error {
	db, appEnv := deps.DB, deps.Env
	requireAuth := kit.RequireAuth(id.sessions, id.users)
	webhookStore := webhooks.NewStore(db)
	if err := kit.Mount(r, kit.Config{Sessions: id.sessions, OIDC: id.kit}); err != nil {
		return err
	}
	auth.RegisterRoutes(r, id.authService, requireAuth)
	checker.RegisterHeartbeat(r, deps.Engine)
	users.RegisterRoutes(r, users.NewService(db), requireAuth)
	httpapi.RegisterRoutes(r, httpapi.New(db, httpapi.Deps{
		Monitors:         monitors.NewStore(db),
		Uptime:           uptime.NewService(db),
		Stats:            stats.NewService(db),
		Bus:              deps.Publisher,
		JournalURL:       appEnv.JournalURL,
		RawRetentionDays: appEnv.RawRetentionDays,
	}), requireAuth)
	webhooks.RegisterRoutes(r, webhookStore, webhooks.NewDispatcher(webhookStore, monitors.NewStore(db), deps.Logger), requireAuth)
	fleet.RegisterRoutes(r, fleet.NewStore(db), requireAuth, fleet.Config{Token: appEnv.FleetToken, Admins: appEnv.FleetAdmins, Logger: deps.Logger, Dispatch: fleet.Dispatcher{Token: appEnv.FleetDispatchToken}})
	return nil
}
