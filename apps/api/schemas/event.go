package schemas

import "time"

type Event struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	Name      string     `gorm:"column:name"`
	StartsAt  time.Time  `gorm:"column:starts_at"`
	EndsAt    *time.Time `gorm:"column:ends_at"`
	OwnerID   string     `gorm:"column:owner_id;index"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (Event) TableName() string { return "events" }
