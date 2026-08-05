package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/authcrypto"
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

func (service *Service) registerUser(context context.Context, email string, password string) (userID string, token string, err error) {
	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		return "", "", errors.Invalid("invalid password")
	}

	record := &schemas.User{
		Email:        email,
		PasswordHash: hash,
	}
	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return "", "", errors.Conflict("email already registered")
		}
		return "", "", errors.Internal("failed to create user", err)
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}

	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) loginUser(context context.Context, email string, password string) (userID string, token string, err error) {
	var record schemas.User
	err = service.orm.WithContext(context).Where("email = ?", email).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", errors.Unauthorized("invalid credentials")
	}
	if err != nil {
		return "", "", errors.Internal("failed to read user", err)
	}
	if !authcrypto.VerifyPassword(password, record.PasswordHash) {
		return "", "", errors.Unauthorized("invalid credentials")
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}

	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) insertSession(context context.Context, token string, userID int64) error {
	record := &schemas.Session{
		Token:     hashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		return errors.Internal("failed to persist session", err)
	}
	return nil
}

func normalizeBearer(authorization string) string {
	value := strings.TrimSpace(authorization)
	if len(value) >= 7 && strings.EqualFold(value[:7], "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func (service *Service) authenticateRequest(context context.Context, authorization string) (string, *Data, error) {
	token := normalizeBearer(authorization)
	if token == "" {
		return "", nil, errors.Unauthorized("missing auth token")
	}

	var out struct {
		UserID    int64
		Email     string
		ExpiresAt time.Time
	}
	err := service.orm.WithContext(context).
		Table("sessions s").
		Select("u.id as user_id, u.email as email, s.expires_at as expires_at").
		Joins("join users u on u.id = s.user_id").
		Where("s.token = ?", hashToken(token)).
		Scan(&out).Error
	if err != nil {
		return "", nil, errors.Internal("failed to validate auth token", err)
	}
	if out.UserID == 0 {
		return "", nil, errors.Unauthorized("invalid auth token")
	}
	if time.Now().After(out.ExpiresAt) {
		return "", nil, errors.Unauthorized("expired auth token")
	}

	return strconv.FormatInt(out.UserID, 10), &Data{Email: out.Email}, nil
}

func (service *Service) Authenticate(context context.Context, authorization string) (string, any, error) {
	return service.authenticateRequest(context, authorization)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
