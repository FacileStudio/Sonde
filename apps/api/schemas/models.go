package schemas

import "time"

type Monitor struct {
	ID              int64     `gorm:"column:id;primaryKey"`
	OwnerID         int64     `gorm:"column:owner_id;index"`
	Slug            string    `gorm:"column:slug;uniqueIndex"`
	Name            string    `gorm:"column:name"`
	Type            string    `gorm:"column:type"`
	Target          string    `gorm:"column:target"`
	IntervalSeconds int       `gorm:"column:interval_seconds"`
	TimeoutSeconds  int       `gorm:"column:timeout_seconds"`
	ExpectedStatus  int       `gorm:"column:expected_status"`
	ExpectedKeyword string    `gorm:"column:expected_keyword"`
	Enabled         bool      `gorm:"column:enabled"`
	PushToken       *string   `gorm:"column:push_token;uniqueIndex"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (Monitor) TableName() string { return "monitors" }

type CheckRaw struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	MonitorID    int64      `gorm:"column:monitor_id;index"`
	OK           bool       `gorm:"column:ok"`
	LatencyMS    int64      `gorm:"column:latency_ms"`
	Error        string     `gorm:"column:error"`
	TLSExpiresAt *time.Time `gorm:"column:tls_expires_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (CheckRaw) TableName() string { return "checks_raw" }

type CheckHourly struct {
	MonitorID    int64     `gorm:"column:monitor_id;primaryKey"`
	BucketStart  time.Time `gorm:"column:bucket_start;primaryKey"`
	Total        int64     `gorm:"column:total"`
	Failed       int64     `gorm:"column:failed"`
	AvgLatencyMS int64     `gorm:"column:avg_latency_ms"`
}

func (CheckHourly) TableName() string { return "checks_hourly" }

type CheckDaily struct {
	MonitorID    int64     `gorm:"column:monitor_id;primaryKey"`
	BucketStart  time.Time `gorm:"column:bucket_start;primaryKey"`
	Total        int64     `gorm:"column:total"`
	Failed       int64     `gorm:"column:failed"`
	AvgLatencyMS int64     `gorm:"column:avg_latency_ms"`
}

func (CheckDaily) TableName() string { return "checks_daily" }

type Incident struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	MonitorID  int64      `gorm:"column:monitor_id;index"`
	OpenedAt   time.Time  `gorm:"column:opened_at"`
	ResolvedAt *time.Time `gorm:"column:resolved_at"`
	Cause      string     `gorm:"column:cause"`
}

func (Incident) TableName() string { return "incidents" }

type StatusPage struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Slug      string    `gorm:"column:slug;uniqueIndex"`
	Title     string    `gorm:"column:title"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (StatusPage) TableName() string { return "status_pages" }

type StatusPageMonitor struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	PageID    int64 `gorm:"column:page_id;uniqueIndex:idx_page_monitor"`
	MonitorID int64 `gorm:"column:monitor_id;uniqueIndex:idx_page_monitor"`
	Position  int   `gorm:"column:position"`
}

func (StatusPageMonitor) TableName() string { return "status_page_monitors" }
