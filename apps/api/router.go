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
	"github.com/FacileStudio/Sonde/apps/api/internal/httpapi"
	"github.com/FacileStudio/Sonde/apps/api/internal/middleware"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/stats"
	"github.com/FacileStudio/Sonde/apps/api/internal/uptime"
	"github.com/FacileStudio/Sonde/apps/api/internal/webhooks"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"
	"github.com/FacileStudio/Sonde/apps/api/modules/users"

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
// One session manager owns the sessions table; the OIDC kit issues through it.
// Sign-in goes through Registre only, so no password kit is built.
func buildRouter(deps routerDeps) (chi.Router, error) {
	db, sqlDB, appEnv, appLogger := deps.DB, deps.SQL, deps.Env, deps.Logger
	identityStore, err := newPorteStore(sqlDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open the porte store: %w", err)
	}
	resealInBackground(context.Background(), identityStore, appLogger)
	userStore := auth.NewUserStore(db)

	sessions, err := session.New(appEnv.Porte, session.Deps{
		Sessions: identityStore.Sessions(),
		Logger:   appLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure sessions: %w", err)
	}

	kit, err := oidc.New(context.Background(), appEnv.Porte, oidc.Deps{
		Users:      userStore,
		Identities: identityStore.Identities(),
		Codes:      identityStore.LoginCodes(),
		Sessions:   sessions,
		Logger:     appLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure authentication: %w", err)
	}
	if kit.Enabled() {
		appLogger.Info("single sign-on enabled", slog.String("issuer", appEnv.Porte.Issuer))
	}

	authService := auth.NewService(db)
	userService := users.NewService(db)
	httpService := httpapi.New(db, httpapi.Deps{
		Monitors:         monitors.NewStore(db),
		Uptime:           uptime.NewService(db),
		Stats:            stats.NewService(db),
		Bus:              deps.Publisher,
		JournalURL:       appEnv.JournalURL,
		RawRetentionDays: appEnv.RawRetentionDays,
	})

	clientDir := spa.DirFromEnv()
	router := httpx.NewRouter(httpx.Config{
		Logger:   appLogger,
		Security: securityConfig(clientDir, appLogger),
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins: appEnv.CORSAllowedOrigins,
		},
	})

	health.Mount(router, health.DB(sqlDB))
	router.Get("/version", versionHandler)
	apiref.Mount(router, refConfig())

	requireAuth := middleware.RequireAuth(sessions, authService)

	router.Route("/api", func(r chi.Router) {
		webhookStore := webhooks.NewStore(db)
		webhookDispatcher := webhooks.NewDispatcher(webhookStore, monitors.NewStore(db), appLogger)

		sessions.Mount(r)
		kit.Mount(r)
		auth.RegisterRoutes(r, authService, requireAuth)
		checker.RegisterHeartbeat(r, deps.Engine)
		users.RegisterRoutes(r, userService, requireAuth)
		httpapi.RegisterRoutes(r, httpService, requireAuth)
		webhooks.RegisterRoutes(r, webhookStore, webhookDispatcher, requireAuth)
	})

	if spa.Available(clientDir) {
		router.Handle("/*", spa.Handler(spa.Config{Dir: clientDir}))
		appLogger.Info("serving client", slog.String("dir", clientDir))
	}

	return router, nil
}
