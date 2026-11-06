package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/Sonde/apps/api/internal/database"
	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/internal/retention"
	"github.com/FacileStudio/Sonde/apps/api/internal/webhooks"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"

	"github.com/FacileStudio/Journal/sdk/journal"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/logger"
	"github.com/FacileStudio/tronc/migrate"

	"gorm.io/gorm"
)

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
	backfillAvatars(db, appEnv.Porte.Issuer, appLogger)
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	backgroundCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	// Dialing happens in the background: a bus that is slow or down must not
	// hold up the server that reports on it, and Sonde and Antenne restart
	// together on a suite deploy.
	publisher := events.New(appEnv.Events, monitors.NewStore(db), appLogger)
	publisher.Start(backgroundCtx)
	defer publisher.Close()

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
	runDeletions(backgroundCtx, newDeletions(db, sqlDB, appEnv, appLogger))
	go func() {
		for {
			select {
			case <-backgroundCtx.Done():
				return
			case transition, open := <-engine.Transitions():
				if !open {
					return
				}
				publisher.Publish(backgroundCtx, transition)
				webhookDispatcher.Dispatch(backgroundCtx, transition)
			}
		}
	}()

	router, err := buildRouter(routerDeps{
		DB:        db,
		SQL:       sqlDB,
		Engine:    engine,
		Publisher: publisher,
		Env:       appEnv,
		Logger:    appLogger,
	})
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

// backfillAvatars fills the picture URL of every account that signed in before
// Sonde stored the claim. A failure is logged and boot goes on: a missing
// picture draws the initial, which is no reason to refuse to monitor anything.
func backfillAvatars(db *gorm.DB, issuer string, logger *slog.Logger) {
	filled, err := auth.NewUserStore(db).BackfillAvatars(context.Background(), issuer)
	if err != nil {
		logger.Warn("avatar backfill failed", slog.Any("error", err))
		return
	}
	if filled > 0 {
		logger.Info("avatar backfill", slog.Int64("users", filled))
	}
}
