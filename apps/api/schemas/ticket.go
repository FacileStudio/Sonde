package schemas

import "time"

type Ticket struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	EventID   int64      `gorm:"column:event_id;index"`
	CodeHash  string     `gorm:"column:code_hash;uniqueIndex"`
	Status    string     `gorm:"column:status"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (Ticket) TableName() string { return "tickets" }
