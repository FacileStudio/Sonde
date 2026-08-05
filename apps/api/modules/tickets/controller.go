package tickets

import (
	"context"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) generateTicket(context context.Context, eventID int64) (*GenerateResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	event, err := controller.service.events.GetEvent(context, eventID)
	if err != nil {
		return nil, err
	}
	if event.OwnerID != identity.UserID {
		return nil, errors.Forbidden("only the event owner can generate tickets")
	}

	ticket, code, err := controller.service.generateTicket(context, eventID)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{Code: code, Ticket: *ticket}, nil
}

func (controller *Controller) validateTicket(context context.Context, req *ValidateRequest) (*ValidateResponse, error) {
	return controller.service.validateTicket(context, req.Code)
}

func (controller *Controller) checkInTicket(context context.Context, req *CheckInRequest) (*CheckInResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	return controller.service.checkInTicket(context, req.Code, identity.UserID)
}
