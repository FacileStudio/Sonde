package users

import (
	"context"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) me(context context.Context) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	user, err := controller.service.userByID(context, identity.UserID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{User: User{
		ID:      user.ID,
		Email:   user.Email,
		Name:    user.Name,
		IsAdmin: user.IsAdmin,
	}}, nil
}

func (controller *Controller) updateMe(context context.Context, req *UpdateRequest) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	var email *string
	if req.Email != nil {
		normalized := strings.TrimSpace(strings.ToLower(*req.Email))
		if normalized == "" || !strings.Contains(normalized, "@") {
			return nil, errors.Invalid("invalid email")
		}
		email = &normalized
	}
	name := strings.TrimSpace(deref(req.Name))
	if req.Name != nil && name == "" {
		return nil, errors.Invalid("invalid name")
	}

	if email == nil && req.Name == nil {
		return nil, errors.Invalid("at least one field must be provided")
	}

	user, err := controller.service.updateUser(context, identity.UserID, email, name)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: User{
		ID:      user.ID,
		Email:   user.Email,
		Name:    user.Name,
		IsAdmin: user.IsAdmin,
	}}, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
