package schemas

import "time"

// MonitorSchedule is the probe cadence embedded in Monitor.
type MonitorSchedule struct {
	IntervalSeconds int `gorm:"column:interval_seconds"`
	TimeoutSeconds  int `gorm:"column:timeout_seconds"`
}

// MonitorExpectation is what a probe must match to count as up, embedded in Monitor.
type MonitorExpectation struct {
	ExpectedStatus  int    `gorm:"column:expected_status"`
	ExpectedKeyword string `gorm:"column:expected_keyword"`
}

// Monitor is one watched thing: an HTTP URL, a TCP port, or a push heartbeat.
// ProjectID links it to one row of projects, the GitHub repository it watches; nil when unlinked.
type Monitor struct {
	ID      int64  `gorm:"column:id;primaryKey"`
	OwnerID int64  `gorm:"column:owner_id;index"`
	Slug    string `gorm:"column:slug;uniqueIndex"`
	Name    string `gorm:"column:name"`
	Type    string `gorm:"column:type"`
	Target  string `gorm:"column:target"`
	MonitorSchedule
	MonitorExpectation
	Enabled   bool    `gorm:"column:enabled"`
	PushToken *string `gorm:"column:push_token;uniqueIndex"`
	ProjectID *int64  `gorm:"column:project_id"`
	MonitorStamps
}

// MonitorStamps is when a monitor was created and last edited, embedded in Monitor.
type MonitorStamps struct {
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Monitor) TableName() string { return "monitors" }

// CheckRaw is one probe result, kept for the raw retention window only.
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

// CheckBucket is one rollup row of totals and average latency over a bucket.
type CheckBucket struct {
	MonitorID    int64     `gorm:"column:monitor_id;primaryKey"`
	BucketStart  time.Time `gorm:"column:bucket_start;primaryKey"`
	Total        int64     `gorm:"column:total"`
	Failed       int64     `gorm:"column:failed"`
	AvgLatencyMS int64     `gorm:"column:avg_latency_ms"`
}

// CheckHourly is the hourly rollup over checks_raw.
type CheckHourly CheckBucket

func (CheckHourly) TableName() string { return "checks_hourly" }

// CheckDaily is the daily rollup over checks_raw.
type CheckDaily CheckBucket

func (CheckDaily) TableName() string { return "checks_daily" }

// Incident is one downtime episode for a monitor.
type Incident struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	MonitorID  int64      `gorm:"column:monitor_id;index"`
	OpenedAt   time.Time  `gorm:"column:opened_at"`
	ResolvedAt *time.Time `gorm:"column:resolved_at"`
	Cause      string     `gorm:"column:cause"`
}

func (Incident) TableName() string { return "incidents" }

// StatusPage is a public page grouping monitors under one slug.
type StatusPage struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Slug      string    `gorm:"column:slug;uniqueIndex"`
	Title     string    `gorm:"column:title"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (StatusPage) TableName() string { return "status_pages" }

// StatusPageMonitor is one monitor's membership and display order on a status page.
type StatusPageMonitor struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	PageID    int64 `gorm:"column:page_id;uniqueIndex:idx_page_monitor"`
	MonitorID int64 `gorm:"column:monitor_id;uniqueIndex:idx_page_monitor"`
	Position  int   `gorm:"column:position"`
}

func (StatusPageMonitor) TableName() string { return "status_page_monitors" }
