package checker

import (
	"context"
	"database/sql"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"

	"gorm.io/gorm"
)

// Events queues a transition inside the transaction that writes its incident,
// so the event and the state change commit or roll back together.
// events.Outbox satisfies it.
type Events interface {
	Enqueue(ctx context.Context, tx *sql.Tx, transition Transition) error
}

type GormRecorder struct {
	db     *gorm.DB
	events Events
}

// NewGormRecorder returns the recorder over db. A nil events queues nothing.
func NewGormRecorder(db *gorm.DB, events Events) *GormRecorder {
	return &GormRecorder{db: db, events: events}
}

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
SELECT ?, date_trunc('hour', ?::timestamptz), 1, ?, ?
ON CONFLICT (monitor_id, bucket_start) DO UPDATE SET
    total = checks_hourly.total + 1,
    failed = checks_hourly.failed + EXCLUDED.failed,
    avg_latency_ms =
        (checks_hourly.avg_latency_ms * checks_hourly.total + EXCLUDED.avg_latency_ms)
        / (checks_hourly.total + 1)`,
			monitorID, time.Now(), failed, latencyMS).Error
	})
}

// OpenIncident stores the incident a down transition opens and queues the
// transition in the same transaction.
func (r *GormRecorder) OpenIncident(ctx context.Context, down Transition, cause string) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO incidents (monitor_id, opened_at, cause) VALUES ($1, $2, $3)`,
			down.MonitorID, time.Now(), cause); err != nil {
			return err
		}
		return r.enqueue(ctx, tx, &down)
	})
}

// ResolveIncidents closes every open incident of the monitor. A non-nil
// recovery is queued in the same transaction.
func (r *GormRecorder) ResolveIncidents(ctx context.Context, monitorID int64, recovery *Transition) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE incidents SET resolved_at = $1 WHERE monitor_id = $2 AND resolved_at IS NULL`,
			time.Now(), monitorID); err != nil {
			return err
		}
		return r.enqueue(ctx, tx, recovery)
	})
}

func (r *GormRecorder) enqueue(ctx context.Context, tx *sql.Tx, transition *Transition) error {
	if r.events == nil || transition == nil {
		return nil
	}
	return r.events.Enqueue(ctx, tx, *transition)
}

func (r *GormRecorder) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
