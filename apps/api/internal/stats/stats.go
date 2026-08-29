// Package stats answers the aggregate questions a dashboard asks, in one
// round trip each.
//
// The rollup tables were already populated and unreachable: every uptime
// percentage and latency trend in the client was being reassembled from
// checks_raw, one HTTP request per monitor. Fleet reads the whole overview in
// three queries regardless of how many monitors exist, Buckets reads a time
// series straight off checks_hourly or checks_daily, and Incidents reads the
// incident aggregates the incidents page needs. Nothing here writes.
package stats

import (
	"time"

	"gorm.io/gorm"
)

// Status is a monitor's current state as the dashboard renders it. Paused is
// separate from down on purpose: a monitor nobody is probing is not a monitor
// that failed, and colouring the two the same makes a quiet fleet look like a
// burning one.
const (
	StatusUp      = "up"
	StatusDown    = "down"
	StatusPaused  = "paused"
	StatusUnknown = "unknown"
)

// Resolution names the rollup table a window reads from.
const (
	ResolutionHour = "hour"
	ResolutionDay  = "day"
)

// Window is one of the four spans the API accepts, and the shape of the series
// it produces. Hours drives the uptime sum; Points and Resolution drive the
// bucket series, so 24h and 7d read hourly buckets and the longer two read
// daily ones rather than returning 2160 points for a 90-day chart.
type Window struct {
	Name       string
	Hours      int
	Resolution string
	Points     int
}

var windows = map[string]Window{
	"24h": {Name: "24h", Hours: 24, Resolution: ResolutionHour, Points: 24},
	"7d":  {Name: "7d", Hours: 7 * 24, Resolution: ResolutionHour, Points: 7 * 24},
	"30d": {Name: "30d", Hours: 30 * 24, Resolution: ResolutionDay, Points: 30},
	"90d": {Name: "90d", Hours: 90 * 24, Resolution: ResolutionDay, Points: 90},
}

// LookupWindow resolves a query-parameter value, reporting whether it is one
// of the four the API accepts.
func LookupWindow(name string) (Window, bool) {
	window, ok := windows[name]
	return window, ok
}

// Bucket is one point of a time series. Uptime is nil when the bucket holds no
// checks at all, which is how a zero-filled gap tells itself apart from a
// bucket where everything failed.
type Bucket struct {
	BucketStart  time.Time `json:"bucket_start"`
	Total        int64     `json:"total"`
	Failed       int64     `json:"failed"`
	AvgLatencyMS int64     `json:"avg_latency_ms"`
	Uptime       *float64  `json:"uptime"`
}

// Series is a monitor's history over one window, zero-filled so the chart's
// x-axis is the window and not merely the buckets that happen to exist.
type Series struct {
	MonitorID  int64    `json:"monitor_id"`
	Window     string   `json:"window"`
	Resolution string   `json:"resolution"`
	Buckets    []Bucket `json:"buckets"`
}

// LastCheck is the newest probe, embedded in MonitorSummary. All three fields
// are absent together — a monitor nobody has probed yet has no last anything —
// so they travel as one group. Embedding keeps them flat on the wire, the same
// shape schemas.Monitor uses for its schedule and expectation groups.
type LastCheck struct {
	LastCheckAt   *time.Time `json:"last_check_at"`
	LastLatencyMS *int64     `json:"last_latency_ms"`
	LastError     string     `json:"last_error,omitempty"`
}

// Day24h is a monitor's last 24 hours, embedded in MonitorSummary.
type Day24h struct {
	Uptime24h     *float64 `json:"uptime_24h"`
	AvgLatency24h *int64   `json:"avg_latency_24h"`
	Checks24h     int64    `json:"checks_24h"`
	Failed24h     int64    `json:"failed_24h"`
}

// MonitorSummary is one row of the fleet overview: the monitor's identity, the
// last thing that happened to it, and its 24-hour shape.
type MonitorSummary struct {
	ID              int64  `json:"id"`
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Target          string `json:"target"`
	IntervalSeconds int    `json:"interval_seconds"`
	Enabled         bool   `json:"enabled"`
	Status          string `json:"status"`
	LastCheck
	Day24h
	OpenIncidents int64    `json:"open_incidents"`
	History       []Bucket `json:"history"`
}

// FleetTotals is the top row of the monitors page.
type FleetTotals struct {
	Monitors      int64    `json:"monitors"`
	Up            int64    `json:"up"`
	Down          int64    `json:"down"`
	Paused        int64    `json:"paused"`
	Unknown       int64    `json:"unknown"`
	OpenIncidents int64    `json:"open_incidents"`
	Uptime24h     *float64 `json:"uptime_24h"`
	AvgLatency24h *int64   `json:"avg_latency_24h"`
	Checks24h     int64    `json:"checks_24h"`
}

// Fleet is the whole monitors page in one response.
type Fleet struct {
	Totals   FleetTotals      `json:"totals"`
	Monitors []MonitorSummary `json:"monitors"`
}

// IncidentDay is one bar of the incidents histogram.
type IncidentDay struct {
	Day             time.Time `json:"day"`
	Opened          int64     `json:"opened"`
	DowntimeSeconds int64     `json:"downtime_seconds"`
}

// IncidentMonitor is one monitor's share of the incident load over the window.
type IncidentMonitor struct {
	MonitorID       int64  `json:"monitor_id"`
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Incidents       int64  `json:"incidents"`
	DowntimeSeconds int64  `json:"downtime_seconds"`
}

// IncidentStats is the incidents page's header and charts.
//
// MTTRSeconds averages resolved incidents only. An incident that is still open
// has no repair time yet, and folding "so far" into a mean makes the number
// climb on its own while nothing is being fixed.
type IncidentStats struct {
	Days                 int               `json:"days"`
	Total                int64             `json:"total"`
	Open                 int64             `json:"open"`
	Resolved             int64             `json:"resolved"`
	MTTRSeconds          *int64            `json:"mttr_seconds"`
	LongestSeconds       *int64            `json:"longest_seconds"`
	TotalDowntimeSeconds int64             `json:"total_downtime_seconds"`
	ByDay                []IncidentDay     `json:"by_day"`
	ByMonitor            []IncidentMonitor `json:"by_monitor"`
}

// Service reads the aggregates. It never writes.
type Service struct {
	db *gorm.DB
}

// NewService returns a stats service over the given database.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }
