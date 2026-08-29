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
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

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

// MonitorSummary is one row of the fleet overview: the monitor's identity, the
// last thing that happened to it, and its 24-hour shape.
type MonitorSummary struct {
	ID              int64      `json:"id"`
	Slug            string     `json:"slug"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Target          string     `json:"target"`
	IntervalSeconds int        `json:"interval_seconds"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	LastCheckAt     *time.Time `json:"last_check_at"`
	LastLatencyMS   *int64     `json:"last_latency_ms"`
	LastError       string     `json:"last_error,omitempty"`
	Uptime24h       *float64   `json:"uptime_24h"`
	AvgLatency24h   *int64     `json:"avg_latency_24h"`
	Checks24h       int64      `json:"checks_24h"`
	Failed24h       int64      `json:"failed_24h"`
	OpenIncidents   int64      `json:"open_incidents"`
	History         []Bucket   `json:"history"`
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

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// Fleet returns every monitor the owner has, each with its current status and
// 24-hour history, plus the totals across all of them.
//
// Three queries, not three per monitor: the last check, the 24h hourly rollups
// and the open-incident counts each come back for the whole set at once. The
// page this backs used to fan out one /checks request per monitor from the
// browser.
func (s *Service) Fleet(ctx context.Context, ownerID int64) (Fleet, error) {
	var monitors []schemas.Monitor
	if err := s.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("id").
		Find(&monitors).Error; err != nil {
		return Fleet{}, errors.Internal("could not list monitors", err)
	}

	out := Fleet{Monitors: make([]MonitorSummary, 0, len(monitors))}
	if len(monitors) == 0 {
		return out, nil
	}

	ids := make([]int64, 0, len(monitors))
	for i := range monitors {
		ids = append(ids, monitors[i].ID)
	}

	latest, err := s.latestChecks(ctx, ids)
	if err != nil {
		return Fleet{}, err
	}
	history, err := s.recentHourly(ctx, ids)
	if err != nil {
		return Fleet{}, err
	}
	openIncidents, err := s.openIncidents(ctx, ids)
	if err != nil {
		return Fleet{}, err
	}

	start := truncate(time.Now().UTC(), ResolutionHour).Add(-23 * time.Hour)
	var fleetTotal, fleetFailed, fleetLatencyWeight int64

	for i := range monitors {
		monitor := &monitors[i]
		buckets := fill(history[monitor.ID], start, ResolutionHour, 24)

		var total, failed, latencyWeight int64
		for _, bucket := range buckets {
			total += bucket.Total
			failed += bucket.Failed
			latencyWeight += bucket.AvgLatencyMS * bucket.Total
		}
		fleetTotal += total
		fleetFailed += failed
		fleetLatencyWeight += latencyWeight

		summary := MonitorSummary{
			ID:              monitor.ID,
			Slug:            monitor.Slug,
			Name:            monitor.Name,
			Type:            monitor.Type,
			Target:          monitor.Target,
			IntervalSeconds: monitor.IntervalSeconds,
			Enabled:         monitor.Enabled,
			Status:          StatusUnknown,
			Uptime24h:       percentage(total, failed),
			AvgLatency24h:   mean(latencyWeight, total),
			Checks24h:       total,
			Failed24h:       failed,
			OpenIncidents:   openIncidents[monitor.ID],
			History:         buckets,
		}

		if check, ok := latest[monitor.ID]; ok {
			at := check.CreatedAt
			latency := check.LatencyMS
			summary.LastCheckAt = &at
			summary.LastLatencyMS = &latency
			summary.LastError = check.Error
			if check.OK {
				summary.Status = StatusUp
			} else {
				summary.Status = StatusDown
			}
		}
		// Paused wins over the last probe: the state is "nobody is watching
		// this", whatever the final check happened to say before it stopped.
		if !monitor.Enabled {
			summary.Status = StatusPaused
		}

		switch summary.Status {
		case StatusUp:
			out.Totals.Up++
		case StatusDown:
			out.Totals.Down++
		case StatusPaused:
			out.Totals.Paused++
		default:
			out.Totals.Unknown++
		}
		out.Totals.OpenIncidents += summary.OpenIncidents
		out.Monitors = append(out.Monitors, summary)
	}

	out.Totals.Monitors = int64(len(monitors))
	out.Totals.Checks24h = fleetTotal
	out.Totals.Uptime24h = percentage(fleetTotal, fleetFailed)
	out.Totals.AvgLatency24h = mean(fleetLatencyWeight, fleetTotal)
	return out, nil
}

// Buckets returns one monitor's history over a window, zero-filled.
func (s *Service) Buckets(ctx context.Context, monitorID int64, window Window) (Series, error) {
	table := "checks_hourly"
	if window.Resolution == ResolutionDay {
		table = "checks_daily"
	}
	start := truncate(time.Now().UTC(), window.Resolution).
		Add(-time.Duration(window.Points-1) * step(window.Resolution))

	var rows []schemas.CheckBucket
	if err := s.db.WithContext(ctx).
		Table(table).
		Select("monitor_id, bucket_start, total, failed, avg_latency_ms").
		Where("monitor_id = ? AND bucket_start >= ?", monitorID, start).
		Order("bucket_start").
		Find(&rows).Error; err != nil {
		return Series{}, errors.Internal("could not load the check history", err)
	}

	indexed := make(map[time.Time]schemas.CheckBucket, len(rows))
	for _, row := range rows {
		indexed[row.BucketStart.UTC()] = row
	}

	return Series{
		MonitorID:  monitorID,
		Window:     window.Name,
		Resolution: window.Resolution,
		Buckets:    fill(indexed, start, window.Resolution, window.Points),
	}, nil
}

// Incidents aggregates the owner's incidents over the last `days` days.
func (s *Service) Incidents(ctx context.Context, ownerID int64, days int) (IncidentStats, error) {
	start := truncate(time.Now().UTC(), ResolutionDay).Add(-time.Duration(days-1) * 24 * time.Hour)

	var found []struct {
		schemas.Incident
		Slug string
		Name string
	}
	if err := s.db.WithContext(ctx).
		Table("incidents").
		Select("incidents.*, monitors.slug AS slug, monitors.name AS name").
		Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Where("monitors.owner_id = ? AND incidents.opened_at >= ?", ownerID, start).
		Order("incidents.opened_at").
		Scan(&found).Error; err != nil {
		return IncidentStats{}, errors.Internal("could not aggregate incidents", err)
	}

	now := time.Now().UTC()
	stats := IncidentStats{Days: days, Total: int64(len(found))}

	daily := make(map[time.Time]*IncidentDay, days)
	perMonitor := make(map[int64]*IncidentMonitor, len(found))
	var repaired, repairedCount, longest int64

	for i := range found {
		row := &found[i]
		// An open incident is still accruing, so its downtime is measured to
		// now. That is honest for the totals; it is why it stays out of MTTR.
		end := now
		if row.ResolvedAt != nil {
			end = row.ResolvedAt.UTC()
			stats.Resolved++
		} else {
			stats.Open++
		}
		seconds := int64(end.Sub(row.OpenedAt.UTC()).Seconds())
		if seconds < 0 {
			seconds = 0
		}
		stats.TotalDowntimeSeconds += seconds
		if row.ResolvedAt != nil {
			repaired += seconds
			repairedCount++
		}
		if seconds > longest {
			longest = seconds
		}

		day := truncate(row.OpenedAt.UTC(), ResolutionDay)
		if bucket, ok := daily[day]; ok {
			bucket.Opened++
			bucket.DowntimeSeconds += seconds
		} else {
			daily[day] = &IncidentDay{Day: day, Opened: 1, DowntimeSeconds: seconds}
		}

		if entry, ok := perMonitor[row.MonitorID]; ok {
			entry.Incidents++
			entry.DowntimeSeconds += seconds
		} else {
			perMonitor[row.MonitorID] = &IncidentMonitor{
				MonitorID:       row.MonitorID,
				Slug:            row.Slug,
				Name:            row.Name,
				Incidents:       1,
				DowntimeSeconds: seconds,
			}
		}
	}

	if repairedCount > 0 {
		mttr := repaired / repairedCount
		stats.MTTRSeconds = &mttr
	}
	if stats.Total > 0 {
		stats.LongestSeconds = &longest
	}

	stats.ByDay = make([]IncidentDay, 0, days)
	for i := 0; i < days; i++ {
		day := start.Add(time.Duration(i) * 24 * time.Hour)
		if bucket, ok := daily[day]; ok {
			stats.ByDay = append(stats.ByDay, *bucket)
			continue
		}
		stats.ByDay = append(stats.ByDay, IncidentDay{Day: day})
	}

	stats.ByMonitor = make([]IncidentMonitor, 0, len(perMonitor))
	for _, entry := range perMonitor {
		stats.ByMonitor = append(stats.ByMonitor, *entry)
	}
	sortByLoad(stats.ByMonitor)
	return stats, nil
}

func (s *Service) latestChecks(ctx context.Context, ids []int64) (map[int64]schemas.CheckRaw, error) {
	var rows []schemas.CheckRaw
	if err := s.db.WithContext(ctx).Raw(
		`SELECT DISTINCT ON (monitor_id) id, monitor_id, ok, latency_ms, error, created_at
         FROM checks_raw
         WHERE monitor_id IN ?
         ORDER BY monitor_id, created_at DESC`, ids,
	).Scan(&rows).Error; err != nil {
		return nil, errors.Internal("could not load the latest checks", err)
	}
	out := make(map[int64]schemas.CheckRaw, len(rows))
	for _, row := range rows {
		out[row.MonitorID] = row
	}
	return out, nil
}

func (s *Service) recentHourly(ctx context.Context, ids []int64) (map[int64]map[time.Time]schemas.CheckBucket, error) {
	start := truncate(time.Now().UTC(), ResolutionHour).Add(-23 * time.Hour)
	var rows []schemas.CheckBucket
	if err := s.db.WithContext(ctx).
		Table("checks_hourly").
		Select("monitor_id, bucket_start, total, failed, avg_latency_ms").
		Where("monitor_id IN ? AND bucket_start >= ?", ids, start).
		Order("bucket_start").
		Find(&rows).Error; err != nil {
		return nil, errors.Internal("could not load the recent history", err)
	}
	out := make(map[int64]map[time.Time]schemas.CheckBucket, len(ids))
	for _, row := range rows {
		bucket, ok := out[row.MonitorID]
		if !ok {
			bucket = make(map[time.Time]schemas.CheckBucket, 24)
			out[row.MonitorID] = bucket
		}
		bucket[row.BucketStart.UTC()] = row
	}
	return out, nil
}

func (s *Service) openIncidents(ctx context.Context, ids []int64) (map[int64]int64, error) {
	var rows []struct {
		MonitorID int64
		Count     int64
	}
	if err := s.db.WithContext(ctx).
		Table("incidents").
		Select("monitor_id, count(*) AS count").
		Where("resolved_at IS NULL AND monitor_id IN ?", ids).
		Group("monitor_id").
		Scan(&rows).Error; err != nil {
		return nil, errors.Internal("could not count open incidents", err)
	}
	out := make(map[int64]int64, len(rows))
	for _, row := range rows {
		out[row.MonitorID] = row.Count
	}
	return out, nil
}

// fill turns the buckets that exist into the buckets the window has. A missing
// bucket is a real gap — nothing was probed — so it lands with Total 0 and a
// nil Uptime rather than being dropped, which would slide the whole series
// left and make the chart claim continuous coverage it never had.
func fill(rows map[time.Time]schemas.CheckBucket, start time.Time, resolution string, points int) []Bucket {
	out := make([]Bucket, 0, points)
	interval := step(resolution)
	for i := 0; i < points; i++ {
		at := start.Add(time.Duration(i) * interval)
		row, ok := rows[at]
		if !ok {
			out = append(out, Bucket{BucketStart: at})
			continue
		}
		out = append(out, Bucket{
			BucketStart:  at,
			Total:        row.Total,
			Failed:       row.Failed,
			AvgLatencyMS: row.AvgLatencyMS,
			Uptime:       percentage(row.Total, row.Failed),
		})
	}
	return out
}

func step(resolution string) time.Duration {
	if resolution == ResolutionDay {
		return 24 * time.Hour
	}
	return time.Hour
}

// truncate mirrors Postgres date_trunc so a bucket built in Go lands on the
// same instant as the one the rollup job wrote.
func truncate(at time.Time, resolution string) time.Time {
	at = at.UTC()
	if resolution == ResolutionDay {
		return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	}
	return time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), 0, 0, 0, time.UTC)
}

// percentage reports no percentage rather than a fabricated 100 when nothing
// was measured, matching internal/uptime.
func percentage(total int64, failed int64) *float64 {
	if total == 0 {
		return nil
	}
	value := float64(total-failed) / float64(total) * 100
	return &value
}

func mean(weight int64, total int64) *int64 {
	if total == 0 {
		return nil
	}
	value := weight / total
	return &value
}

// sortByLoad ranks the noisiest monitor first, and breaks every tie so the
// order is total. Ranging a map is randomised, so without the id fallback the
// same data reorders itself between two requests.
func sortByLoad(entries []IncidentMonitor) {
	slices.SortFunc(entries, func(a IncidentMonitor, b IncidentMonitor) int {
		if a.Incidents != b.Incidents {
			return cmp.Compare(b.Incidents, a.Incidents)
		}
		if a.DowntimeSeconds != b.DowntimeSeconds {
			return cmp.Compare(b.DowntimeSeconds, a.DowntimeSeconds)
		}
		return cmp.Compare(a.MonitorID, b.MonitorID)
	})
}
