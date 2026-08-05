package events

import (
	"context"
	stderrors "errors"
	"time"

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

func (service *Service) createEvent(context context.Context, name string, starts time.Time, ends time.Time, ownerID string) (int64, error) {
	record := &schemas.Event{
		Name:     name,
		StartsAt: starts,
		OwnerID:  ownerID,
	}
	if !ends.IsZero() {
		record.EndsAt = &ends
	}

	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		return 0, errors.Internal("failed to create event", err)
	}
	return record.ID, nil
}

func (service *Service) listEvents(context context.Context) ([]*Event, error) {
	var records []schemas.Event
	if err := service.orm.WithContext(context).Order("id desc").Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list events", err)
	}

	out := make([]*Event, 0, len(records))
	for _, record := range records {
		event := &Event{
			ID:      record.ID,
			Name:    record.Name,
			Starts:  record.StartsAt,
			OwnerID: record.OwnerID,
		}
		if record.EndsAt != nil {
			event.Ends = *record.EndsAt
		}
		out = append(out, event)
	}
	return out, nil
}

func (service *Service) getEvent(context context.Context, id int64) (*Event, error) {
	var record schemas.Event
	err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound("event not found")
	}
	if err != nil {
		return nil, errors.Internal("failed to fetch event", err)
	}

	event := &Event{
		ID:      record.ID,
		Name:    record.Name,
		Starts:  record.StartsAt,
		OwnerID: record.OwnerID,
	}
	if record.EndsAt != nil {
		event.Ends = *record.EndsAt
	}
	return event, nil
}

func (service *Service) GetEvent(context context.Context, id int64) (*Event, error) {
	return service.getEvent(context, id)
}
