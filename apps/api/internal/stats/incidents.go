package stats

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// incidentRow is an incident joined to the monitor it belongs to, so the
// per-monitor breakdown does not need a second query to name anything.
type incidentRow struct {
	schemas.Incident
	Slug string
	Name string
}

// Incidents aggregates the owner's incidents over the last `days` days.
func (s *Service) Incidents(ctx context.Context, ownerID int64, days int) (IncidentStats, error) {
	start := truncate(time.Now().UTC(), ResolutionDay).Add(-time.Duration(days-1) * 24 * time.Hour)

	var found []incidentRow
	if err := s.db.WithContext(ctx).
		Table("incidents").
		Select("incidents.*, monitors.slug AS slug, monitors.name AS name").
		Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Where("monitors.owner_id = ? AND incidents.opened_at >= ?", ownerID, start).
		Order("incidents.opened_at").
		Scan(&found).Error; err != nil {
		return IncidentStats{}, errors.Internal("could not aggregate incidents", err)
	}

	tally := newTally(days, len(found))
	for i := range found {
		tally.record(&found[i])
	}
	return tally.result(days, start), nil
}

// tally accumulates one pass over the incidents into every figure the page
// shows, so the rows are walked once rather than once per statistic.
type tally struct {
	stats         IncidentStats
	now           time.Time
	daily         map[time.Time]*IncidentDay
	perMonitor    map[int64]*IncidentMonitor
	repaired      int64
	repairedCount int64
	longest       int64
}

func newTally(days int, rows int) *tally {
	return &tally{
		stats:      IncidentStats{Days: days, Total: int64(rows)},
		now:        time.Now().UTC(),
		daily:      make(map[time.Time]*IncidentDay, days),
		perMonitor: make(map[int64]*IncidentMonitor, rows),
	}
}

func (t *tally) record(row *incidentRow) {
	seconds := t.downtime(row)
	t.stats.TotalDowntimeSeconds += seconds
	if seconds > t.longest {
		t.longest = seconds
	}

	day := truncate(row.OpenedAt.UTC(), ResolutionDay)
	if bucket, ok := t.daily[day]; ok {
		bucket.Opened++
		bucket.DowntimeSeconds += seconds
	} else {
		t.daily[day] = &IncidentDay{Day: day, Opened: 1, DowntimeSeconds: seconds}
	}

	if entry, ok := t.perMonitor[row.MonitorID]; ok {
		entry.Incidents++
		entry.DowntimeSeconds += seconds
		return
	}
	t.perMonitor[row.MonitorID] = &IncidentMonitor{
		MonitorID:       row.MonitorID,
		Slug:            row.Slug,
		Name:            row.Name,
		Incidents:       1,
		DowntimeSeconds: seconds,
	}
}

// downtime measures an open incident to now, because it is still accruing.
// That is honest for the running totals, and it is exactly why an open
// incident stays out of MTTR: a mean that keeps climbing while nobody touches
// anything is not a repair time.
func (t *tally) downtime(row *incidentRow) int64 {
	end := t.now
	if row.ResolvedAt != nil {
		end = row.ResolvedAt.UTC()
		t.stats.Resolved++
	} else {
		t.stats.Open++
	}

	seconds := int64(end.Sub(row.OpenedAt.UTC()).Seconds())
	if seconds < 0 {
		seconds = 0
	}
	if row.ResolvedAt != nil {
		t.repaired += seconds
		t.repairedCount++
	}
	return seconds
}

func (t *tally) result(days int, start time.Time) IncidentStats {
	stats := t.stats
	if t.repairedCount > 0 {
		mttr := t.repaired / t.repairedCount
		stats.MTTRSeconds = &mttr
	}
	if stats.Total > 0 {
		stats.LongestSeconds = &t.longest
	}

	stats.ByDay = make([]IncidentDay, 0, days)
	for i := 0; i < days; i++ {
		day := start.Add(time.Duration(i) * 24 * time.Hour)
		if bucket, ok := t.daily[day]; ok {
			stats.ByDay = append(stats.ByDay, *bucket)
			continue
		}
		stats.ByDay = append(stats.ByDay, IncidentDay{Day: day})
	}

	stats.ByMonitor = make([]IncidentMonitor, 0, len(t.perMonitor))
	for _, entry := range t.perMonitor {
		stats.ByMonitor = append(stats.ByMonitor, *entry)
	}
	sortByLoad(stats.ByMonitor)
	return stats
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
