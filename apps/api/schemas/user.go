package schemas

import "time"

// User is a Sonde account; accounts sign in through Registre, so PasswordHash stays empty on new rows.
type User struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	Email        string    `gorm:"column:email;uniqueIndex"`
	Name         string    `gorm:"column:name"`
	IsAdmin      bool      `gorm:"column:is_admin"`
	PasswordHash string    `gorm:"column:password_hash"`
	AvatarURL    string    `gorm:"column:avatar_url"`
	Locale       string    `gorm:"column:locale"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }
