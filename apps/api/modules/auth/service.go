package auth

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service exposes login, logout and the current-user endpoint, delegating
// password work to the wrapped porte local kit.
type Service struct {
	orm   *gorm.DB
	local *local.Kit
}

// NewService takes porte's local kit, which owns the passwords now: argon2id,
// the constant-time compare, the equalised timing on an unknown address, and
// the session the credential lands in. What is left here is what a Sonde user
// is and who is an administrator.
func NewService(orm *gorm.DB, passwords *local.Kit) *Service {
	return &Service{orm: orm, local: passwords}
}

func (s *Service) Register(ctx context.Context, w http.ResponseWriter, r *http.Request, email, name, password string) (*schemas.User, string, error) {
	userID, token, err := s.local.Register(ctx, w, r, email, name, password)
	if err != nil {
		return nil, "", err
	}
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (*schemas.User, string, error) {
	userID, token, err := s.local.Login(ctx, w, r, email, password)
	if err != nil {
		return nil, "", err
	}
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// IdentityForUser turns the user id porte authenticated into the identity the
// rest of Sonde reads. porte hands the middleware a session and a user id and
// stops there: it holds no email and no is_admin. A session pointing at a user
// that no longer exists — the foreign key cascades a delete — is still not an
// authenticated request and is refused.
func (s *Service) IdentityForUser(ctx context.Context, userID int64) (authcontext.Identity, error) {
	var out struct {
		ID      int64
		Email   string
		IsAdmin bool
	}
	err := s.orm.WithContext(ctx).
		Model(&schemas.User{}).
		Select("id", "email", "is_admin").
		Where("id = ?", userID).
		Scan(&out).Error
	if err != nil {
		return authcontext.Identity{}, errors.Internal("failed to load the account", err)
	}
	if out.ID == 0 {
		return authcontext.Identity{}, errors.Unauthorized("invalid auth token")
	}
	return authcontext.Identity{UserID: out.ID, Email: out.Email, IsAdmin: out.IsAdmin}, nil
}

func (s *Service) UserByID(ctx context.Context, id int64) (*schemas.User, error) {
	var user schemas.User
	if err := s.orm.WithContext(ctx).First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to load user", err)
	}
	return &user, nil
}
