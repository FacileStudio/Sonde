package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/authcrypto"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/database"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/env"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/ticketcode"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/schemas"

	"gorm.io/gorm"
)

const (
	seedEmail    = "demo@example.com"
	seedPassword = "password123"
	seedEvent    = "Demo Event"
)

func main() {
	env, err := env.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err != nil {
		logger.Error("failed to load env", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := database.Open(env.DatabaseURL)
	if err != nil {
		logger.Error("failed to open database", slog.Any("error", err))
		os.Exit(1)
	}

	if err := schemas.Migrate(db); err != nil {
		logger.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	user, created, err := ensureUser(db)
	if err != nil {
		logger.Error("failed to seed user", slog.Any("error", err))
		os.Exit(1)
	}

	event, eventCreated, err := ensureEvent(db, user)
	if err != nil {
		logger.Error("failed to seed event", slog.Any("error", err))
		os.Exit(1)
	}

	code, ticket, err := createTicket(db, event)
	if err != nil {
		logger.Error("failed to seed ticket", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("seed complete",
		slog.Bool("user_created", created),
		slog.Bool("event_created", eventCreated),
		slog.String("email", user.Email),
		slog.String("password", seedPassword),
		slog.Int64("event_id", event.ID),
		slog.String("event_name", event.Name),
		slog.Int64("ticket_id", ticket.ID),
		slog.String("ticket_code", code),
	)
}

func ensureUser(db *gorm.DB) (*schemas.User, bool, error) {
	var user schemas.User
	err := db.Where("email = ?", seedEmail).First(&user).Error
	if err == nil {
		return &user, false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	hash, err := authcrypto.HashPassword(seedPassword)
	if err != nil {
		return nil, false, err
	}

	user = schemas.User{
		Email:        seedEmail,
		PasswordHash: hash,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, false, err
	}
	return &user, true, nil
}

func ensureEvent(db *gorm.DB, user *schemas.User) (*schemas.Event, bool, error) {
	var event schemas.Event
	err := db.Where("name = ? AND owner_id = ?", seedEvent, fmt.Sprint(user.ID)).First(&event).Error
	if err == nil {
		return &event, false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	startsAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Minute)
	endsAt := startsAt.Add(3 * time.Hour)

	event = schemas.Event{
		Name:     seedEvent,
		StartsAt: startsAt,
		EndsAt:   &endsAt,
		OwnerID:  fmt.Sprint(user.ID),
	}
	if err := db.Create(&event).Error; err != nil {
		return nil, false, err
	}
	return &event, true, nil
}

func createTicket(db *gorm.DB, event *schemas.Event) (string, *schemas.Ticket, error) {
	code, codeHash, err := ticketcode.NewCode()
	if err != nil {
		return "", nil, err
	}

	ticket := schemas.Ticket{
		EventID:  event.ID,
		CodeHash: codeHash,
		Status:   "valid",
	}
	if err := db.Create(&ticket).Error; err != nil {
		return "", nil, err
	}
	return code, &ticket, nil
}
