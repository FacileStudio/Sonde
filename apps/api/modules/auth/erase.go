package auth

import (
	"context"
	"database/sql"
	stderrors "errors"

	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EraseAccount deletes the account behind a (provider, subject) identity and
// everything it owns: its monitors, with their checks, incidents and status
// page memberships, its webhooks, its sessions and its login codes. Status
// pages themselves are administrator-owned and stay.
//
// It is the OnPersonDeleted callback of porte's deletion receiver, which
// delivers at least once, so an identity that is already gone is a success.
// The callback runs before porte erases its own rows, which is why the account
// can still be resolved from porte_identities here.
func (s *UserStore) EraseAccount(ctx context.Context, provider, subject string) error {
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var userID int64
		row := tx.Raw(`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`,
			provider, subject).Row()
		if err := row.Scan(&userID); err != nil {
			if stderrors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if err := tx.Exec(`DELETE FROM monitors WHERE owner_id = ?`, userID).Error; err != nil {
			return err
		}
		return tx.Exec(`DELETE FROM users WHERE id = ?`, userID).Error
	})
	if err != nil {
		return errors.Internal("failed to erase the account", err)
	}
	return nil
}

const deletionStateKey = "deletions"

// DeletionState persists the deletion receiver's state in directory_state, as
// one row. It satisfies porte/directory's StateStore.
type DeletionState struct {
	orm *gorm.DB
}

// NewDeletionState returns the state store over Sonde's database.
func NewDeletionState(orm *gorm.DB) *DeletionState {
	return &DeletionState{orm: orm}
}

// Load returns the saved blob, or nil when nothing was saved yet.
func (d *DeletionState) Load(ctx context.Context) ([]byte, error) {
	var state []byte
	row := d.orm.WithContext(ctx).Raw(`SELECT state FROM directory_state WHERE key = ?`, deletionStateKey).Row()
	if err := row.Scan(&state); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

// Save replaces the saved blob.
func (d *DeletionState) Save(ctx context.Context, state []byte) error {
	row := map[string]any{"key": deletionStateKey, "state": state, "updated_at": gorm.Expr("now()")}
	return d.orm.WithContext(ctx).Table("directory_state").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"state", "updated_at"}),
	}).Create(row).Error
}
