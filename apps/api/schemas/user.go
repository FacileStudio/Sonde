package schemas

import "time"

// User is a Sonde account; passwords hash through porte's argon2id kit.
type User struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	Email        string    `gorm:"column:email;uniqueIndex"`
	Name         string    `gorm:"column:name"`
	IsAdmin      bool      `gorm:"column:is_admin"`
	PasswordHash string    `gorm:"column:password_hash"`
	AvatarURL    string    `gorm:"column:avatar_url"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }
