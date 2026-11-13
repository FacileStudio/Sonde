package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
)

// UserStore is porte/kit's store over Sonde's users table, plus the one claim
// the kit does not know: the GitHub account Registre reports under
// connections, which decides the projects a person may link a monitor to.
type UserStore struct {
	*kit.Store
}

var _ porte.UserStore = (*UserStore)(nil)

// NewUserStore returns the porte user store over Sonde's users table, built by
// kit.NewStore, which reads the table's columns once and fails when the
// migrations have not run or a column the store names is missing. The first
// account becomes admin, and erasing an account takes its monitors with it;
// their checks, incidents and status page memberships cascade. The GitHub id
// lives on the users row, so it goes with the account.
func NewUserStore(ctx context.Context, sqlDB *sql.DB) (*UserStore, error) {
	store, err := kit.NewStore(ctx, kit.Store{DB: sqlDB, Columns: kit.Columns{Admin: "is_admin"}, EraseData: eraseOwned})
	if err != nil {
		return nil, err
	}
	return &UserStore{Store: store}, nil
}

// UpsertFromOIDC applies the kit's sign-in rules, then stores the GitHub id
// when the claims carry one; an empty github_id means no account is linked. A sign-in without the claim leaves a stored id
// alone: Registre omits connections when its client lacks the scope.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	userID, err := s.Store.UpsertFromOIDC(ctx, claims)
	if err != nil {
		return 0, err
	}
	githubID := extractGitHubID(claims.Extra)
	if githubID == "" {
		return userID, nil
	}
	if _, err := s.DB.ExecContext(ctx, `UPDATE users SET github_id = $1 WHERE id = $2`, githubID, userID); err != nil {
		return 0, errors.Internal("failed to store the github account", err)
	}
	return userID, nil
}

func extractGitHubID(extra map[string]json.RawMessage) string {
	raw, ok := extra["connections"]
	if !ok || len(raw) == 0 {
		return ""
	}
	var connections struct {
		GitHub string `json:"github"`
	}
	if err := json.Unmarshal(raw, &connections); err != nil {
		return ""
	}
	return strings.TrimSpace(connections.GitHub)
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
