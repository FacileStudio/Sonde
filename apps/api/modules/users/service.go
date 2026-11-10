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

// updateName changes the display name. The email is not editable here: it
// comes from the identity provider on every sign-in, and the fleet allow-list
// trusts it.
func (service *Service) updateName(context context.Context, userID int64, name string) (*schemas.User, error) {
	result := service.orm.WithContext(context).
		Model(&schemas.User{}).
		Where("id = ?", userID).
		Update("name", name)
	if result.Error != nil {
		return nil, errors.Internal("failed to update user", result.Error)
	}
	return service.userByID(context, userID)
}
