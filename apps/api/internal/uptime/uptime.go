// Package uptime answers availability questions from the rollup tables.
//
// The hourly rollups are kept for 13 months, which covers every query window
// v1 needs (24h / 7d / 30d / 90d); daily rollups exist for history beyond
// that. A window with no data reports no percentage rather than inventing a
// perfect score.
package uptime

import (
	"context"

	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

var windows = []struct {
	Name  string
	Hours int
}{
	{Name: "24h", Hours: 24},
	{Name: "7d", Hours: 7 * 24},
	{Name: "30d", Hours: 30 * 24},
	{Name: "90d", Hours: 90 * 24},
}

type Stat struct {
	Window string   `json:"window"`
	Uptime *float64 `json:"uptime"`
	Total  int64    `json:"total"`
	Failed int64    `json:"failed"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Summary(ctx context.Context, monitorID int64) ([]Stat, error) {
	stats := make([]Stat, 0, len(windows))
	for _, window := range windows {
		var row struct {
			Total  int64
			Failed int64
		}
		err := s.db.WithContext(ctx).Raw(
			`SELECT COALESCE(SUM(total), 0) AS total, COALESCE(SUM(failed), 0) AS failed
             FROM checks_hourly
             WHERE monitor_id = ? AND bucket_start >= date_trunc('hour', now()) - make_interval(hours => ?)`,
			monitorID, window.Hours,
		).Scan(&row).Error
		if err != nil {
			return nil, errors.Internal("could not compute uptime", err)
		}
		stats = append(stats, Stat{
			Window: window.Name,
			Uptime: percentage(row.Total, row.Failed),
			Total:  row.Total,
			Failed: row.Failed,
		})
	}
	return stats, nil
}

func percentage(total int64, failed int64) *float64 {
	if total == 0 {
		return nil
	}
	value := float64(total-failed) / float64(total) * 100
	return &value
}
