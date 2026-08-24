package auth

import (
	"context"
	stderrors "errors"
	"net/mail"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// registrationLockKey is the advisory lock the account rules are taken under.
// porte cannot take it — it does not own this database — which is why the
// first-account rule lives in this package's UserStore.
const registrationLockKey = 0x736F6E6465

// UserStore is the adapter Sonde hands to the porte kits so their password and
// federated flows land in this app's users table. porte ships pg's own
// porte_users, but is_admin and monitors.owner_id's target are app business,
// so the app owns the write — that is what porte.UserStore is the escape hatch
// for.
type UserStore struct {
	orm *gorm.DB
}

func NewUserStore(orm *gorm.DB) *UserStore {
	return &UserStore{orm: orm}
}

var (
	_ porte.UserStore         = (*UserStore)(nil)
	_ porte.PasswordUserStore = (*UserStore)(nil)
)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a
// verified email, and creates the account when neither finds anything.
//
// Matching an existing account on the address alone is an account-takeover
// primitive when the provider does not vouch for it, so an unverified claim
// may only create a new row. The advisory lock keeps two simultaneous first
// sign-ins from both counting zero users and both becoming admin. The first
// account ever created becomes admin.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return 0, errors.Invalid("the identity provider returned no usable email")
	}
	name := claims.DisplayName()

	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return errors.Internal("failed to acquire registration lock", err)
		}

		var linked int64
		err := tx.Raw(
			`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`,
			claims.Provider, claims.Subject,
		).Scan(&linked).Error
		if err != nil {
			return errors.Internal("failed to resolve the identity", err)
		}
		if linked != 0 {
			userID = linked
			return refreshProfile(tx, linked, email, name)
		}

		var existing schemas.User
		err = tx.Where("email = ?", email).First(&existing).Error
		switch {
		case err == nil:
			if !claims.EmailVerified {
				return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
			}
			userID = existing.ID
			return refreshProfile(tx, existing.ID, email, name)
		case !stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Internal("failed to look up the account", err)
		}

		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}

		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0}
		if err := tx.Create(&user).Error; err != nil {
			return errors.Internal("failed to create the account", err)
		}
		userID = user.ID
		return nil
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

// refreshProfile keeps the address and name the provider asserts, which makes
// the identity provider the source of truth for both. An empty name is not
// written over one the user already has: a provider that stops sending a claim
// should not blank the dashboard.
func refreshProfile(tx *gorm.DB, userID int64, email, name string) error {
	updates := map[string]any{"email": email}
	if name != "" {
		updates["name"] = name
	}
	if err := tx.Model(&schemas.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return errors.Internal("failed to update the account", err)
	}
	return nil
}

// CreateFromPassword creates a user row for a local registration. porte has
// already validated the address and hashed the password; what is left is the
// part porte has no opinion about: the first account created becomes the
// administrator, under the same lock UpsertFromOIDC takes.
func (s *UserStore) CreateFromPassword(ctx context.Context, email, name string) (int64, error) {
	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return errors.Internal("failed to acquire registration lock", err)
		}
		var existing int64
		if err := tx.Model(&schemas.User{}).Where("email = ?", email).Count(&existing).Error; err != nil {
			return errors.Internal("failed to check email", err)
		}
		if existing > 0 {
			return errors.Conflict("an account with this email already exists")
		}
		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}
		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0}
		if err := tx.Create(&user).Error; err != nil {
			if stderrors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.Conflict("an account with this email already exists")
			}
			return errors.Internal("failed to create the account", err)
		}
		userID = user.ID
		return nil
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

// FindByEmail returns the user id for an address, or porte.ErrNotFound.
func (s *UserStore) FindByEmail(ctx context.Context, email string) (int64, error) {
	var user schemas.User
	err := s.orm.WithContext(ctx).Select("id").Where("email = ?", email).First(&user).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return 0, porte.ErrNotFound
	}
	if err != nil {
		return 0, errors.Internal("failed to look up the account", err)
	}
	return user.ID, nil
}

// CountUsers backs porte/local's registration gate and the first-account rule.
func (s *UserStore) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := s.orm.WithContext(ctx).Model(&schemas.User{}).Count(&count).Error; err != nil {
		return 0, errors.Internal("failed to count users", err)
	}
	return count, nil
}

// ConfigExtra is Sonde's addition to GET /auth/config. porte serves sso_only
// and oidc_enabled there; allow_registration is Sonde's own and the login
// screen reads it to decide whether to draw the register form.
func ConfigExtra(allowRegistration bool) func() map[string]any {
	return func() map[string]any {
		return map[string]any{"allow_registration": allowRegistration}
	}
}
