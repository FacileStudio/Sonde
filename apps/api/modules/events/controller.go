package events

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

func (controller *Controller) createEvent(context context.Context, req *CreateRequest) (*CreateResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.Invalid("name required")
	}
	if req.Starts.IsZero() {
		return nil, errors.Invalid("starts required")
	}
	if !req.Ends.IsZero() && req.Ends.Before(req.Starts) {
		return nil, errors.Invalid("ends must be after starts")
	}

	id, err := controller.service.createEvent(context, name, req.Starts, req.Ends, identity.UserID)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{ID: id}, nil
}

func (controller *Controller) listEvents(context context.Context) (*ListResponse, error) {
	events, err := controller.service.listEvents(context)
	if err != nil {
		return nil, err
	}
	return &ListResponse{Events: events}, nil
}

func (controller *Controller) getEvent(context context.Context, id int64) (*Event, error) {
	return controller.service.getEvent(context, id)
}
