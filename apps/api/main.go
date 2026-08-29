package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/Sonde/apps/api/internal/database"
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/Sonde/apps/api/internal/httpapi"
	"github.com/FacileStudio/Sonde/apps/api/internal/middleware"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/retention"
	"github.com/FacileStudio/Sonde/apps/api/internal/uptime"
	"github.com/FacileStudio/Sonde/apps/api/internal/webhooks"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"
	"github.com/FacileStudio/Sonde/apps/api/modules/users"

	"github.com/FacileStudio/Journal/sdk/journal"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/migrate"
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

// buildRouter assembles the router the server serves. It is a function rather
// than inline in main so a test can walk the real route table; it tolerates a
// nil database because no service touches the handle until a request arrives.
//
// One session manager owns one sessions table, and both kits issue through it:
// a password login and a federated one are the same row, the same cookie and
// the same logout.
func buildRouter(db *gorm.DB, sqlDB *sql.DB, engine *checker.Engine, appEnv env.Config, appLogger *slog.Logger) (chi.Router, error) {
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
	httpService := httpapi.New(db, monitors.NewStore(db), uptime.NewService(db))

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
		checker.RegisterHeartbeat(r, engine)
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

func main() { os.Exit(run()) }

// run is main's body so that every failure path can pick an exit code and
// still let the deferred cleanup happen. A boot that fails must exit non-zero:
// Dokploy reads a zero exit as a clean shutdown, so a failed migration or a bad
// config would otherwise look like a container that was asked to stop.
func run() int {
	if healthcheck.Handle(os.Args) {
		return 0
	}

	appEnv, err := env.Load()
	appLogger := logger.New(logger.Config{})
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		return 1
	}
	var journalClient *journal.Client
	appLogger = logger.New(logger.Config{
		Level: appEnv.LogLevel,
		Wrap: func(handler slog.Handler) slog.Handler {
			if appEnv.JournalURL == "" || appEnv.JournalToken == "" {
				return handler
			}
			journalClient = journal.New(journal.Config{URL: appEnv.JournalURL, Token: appEnv.JournalToken})
			return journal.NewHandler(journalClient, handler)
		},
	})
	if journalClient != nil {
		defer journalClient.Close()
	}

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		return 1
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		return 1
	}

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		appLogger.Error("failed to locate embedded migrations", slog.Any("error", err))
		return 1
	}
	migrateConfig := migrate.Config{DB: sqlDB, FS: migrations, Logger: appLogger}
	if handled, err := migrate.Command(context.Background(), os.Args, migrateConfig); handled {
		if err != nil {
			appLogger.Error("migrate", slog.Any("error", err))
			return 1
		}
		return 0
	}
	if err := migrate.Run(context.Background(), migrateConfig); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	backgroundCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	publisher := events.New(backgroundCtx, appEnv.Events, monitors.NewStore(db), appLogger)
	webhookStore := webhooks.NewStore(db)
	webhookDispatcher := webhooks.NewDispatcher(webhookStore, monitors.NewStore(db), appLogger)

	engine := checker.NewGorm(db, checker.WithLogger(appLogger))
	pruner := retention.New(db, appEnv.RawRetentionDays, appLogger)

	go func() {
		if err := engine.Run(backgroundCtx); err != nil {
			appLogger.Error("checker stopped", slog.Any("error", err))
		}
	}()
	go pruner.Loop(backgroundCtx)
	go func() {
		for {
			select {
			case <-backgroundCtx.Done():
				return
			case transition, open := <-engine.Transitions():
				if !open {
					return
				}
				if publisher != nil {
					publisher.Publish(backgroundCtx, transition)
				}
				webhookDispatcher.Dispatch(backgroundCtx, transition)
			}
		}
	}()

	router, err := buildRouter(db, sqlDB, engine, appEnv, appLogger)
	if err != nil {
		appLogger.Error("failed to build the router", slog.Any("error", err))
		return 1
	}

	addr := ":" + appEnv.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appLogger.Info("server starting", slog.String("addr", addr))
	select {
	case err := <-serverErrCh:
		if !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server stopped", slog.Any("error", err))
			return 1
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stopBackground()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			return 1
		}
		appLogger.Info("server stopped")
	}
	return 0
}
