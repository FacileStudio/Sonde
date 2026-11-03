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
	avatar := pictureURL(claims.Picture)

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
			return refreshProfile(tx, linked, email, name, avatar)
		}

		var existing schemas.User
		err = tx.Where("email = ?", email).First(&existing).Error
		switch {
		case err == nil:
			if !claims.EmailVerified {
				return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
			}
			userID = existing.ID
			return refreshProfile(tx, existing.ID, email, name, avatar)
		case !stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Internal("failed to look up the account", err)
		}

		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}

		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0, AvatarURL: avatar}
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

// refreshProfile keeps the address, name and picture the provider asserts,
// which makes the identity provider the source of truth for all three. An
// empty name is not written over one the user already has: a provider that
// stops sending a claim should not blank the dashboard. The picture is always
// written, because Registre sends it for every human and a stale URL is worse
// than the initials.
func refreshProfile(tx *gorm.DB, userID int64, email, name, avatar string) error {
	updates := map[string]any{"email": email, "avatar_url": avatar}
	if name != "" {
		updates["name"] = name
	}
	if err := tx.Model(&schemas.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return errors.Internal("failed to update the account", err)
	}
	return nil
}

// pictureURL keeps the picture claim only when it is an https URL. Registre
// publishes a stable https://sso.facile.studio/avatars/<sub> for every human;
// anything else (a data: placeholder, plain http) is not something a browser on
// this origin should be pointed at, so it reads as no picture and the client
// draws the initial.
func pictureURL(claim string) string {
	if strings.HasPrefix(strings.ToLower(claim), "https://") {
		return claim
	}
	return ""
}

// BackfillAvatars gives every account linked to issuer the picture URL it
// would have received at its next sign-in, so a user who signed in before
// Sonde stored the claim sees their picture without signing in again.
//
// It only fills empty rows, so it is idempotent and never overwrites what a
// sign-in wrote. The URL is Registre's stable <issuer>/avatars/<subject>, which
// answers 404 for a user with no picture; the client falls back to the
// initial. A non-https issuer is skipped for the same reason pictureURL
// refuses one.
func (s *UserStore) BackfillAvatars(ctx context.Context, issuer string) (int64, error) {
	prefix := pictureURL(strings.TrimRight(issuer, "/") + "/avatars/")
	if prefix == "" {
		return 0, nil
	}
	result := s.orm.WithContext(ctx).Exec(
		`UPDATE users SET avatar_url = ? || porte_identities.subject
		   FROM porte_identities
		  WHERE porte_identities.user_id = users.id
		    AND porte_identities.provider = ?
		    AND users.avatar_url = ''`,
		prefix, issuer,
	)
	if result.Error != nil {
		return 0, errors.Internal("failed to backfill avatars", result.Error)
	}
	return result.RowsAffected, nil
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
