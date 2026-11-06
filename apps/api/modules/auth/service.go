package auth

import (
	"context"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service resolves the account behind a porte session. Passwords no longer
// exist here: Registre signs everyone in, and porte owns the session.
type Service struct {
	orm *gorm.DB
}

// NewService returns the account reader over Sonde's users table.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
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
