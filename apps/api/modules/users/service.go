package users

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
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

func (service *Service) userByID(context context.Context, userID int64) (*schemas.User, error) {
	var record schemas.User
	if err := service.orm.WithContext(context).First(&record, userID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}
	return &record, nil
}

func (service *Service) updateUser(context context.Context, userID int64, email *string, name string) (*schemas.User, error) {
	updates := map[string]any{}
	if email != nil {
		updates["email"] = *email
	}
	if name != "" {
		updates["name"] = name
	}

	result := service.orm.WithContext(context).
		Model(&schemas.User{}).
		Where("id = ?", userID).
		Updates(updates)
	if result.Error != nil {
		if stderrors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("email already registered")
		}
		return nil, errors.Internal("failed to update user", result.Error)
	}
	return service.userByID(context, userID)
}
