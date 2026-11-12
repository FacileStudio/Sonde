package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/modules/auth"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	portepg "github.com/FacileStudio/porte/pg"
	"gorm.io/gorm"
)

// newDeletions builds the GDPR receiver: it polls Registre for erased people,
// erases each one's Sonde account through the user store, then porte's own
// identity rows. It reuses the OIDC client credentials the sign-in already
// holds. A subject it already handled is erased again when an identity holds
// it once more. It returns nil when no issuer is configured, which is a dev setup
// with no Registre to ask.
func newDeletions(db *gorm.DB, sqlDB *sql.DB, appEnv env.Config, logger *slog.Logger) *directory.Deletions {
	if !appEnv.Porte.Enabled() {
		return nil
	}
	identities := portepg.New(sqlDB)
	return &directory.Deletions{
		Client: directory.Client{
			Issuer:       appEnv.Porte.Issuer,
			ClientID:     appEnv.Porte.ClientID,
			ClientSecret: appEnv.Porte.ClientSecret,
		},
		OnPersonDeleted: auth.NewUserStore(sqlDB).EraseAccount,
		Erase:           identities.EraseIdentity,
		Holds:           identities.HoldsIdentity,
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

// erasedGuard is the ledger the sign-in asks before it creates an account. A
// nil receiver gives no guard rather than an interface holding a nil pointer.
func erasedGuard(deletions *directory.Deletions) porte.ErasedSubjects {
	if deletions == nil {
		return nil
	}
	return deletions
}
