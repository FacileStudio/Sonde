// Router assembly, kept out of main.go so the file that owns the process
// lifecycle is not also the file that owns the route table.
package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/Sonde/apps/api/internal/httpapi"
	"github.com/FacileStudio/Sonde/apps/api/internal/middleware"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/stats"
	"github.com/FacileStudio/Sonde/apps/api/internal/uptime"
	"github.com/FacileStudio/Sonde/apps/api/internal/webhooks"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"
	"github.com/FacileStudio/Sonde/apps/api/modules/users"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
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
	Engine    *checker.Engine
	Publisher *events.Publisher
	Env       env.Config
	Logger    *slog.Logger
}

// buildRouter assembles the router the server serves. It is a function rather
// than inline in main so a test can walk the real route table; it tolerates a
// nil database because no service touches the handle until a request arrives.
//
// One session manager owns one sessions table, and both kits issue through it:
// a password login and a federated one are the same row, the same cookie and
// the same logout.
func buildRouter(deps routerDeps) (chi.Router, error) {
	db, sqlDB, appEnv, appLogger := deps.DB, deps.SQL, deps.Env, deps.Logger
	identityStore := portepg.New(sqlDB)
	userStore := auth.NewUserStore(db)

	sessions, err := session.New(appEnv.Porte, session.Deps{
		Sessions: identityStore.Sessions(),
		Logger:   appLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure sessions: %w", err)
	}

	kit, err := oidc.New(context.Background(), appEnv.Porte, oidc.Deps{
		Users:       userStore,
		Identities:  identityStore.Identities(),
		Codes:       identityStore.LoginCodes(),
		Sessions:    sessions,
		Logger:      appLogger,
		ConfigExtra: auth.ConfigExtra(appEnv.AllowRegistration),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure authentication: %w", err)
	}
	if kit.Enabled() {
		appLogger.Info("single sign-on enabled",
			slog.String("issuer", appEnv.Porte.Issuer),
			slog.Bool("sso_only", appEnv.Porte.SSOOnly))
	}

	passwords, err := local.New(local.Config{AllowRegistration: appEnv.AllowRegistration}, local.Deps{
		Users:      userStore,
		Identities: identityStore.Identities(),
		Sessions:   sessions,
		Logger:     appLogger,
		Count:      userStore.CountUsers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure the password login: %w", err)
	}

	authService := auth.NewService(db, passwords)
	userService := users.NewService(db)
	httpService := httpapi.New(db, httpapi.Deps{
		Monitors:         monitors.NewStore(db),
		Uptime:           uptime.NewService(db),
		Stats:            stats.NewService(db),
		Bus:              deps.Publisher,
		JournalURL:       appEnv.JournalURL,
		RawRetentionDays: appEnv.RawRetentionDays,
	})

	router := httpx.NewRouter(httpx.Config{
		Logger: appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins: appEnv.CORSAllowedOrigins,
		},
	})

	health.Mount(router, health.DB(sqlDB))
	apiref.Mount(router, refConfig())

	requireAuth := middleware.RequireAuth(sessions, authService)
	clearCookie := func(w http.ResponseWriter, r *http.Request) {
		sessions.ClearCookie(w, r, porte.SessionCookieName)
	}

	router.Route("/api", func(r chi.Router) {
		webhookStore := webhooks.NewStore(db)
		webhookDispatcher := webhooks.NewDispatcher(webhookStore, monitors.NewStore(db), appLogger)

		sessions.Mount(r)
		kit.Mount(r)
		auth.RegisterRoutes(r, authService, appEnv.Porte.SSOOnly, requireAuth, clearCookie)
		checker.RegisterHeartbeat(r, deps.Engine)
		users.RegisterRoutes(r, userService, requireAuth)
		httpapi.RegisterRoutes(r, httpService, requireAuth)
		webhooks.RegisterRoutes(r, webhookStore, webhookDispatcher, requireAuth)
	})

	clientDir := spa.DirFromEnv()
	if spa.Available(clientDir) {
		router.Handle("/*", spa.Handler(spa.Config{Dir: clientDir}))
		appLogger.Info("serving client", slog.String("dir", clientDir))
	}

	return router, nil
}
