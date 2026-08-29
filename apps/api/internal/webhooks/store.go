package webhooks

import (
	"context"
	stderrors "errors"
	"net/url"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"gorm.io/gorm"
)

const (
	DefaultSecretHeader = "x-sonde-signature-256"
)

type Input struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Secret       string `json:"secret"`
	SecretHeader string `json:"secret_header"`
	Enabled      *bool  `json:"enabled"`
}

func (in *Input) normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.URL = strings.TrimSpace(in.URL)
	in.Secret = strings.TrimSpace(in.Secret)
	in.SecretHeader = strings.TrimSpace(in.SecretHeader)
	if in.SecretHeader == "" {
		in.SecretHeader = DefaultSecretHeader
	}
}

func (in Input) Validate() error {
	if in.URL == "" {
		return errors.Invalid("url is required")
	}
	parsed, err := url.ParseRequestURI(in.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.Invalid("url must be a valid http or https URL")
	}
	return nil
}

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context, ownerID int64) ([]schemas.Webhook, error) {
	var rows []schemas.Webhook
	err := s.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list webhooks", err)
	}
	return rows, nil
}

func (s *Store) ListEnabledForOwner(ctx context.Context, ownerID int64) ([]schemas.Webhook, error) {
	var rows []schemas.Webhook
	err := s.db.WithContext(ctx).
		Where("owner_id = ? AND enabled = true", ownerID).
		Find(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to query webhooks", err)
	}
	return rows, nil
}

func (s *Store) Get(ctx context.Context, ownerID int64, id int64) (*schemas.Webhook, error) {
	var row schemas.Webhook
	err := s.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&row).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("webhook not found")
		}
		return nil, errors.Internal("failed to get webhook", err)
	}
	return &row, nil
}

func (s *Store) Create(ctx context.Context, ownerID int64, in Input) (*schemas.Webhook, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	row := schemas.Webhook{
		OwnerID:      ownerID,
		Name:         in.Name,
		URL:          in.URL,
		Secret:       in.Secret,
		SecretHeader: in.SecretHeader,
		Enabled:      enabled,
	}

	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, errors.Internal("failed to create webhook", err)
	}
	return &row, nil
}

func (s *Store) Update(ctx context.Context, ownerID int64, id int64, in Input) (*schemas.Webhook, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"name":          in.Name,
		"url":           in.URL,
		"secret_header": in.SecretHeader,
	}
	if in.Secret != "" {
		updates["secret"] = in.Secret
	}
	if in.Enabled != nil {
		updates["enabled"] = *in.Enabled
	}

	if err := s.db.WithContext(ctx).Model(existing).Updates(updates).Error; err != nil {
		return nil, errors.Internal("failed to update webhook", err)
	}
	return s.Get(ctx, ownerID, id)
}

func (s *Store) Delete(ctx context.Context, ownerID int64, id int64) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		Delete(&schemas.Webhook{})
	if res.Error != nil {
		return errors.Internal("failed to delete webhook", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound("webhook not found")
	}
	return nil
}
