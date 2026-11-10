package users

import (
	"context"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) me(context context.Context) (*MeResponse, error) {
	identity, ok := kit.FromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	user, err := controller.service.userByID(context, identity.UserID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{User: toUser(user)}, nil
}

func (controller *Controller) updateMe(context context.Context, req *UpdateRequest) (*MeResponse, error) {
	identity, ok := kit.FromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	if req.Email != nil {
		return nil, errors.Invalid("the email comes from your Registre account and cannot be changed here")
	}
	if req.Name == nil {
		return nil, errors.Invalid("name is required")
	}
	name := strings.TrimSpace(*req.Name)
	if name == "" {
		return nil, errors.Invalid("invalid name")
	}

	user, err := controller.service.updateName(context, identity.UserID, name)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: toUser(user)}, nil
}

// toUser is the one place a stored account becomes the users API's shape.
func toUser(user *schemas.User) User {
	return User{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsAdmin:   user.IsAdmin,
		AvatarURL: user.AvatarURL,
		Locale:    user.Locale,
	}
}
