package stats

import (
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
)

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
