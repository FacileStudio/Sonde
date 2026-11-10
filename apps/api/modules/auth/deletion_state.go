package auth

import (
	"context"
	"database/sql"
	stderrors "errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
