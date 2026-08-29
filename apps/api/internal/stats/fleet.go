package stats

import (
	"context"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

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
	var fleet window

	for i := range monitors {
		summary := summarize(&monitors[i], fill(history[monitors[i].ID], start, ResolutionHour, 24))
		summary.OpenIncidents = openIncidents[monitors[i].ID]
		summary.applyLastCheck(latest[monitors[i].ID], monitors[i].Enabled)

		fleet.add(summary.window)
		out.Totals.count(summary.Status)
		out.Totals.OpenIncidents += summary.OpenIncidents
		out.Monitors = append(out.Monitors, summary.MonitorSummary)
	}

	out.Totals.Monitors = int64(len(monitors))
	out.Totals.Checks24h = fleet.total
	out.Totals.Uptime24h = percentage(fleet.total, fleet.failed)
	out.Totals.AvgLatency24h = mean(fleet.latencyWeight, fleet.total)
	return out, nil
}

// window accumulates the three running sums an uptime-and-latency summary
// needs. Latency is weighted by check count, not averaged over buckets: an
// hour with two probes must not pull the mean as hard as an hour with sixty.
type window struct {
	total         int64
	failed        int64
	latencyWeight int64
}

func (w *window) add(other window) {
	w.total += other.total
	w.failed += other.failed
	w.latencyWeight += other.latencyWeight
}

// summarized is a row plus the sums it was built from, so the fleet totals can
// be accumulated without walking every monitor's buckets a second time.
type summarized struct {
	MonitorSummary
	window window
}

func summarize(monitor *schemas.Monitor, buckets []Bucket) summarized {
	var totals window
	for _, bucket := range buckets {
		totals.total += bucket.Total
		totals.failed += bucket.Failed
		totals.latencyWeight += bucket.AvgLatencyMS * bucket.Total
	}

	return summarized{
		MonitorSummary: MonitorSummary{
			ID:              monitor.ID,
			Slug:            monitor.Slug,
			Name:            monitor.Name,
			Type:            monitor.Type,
			Target:          monitor.Target,
			IntervalSeconds: monitor.IntervalSeconds,
			Enabled:         monitor.Enabled,
			Status:          StatusUnknown,
			Day24h: Day24h{
				Uptime24h:     percentage(totals.total, totals.failed),
				AvgLatency24h: mean(totals.latencyWeight, totals.total),
				Checks24h:     totals.total,
				Failed24h:     totals.failed,
			},
			History: buckets,
		},
		window: totals,
	}
}

// applyLastCheck derives the status from the newest probe. A zero CheckRaw is
// the "never probed" case, which stays unknown.
//
// Paused wins over all of it: the state is "nobody is watching this", whatever
// the final check happened to say before the probing stopped.
func (s *summarized) applyLastCheck(check schemas.CheckRaw, enabled bool) {
	if check.CreatedAt.IsZero() {
		if !enabled {
			s.Status = StatusPaused
		}
		return
	}

	at := check.CreatedAt
	latency := check.LatencyMS
	s.LastCheckAt = &at
	s.LastLatencyMS = &latency
	s.LastError = check.Error
	s.Status = StatusDown
	if check.OK {
		s.Status = StatusUp
	}
	if !enabled {
		s.Status = StatusPaused
	}
}

func (t *FleetTotals) count(status string) {
	switch status {
	case StatusUp:
		t.Up++
	case StatusDown:
		t.Down++
	case StatusPaused:
		t.Paused++
	default:
		t.Unknown++
	}
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
