package checker

import (
	"context"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"

	"gorm.io/gorm"
)

type GormRecorder struct {
	db *gorm.DB
}

func NewGormRecorder(db *gorm.DB) *GormRecorder { return &GormRecorder{db: db} }

func (r *GormRecorder) RecordCheck(ctx context.Context, monitorID int64, ok bool, latencyMS int64, cause string, tlsExpiresAt *time.Time) error {
	check := &schemas.CheckRaw{
		MonitorID:    monitorID,
		OK:           ok,
		LatencyMS:    latencyMS,
		Error:        cause,
		TLSExpiresAt: tlsExpiresAt,
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(check).Error; err != nil {
			return err
		}
		failed := 0
		if !ok {
			failed = 1
		}
		return tx.Exec(`
INSERT INTO checks_hourly (monitor_id, bucket_start, total, failed, avg_latency_ms)
SELECT ?, date_trunc('hour', ?), 1, ?, ?
ON CONFLICT (monitor_id, bucket_start) DO UPDATE SET
    total = checks_hourly.total + 1,
    failed = checks_hourly.failed + EXCLUDED.failed,
    avg_latency_ms =
        (checks_hourly.avg_latency_ms * checks_hourly.total + EXCLUDED.avg_latency_ms)
        / (checks_hourly.total + 1)`,
			monitorID, time.Now(), failed, latencyMS).Error
	})
}

func (r *GormRecorder) OpenIncident(ctx context.Context, monitorID int64, cause string) error {
	return r.db.WithContext(ctx).Create(&schemas.Incident{MonitorID: monitorID, Cause: cause}).Error
}

func (r *GormRecorder) ResolveIncidents(ctx context.Context, monitorID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&schemas.Incident{}).
		Where("monitor_id = ? AND resolved_at IS NULL", monitorID).
		Update("resolved_at", now).Error
}
