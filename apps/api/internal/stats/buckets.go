package stats

import (
	"context"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

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
