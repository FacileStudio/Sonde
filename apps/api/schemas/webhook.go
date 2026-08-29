package schemas

import "time"

// Webhook is an HTTP endpoint notified on monitor transitions.
type Webhook struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"id"`
	OwnerID      int64     `gorm:"column:owner_id;index" json:"owner_id"`
	Name         string    `gorm:"column:name" json:"name"`
	URL          string    `gorm:"column:url" json:"url"`
	Secret       string    `gorm:"column:secret" json:"secret,omitempty"`
	SecretHeader string    `gorm:"column:secret_header" json:"secret_header"`
	Enabled      bool      `gorm:"column:enabled" json:"enabled"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Webhook) TableName() string { return "webhooks" }
