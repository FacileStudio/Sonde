// Package retention runs the daily prune job: raw checks older than the
// configured window are deleted once their buckets have been rolled up into
// checks_hourly and checks_daily.
package retention

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

const (
	DefaultRawRetentionDays = 30

	// HourlyRetentionMonths covers every window the API offers, so a 90-day
	// uptime question is always answerable from checks_hourly. Exported
	// because settings reports it: the number that decides how far back a
	// chart can go should not be a secret of this package.
	HourlyRetentionMonths = 13

	// FleetRetentionDays is how long a resolved fleet finding and a fleet run
	// are kept. An open finding is never pruned: it is current state.
	FleetRetentionDays = 90
)

type Job struct {
	db               *gorm.DB
	rawRetentionDays int
	logger           *slog.Logger
	interval         time.Duration
}

func New(db *gorm.DB, rawRetentionDays int, logger *slog.Logger) *Job {
	if rawRetentionDays <= 0 {
		rawRetentionDays = DefaultRawRetentionDays
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Job{db: db, rawRetentionDays: rawRetentionDays, logger: logger, interval: 24 * time.Hour}
}

func (j *Job) Run(ctx context.Context) error {
	if err := j.rollupHours(ctx); err != nil {
		return err
	}
	if err := j.rollupDays(ctx); err != nil {
		return err
	}
	if err := j.prune(ctx); err != nil {
		return err
	}
	return j.pruneFleet(ctx)
}

func (j *Job) pruneFleet(ctx context.Context) error {
	findings := j.db.WithContext(ctx).Exec(
		`DELETE FROM fleet_findings WHERE resolved_at IS NOT NULL AND resolved_at < now() - make_interval(days => ?)`,
		FleetRetentionDays)
	j.log("pruned resolved fleet findings", findings)

	runs := j.db.WithContext(ctx).Exec(
		`DELETE FROM fleet_runs WHERE created_at < now() - make_interval(days => ?)`,
		FleetRetentionDays)
	j.log("pruned fleet runs", runs)

	return firstError(findings.Error, runs.Error)
}

func (j *Job) rollupHours(ctx context.Context) error {
	result := j.db.WithContext(ctx).Exec(`
INSERT INTO checks_hourly (monitor_id, bucket_start, total, failed, avg_latency_ms)
SELECT monitor_id,
       date_trunc('hour', created_at),
       COUNT(*)::bigint,
       COUNT(*) FILTER (WHERE NOT ok)::bigint,
       COALESCE((SUM(latency_ms) / GREATEST(COUNT(*), 1))::bigint, 0)
FROM checks_raw
WHERE created_at < date_trunc('hour', now())
GROUP BY 1, 2
ON CONFLICT (monitor_id, bucket_start) DO UPDATE SET
    total = EXCLUDED.total,
    failed = EXCLUDED.failed,
    avg_latency_ms = EXCLUDED.avg_latency_ms`)
	j.log("rolled up hourly checks", result)
	return result.Error
}

func (j *Job) rollupDays(ctx context.Context) error {
	result := j.db.WithContext(ctx).Exec(`
INSERT INTO checks_daily (monitor_id, bucket_start, total, failed, avg_latency_ms)
SELECT monitor_id,
       date_trunc('day', bucket_start),
       SUM(total)::bigint,
       SUM(failed)::bigint,
       COALESCE((SUM(total * avg_latency_ms) / GREATEST(SUM(total), 1))::bigint, 0)
FROM checks_hourly
WHERE bucket_start < date_trunc('day', now())
GROUP BY 1, 2
ON CONFLICT (monitor_id, bucket_start) DO UPDATE SET
    total = EXCLUDED.total,
    failed = EXCLUDED.failed,
    avg_latency_ms = EXCLUDED.avg_latency_ms`)
	j.log("rolled up daily checks", result)
	return result.Error
}

func (j *Job) prune(ctx context.Context) error {
	raw := j.db.WithContext(ctx).Exec(
		`DELETE FROM checks_raw WHERE created_at < date_trunc('hour', now()) - make_interval(days => ?)`,
		j.rawRetentionDays)
	j.log("pruned raw checks", raw)

	hourly := j.db.WithContext(ctx).Exec(
		`DELETE FROM checks_hourly WHERE bucket_start < date_trunc('hour', now()) - make_interval(months => ?)`,
		HourlyRetentionMonths)
	j.log("pruned hourly checks", hourly)

	return firstError(raw.Error, hourly.Error)
}

func (j *Job) log(operation string, result *gorm.DB) {
	if result.Error != nil {
		j.logger.Error(operation, slog.Any("error", result.Error))
		return
	}
	j.logger.Info(operation, slog.Int64("rows", result.RowsAffected))
}

func (j *Job) Loop(ctx context.Context) {
	if err := j.Run(ctx); err != nil {
		j.logger.Error("retention run failed", slog.Any("error", err))
	}
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := j.Run(ctx); err != nil {
				j.logger.Error("retention run failed", slog.Any("error", err))
			}
		}
	}
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
