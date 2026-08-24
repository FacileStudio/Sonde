// Package monitors owns the monitor domain: validation per check type and the
// CRUD store over GORM.
package monitors

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	TypeHTTP = "http"
	TypeTCP  = "tcp"
	TypePush = "push"

	MinIntervalSeconds = 20
	MaxIntervalSeconds = 24 * 60 * 60

	DefaultIntervalSeconds = 60
	DefaultTimeoutSeconds  = 10
	DefaultExpectedStatus  = 200
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Input struct {
	Slug            string
	Name            string
	Type            string
	Target          string
	IntervalSeconds int
	TimeoutSeconds  int
	ExpectedStatus  int
	ExpectedKeyword string
	Enabled         bool
}

func (in *Input) normalize() {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	in.Type = strings.TrimSpace(in.Type)
	in.Target = strings.TrimSpace(in.Target)
	if in.IntervalSeconds == 0 {
		in.IntervalSeconds = DefaultIntervalSeconds
	}
	if in.TimeoutSeconds == 0 {
		in.TimeoutSeconds = DefaultTimeoutSeconds
	}
	if in.ExpectedStatus == 0 {
		in.ExpectedStatus = DefaultExpectedStatus
	}
}

func Validate(in *Input) error {
	in.normalize()

	switch {
	case !slugPattern.MatchString(in.Slug) || len(in.Slug) > 100:
		return errors.Invalid("slug must be lowercase letters, digits and dashes")
	case in.Name == "" || len(in.Name) > 200:
		return errors.Invalid("name must be between 1 and 200 characters")
	case in.IntervalSeconds < MinIntervalSeconds || in.IntervalSeconds > MaxIntervalSeconds:
		return errors.Invalid(fmt.Sprintf(
			"interval must be between %d and %d seconds", MinIntervalSeconds, MaxIntervalSeconds))
	case in.TimeoutSeconds < 1 || in.TimeoutSeconds >= in.IntervalSeconds:
		return errors.Invalid("timeout must be at least 1 second and shorter than the interval")
	}

	switch in.Type {
	case TypeHTTP:
		parsed, err := url.Parse(in.Target)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errors.Invalid("target must be an http or https URL")
		}
		if in.ExpectedStatus < 100 || in.ExpectedStatus > 599 {
			return errors.Invalid("expected_status must be a valid HTTP status code")
		}
	case TypeTCP:
		host, portText, err := net.SplitHostPort(in.Target)
		if err != nil || host == "" {
			return errors.Invalid("target must be host:port")
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return errors.Invalid("target port must be between 1 and 65535")
		}
	case TypePush:
		if in.Target != "" {
			return errors.Invalid("push monitors take no target; the push token is the endpoint")
		}
	default:
		return errors.Invalid("type must be one of http, tcp, push")
	}

	return nil
}

func newPushToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) Create(ctx context.Context, ownerID int64, in Input) (*schemas.Monitor, error) {
	if err := Validate(&in); err != nil {
		return nil, err
	}

	monitor := &schemas.Monitor{
		OwnerID:         ownerID,
		Slug:            in.Slug,
		Name:            in.Name,
		Type:            in.Type,
		Target:          in.Target,
		IntervalSeconds: in.IntervalSeconds,
		TimeoutSeconds:  in.TimeoutSeconds,
		ExpectedStatus:  in.ExpectedStatus,
		ExpectedKeyword: in.ExpectedKeyword,
		Enabled:         true,
	}
	if monitor.Type == TypePush {
		token, err := newPushToken()
		if err != nil {
			return nil, errors.Internal("could not generate push token", err)
		}
		monitor.PushToken = &token
	}

	if err := s.db.WithContext(ctx).Create(monitor).Error; err != nil {
		return nil, errors.Conflict("slug already exists")
	}
	return monitor, nil
}

func (s *Store) Update(ctx context.Context, ownerID int64, id int64, in Input) (*schemas.Monitor, error) {
	if err := Validate(&in); err != nil {
		return nil, err
	}

	monitor, err := s.get(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}

	monitor.Slug = in.Slug
	monitor.Name = in.Name
	monitor.Type = in.Type
	monitor.Target = in.Target
	monitor.IntervalSeconds = in.IntervalSeconds
	monitor.TimeoutSeconds = in.TimeoutSeconds
	monitor.ExpectedStatus = in.ExpectedStatus
	monitor.ExpectedKeyword = in.ExpectedKeyword
	monitor.Enabled = in.Enabled
	monitor.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(monitor).Error; err != nil {
		return nil, errors.Conflict("slug already exists")
	}
	return monitor, nil
}

func (s *Store) Delete(ctx context.Context, ownerID int64, id int64) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		Delete(&schemas.Monitor{})
	if result.Error != nil {
		return errors.Internal("could not delete monitor", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("monitor not found")
	}
	return nil
}

func (s *Store) Get(ctx context.Context, ownerID int64, id int64) (*schemas.Monitor, error) {
	return s.get(ctx, ownerID, id)
}

func (s *Store) get(ctx context.Context, ownerID int64, id int64) (*schemas.Monitor, error) {
	var monitor schemas.Monitor
	err := s.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&monitor).Error
	if err != nil {
		return nil, errors.NotFound("monitor not found")
	}
	return &monitor, nil
}

func (s *Store) List(ctx context.Context, ownerID int64) ([]schemas.Monitor, error) {
	var monitors []schemas.Monitor
	if err := s.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("id").
		Find(&monitors).Error; err != nil {
		return nil, errors.Internal("could not list monitors", err)
	}
	return monitors, nil
}

func (s *Store) ListEnabled(ctx context.Context) ([]schemas.Monitor, error) {
	var monitors []schemas.Monitor
	if err := s.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id").
		Find(&monitors).Error; err != nil {
		return nil, errors.Internal("could not list monitors", err)
	}
	return monitors, nil
}

func (s *Store) ByPushToken(ctx context.Context, token string) (*schemas.Monitor, error) {
	var monitor schemas.Monitor
	err := s.db.WithContext(ctx).
		Where("push_token = ? AND type = ? AND enabled = ?", token, TypePush, true).
		First(&monitor).Error
	if err != nil {
		return nil, errors.NotFound("unknown push token")
	}
	return &monitor, nil
}
