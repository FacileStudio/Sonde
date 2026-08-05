package users

import (
	"context"
	"strings"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/authcontext"
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

	return &MeResponse{User: User{
		ID:    identity.UserID,
		Email: identity.Email,
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

	var password *string
	if req.Password != nil {
		if len(*req.Password) < 8 {
			return nil, errors.Invalid("password must be at least 8 characters")
		}
		password = req.Password
	}

	if email == nil && password == nil {
		return nil, errors.Invalid("at least one field must be provided")
	}

	user, err := controller.service.updateUser(context, identity.UserID, email, password)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: *user}, nil
}
