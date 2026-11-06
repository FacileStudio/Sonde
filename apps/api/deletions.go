package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"

	"github.com/FacileStudio/porte/directory"
	portepg "github.com/FacileStudio/porte/pg"
	"gorm.io/gorm"
)

// newDeletions builds the GDPR receiver: it polls Registre for erased people,
// erases each one's Sonde account through the user store, then porte's own
// identity rows. It reuses the OIDC client credentials the sign-in already
// holds. It returns nil when no issuer is configured, which is a dev setup
// with no Registre to ask.
func newDeletions(db *gorm.DB, sqlDB *sql.DB, appEnv env.Config, logger *slog.Logger) *directory.Deletions {
	if !appEnv.Porte.Enabled() {
		return nil
	}
	return &directory.Deletions{
		Client: directory.Client{
			Issuer:       appEnv.Porte.Issuer,
			ClientID:     appEnv.Porte.ClientID,
			ClientSecret: appEnv.Porte.ClientSecret,
		},
		OnPersonDeleted: auth.NewUserStore(db).EraseAccount,
		Erase:           portepg.New(sqlDB).EraseIdentity,
		State:           auth.NewDeletionState(db),
		Logger:          logger,
	}
}

// runDeletions polls until ctx ends. Sonde runs as a single replica, which is
// what directory.Deletions.Run requires.
func runDeletions(ctx context.Context, deletions *directory.Deletions) {
	if deletions == nil {
		return
	}
	go deletions.Run(ctx)
}
