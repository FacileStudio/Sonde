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

var _ porte.UserStore = (*UserStore)(nil)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a
// verified email, and creates the account when neither finds anything.
//
// The email fallback follows porte/pg's linking rules. An address the provider
// does not vouch for never reaches an existing account, because matching on it
// alone is an account-takeover primitive. A verified address links only when
// the account holds no identity from this provider yet: a second subject of the
// same issuer with the same address is a recycled address, not the owner, and
// is refused. The advisory lock keeps two simultaneous first sign-ins from both
// counting zero users and both becoming admin. The first account ever created
// becomes admin.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return 0, errors.Invalid("the identity provider returned no usable email")
	}
	profile := profileFrom(claims, email)

	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return errors.Internal("failed to acquire registration lock", err)
		}
		id, err := resolveAccount(tx, claims, profile)
		userID = id
		return err
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

type profile struct {
	email    string
	name     string
	avatar   string
	locale   string
	verified bool
}

func profileFrom(claims porte.Claims, email string) profile {
	return profile{
		email:    email,
		name:     claims.DisplayName(),
		avatar:   pictureURL(claims.Picture),
		locale:   claims.Locale,
		verified: claims.EmailVerified,
	}
}

// resolveAccount finds or creates the row a sign-in lands on and refreshes its
// profile. It runs inside the registration lock.
func resolveAccount(tx *gorm.DB, claims porte.Claims, p profile) (int64, error) {
	var linked int64
	err := tx.Raw(
		`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`,
		claims.Provider, claims.Subject,
	).Scan(&linked).Error
	if err != nil {
		return 0, errors.Internal("failed to resolve the identity", err)
	}
	if linked != 0 {
		return linked, refreshProfile(tx, linked, p)
	}

	var existing schemas.User
	err = tx.Where("email = ?", p.email).First(&existing).Error
	switch {
	case err == nil:
		return linkExisting(tx, existing.ID, claims.Provider, p)
	case !stderrors.Is(err, gorm.ErrRecordNotFound):
		return 0, errors.Internal("failed to look up the account", err)
	}
	return createAccount(tx, p)
}

// linkExisting applies the verified-email rule to an account found by address.
func linkExisting(tx *gorm.DB, userID int64, provider string, p profile) (int64, error) {
	if !p.verified {
		return 0, errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
	}
	var held int64
	err := tx.Raw(
		`SELECT count(*) FROM porte_identities WHERE user_id = ? AND provider = ?`, userID, provider,
	).Scan(&held).Error
	if err != nil {
		return 0, errors.Internal("failed to read the account's identities", err)
	}
	if held > 0 {
		return 0, errors.Conflict("this account is already linked to another identity from the same provider")
	}
	return userID, refreshProfile(tx, userID, p)
}

func createAccount(tx *gorm.DB, p profile) (int64, error) {
	var count int64
	if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
		return 0, errors.Internal("failed to count users", err)
	}
	user := schemas.User{
		Email:     p.email,
		Name:      p.name,
		IsAdmin:   count == 0,
		AvatarURL: p.avatar,
		Locale:    localeOrDefault(p.locale),
	}
	if err := tx.Create(&user).Error; err != nil {
		return 0, errors.Internal("failed to create the account", err)
	}
	return user.ID, nil
}

func localeOrDefault(locale string) string {
	if locale == "" {
		return porte.DefaultLocale
	}
	return locale
}

// refreshProfile keeps the name, picture and locale the provider asserts,
// which makes the identity provider the source of truth for them. The address
// is rewritten only when the provider vouches for it: an unverified claim must
// never move a stored address. An empty name or locale is not written over one
// the user already has: a provider that stops sending a claim should not blank
// the dashboard. The picture is always written, because Registre sends it for
// every human and a stale URL is worse than the initials.
func refreshProfile(tx *gorm.DB, userID int64, p profile) error {
	updates := map[string]any{"avatar_url": p.avatar}
	if p.verified {
		updates["email"] = p.email
	}
	if p.name != "" {
		updates["name"] = p.name
	}
	if p.locale != "" {
		updates["locale"] = p.locale
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
