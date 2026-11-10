package auth

import (
	"context"
	"database/sql"

	"github.com/FacileStudio/porte/kit"
)

// NewUserStore returns the porte user store over Sonde's users table. The
// first account becomes admin, and erasing an account takes its monitors
// with it; their checks, incidents and status page memberships cascade.
func NewUserStore(sqlDB *sql.DB) *kit.Store {
	return &kit.Store{DB: sqlDB, Columns: kit.Columns{Admin: "is_admin"}, EraseData: eraseOwned}
}

func eraseOwned(ctx context.Context, tx *sql.Tx, userID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM monitors WHERE owner_id = $1`, userID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE users SET is_admin = true
		 WHERE id = (SELECT min(id) FROM users WHERE id <> $1)
		   AND NOT EXISTS (SELECT 1 FROM users WHERE is_admin AND id <> $1)`, userID)
	return err
}
