package tickets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"strings"
	"time"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/ticketcode"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/modules/events"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	orm        *gorm.DB
	controller *Controller
	events     *events.Service
}

func NewService(orm *gorm.DB, eventSvc *events.Service) *Service {
	service := &Service{orm: orm, events: eventSvc}
	service.controller = newController(service)
	return service
}

func (service *Service) generateTicket(context context.Context, eventID int64) (ticket *Ticket, code string, err error) {
	for range 3 {
		code, hash, err := ticketcode.NewCode()
		if err != nil {
			return nil, "", errors.Internal("failed to generate ticket", err)
		}

		record := &schemas.Ticket{EventID: eventID, CodeHash: hash, Status: "valid"}
		if err := service.orm.WithContext(context).Create(record).Error; err != nil {
			if stderrors.Is(err, gorm.ErrDuplicatedKey) {
				continue
			}
			return nil, "", errors.Internal("failed to persist ticket", err)
		}
		return &Ticket{ID: record.ID, EventID: record.EventID, Status: record.Status, UsedAt: record.UsedAt}, code, nil
	}

	return nil, "", errors.Internal("failed to generate unique ticket", nil)
}

func (service *Service) validateTicket(context context.Context, code string) (*ValidateResponse, error) {
	hash := hashFromCode(code)
	if hash == "" {
		return nil, errors.Invalid("code required")
	}

	var record schemas.Ticket
	err := service.orm.WithContext(context).Where("code_hash = ?", hash).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return &ValidateResponse{Valid: false, Status: "not_found", Ticket: Ticket{}}, nil
	}
	if err != nil {
		return nil, errors.Internal("failed to validate ticket", err)
	}

	ticket := Ticket{ID: record.ID, EventID: record.EventID, Status: record.Status, UsedAt: record.UsedAt}
	return &ValidateResponse{
		Valid:  record.Status == "valid",
		Status: record.Status,
		Ticket: ticket,
	}, nil
}

func (service *Service) checkInTicket(context context.Context, code string, actorUserID string) (*CheckInResponse, error) {
	hash := hashFromCode(code)
	if hash == "" {
		return nil, errors.Invalid("code required")
	}

	var out Ticket
	err := service.orm.WithContext(context).Transaction(func(tx *gorm.DB) error {
		var record struct {
			schemas.Ticket
			OwnerID string
		}
		err := tx.Table("tickets").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("tickets.id, tickets.event_id, tickets.code_hash, tickets.status, tickets.used_at, tickets.created_at, events.owner_id").
			Joins("join events on events.id = tickets.event_id").
			Where("tickets.code_hash = ?", hash).
			First(&record).Error
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("ticket not found")
		}
		if err != nil {
			return errors.Internal("failed to read ticket", err)
		}
		if record.OwnerID != actorUserID {
			return errors.Forbidden("only the event owner can check in tickets")
		}

		if record.Status == "used" {
			return errors.Failed("ticket already used")
		}
		if record.Status != "valid" {
			return errors.Failed("ticket is not valid")
		}

		now := time.Now()
		record.Status = "used"
		record.UsedAt = &now
		if err := tx.Model(&schemas.Ticket{}).
			Where("id = ?", record.ID).
			Updates(map[string]any{
				"status":  record.Status,
				"used_at": record.UsedAt,
			}).Error; err != nil {
			return errors.Internal("failed to check in ticket", err)
		}

		out = Ticket{ID: record.ID, EventID: record.EventID, Status: record.Status, UsedAt: record.UsedAt}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &CheckInResponse{Ticket: out}, nil
}

func hashFromCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
