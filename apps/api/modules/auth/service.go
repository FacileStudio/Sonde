package auth

import (
	"context"

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
