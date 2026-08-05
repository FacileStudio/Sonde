package users

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/authcrypto"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

type Service struct {
	orm        *gorm.DB
	controller *Controller
}

func NewService(orm *gorm.DB) *Service {
	service := &Service{orm: orm}
	service.controller = newController(service)
	return service
}

func (service *Service) updateUser(context context.Context, userID string, email *string, password *string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	updates := map[string]any{}
	if email != nil {
		updates["email"] = *email
	}
	if password != nil {
		hash, err := authcrypto.HashPassword(*password)
		if err != nil {
			return nil, errors.Invalid("invalid password")
		}
		updates["password_hash"] = hash
	}

	result := service.orm.WithContext(context).
		Model(&schemas.User{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		if stderrors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("email already registered")
		}
		return nil, errors.Internal("failed to update user", result.Error)
	}
	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	return &User{
		ID:    strconv.FormatInt(record.ID, 10),
		Email: record.Email,
	}, nil
}
