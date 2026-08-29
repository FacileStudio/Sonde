package stats

import (
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
)

// A gap in coverage and a bucket where everything failed are different facts
// with different colours on the chart. Dropping the gap slides the series left
// and makes the window claim continuous coverage it never had; reporting it as
// 0% uptime invents an outage nobody had.
func TestFillReportsGapsAsGapsNotOutages(t *testing.T) {
	start := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	rows := map[time.Time]schemas.CheckBucket{
		start:                    {BucketStart: start, Total: 60, Failed: 0, AvgLatencyMS: 120},
		start.Add(2 * time.Hour): {BucketStart: start.Add(2 * time.Hour), Total: 60, Failed: 60, AvgLatencyMS: 0},
	}

	buckets := fill(rows, start, ResolutionHour, 4)

	if len(buckets) != 4 {
		t.Fatalf("got %d buckets, want 4 — the window is the axis, not the rows that exist", len(buckets))
	}
	for i, bucket := range buckets {
		want := start.Add(time.Duration(i) * time.Hour)
		if !bucket.BucketStart.Equal(want) {
			t.Errorf("bucket %d starts at %s, want %s", i, bucket.BucketStart, want)
		}
	}
	if buckets[0].Uptime == nil || *buckets[0].Uptime != 100 {
		t.Errorf("a clean hour reported %v, want 100", buckets[0].Uptime)
	}
	if buckets[1].Uptime != nil {
		t.Errorf("an unprobed hour reported %v%% uptime, want null", *buckets[1].Uptime)
	}
	if buckets[1].Total != 0 {
		t.Errorf("an unprobed hour reported %d checks, want 0", buckets[1].Total)
	}
	if buckets[2].Uptime == nil || *buckets[2].Uptime != 0 {
		t.Errorf("a fully failed hour reported %v, want 0", buckets[2].Uptime)
	}
	if buckets[3].Uptime != nil {
		t.Error("the trailing hour has no rollup row and must report null")
	}
}

// The Go side builds bucket keys that have to land on the same instant as the
// ones Postgres date_trunc wrote, or every lookup misses and the whole series
// zero-fills into a flat "no data" chart over a database full of checks.
func TestTruncateMatchesDateTrunc(t *testing.T) {
	at := time.Date(2026, 8, 29, 14, 37, 12, 500, time.FixedZone("CEST", 2*60*60))

	hour := truncate(at, ResolutionHour)
	if want := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC); !hour.Equal(want) {
		t.Errorf("hour = %s, want %s", hour, want)
	}
	day := truncate(at, ResolutionDay)
	if want := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC); !day.Equal(want) {
		t.Errorf("day = %s, want %s", day, want)
	}
	if hour.Location() != time.UTC || day.Location() != time.UTC {
		t.Error("truncate returned a local time; bucket keys are UTC on both sides of the wire")
	}
}

// Nothing measured is not a perfect score. This mirrors internal/uptime, and it
// is the difference between a fresh monitor reading "no data" and reading
// "100%, everything is fine".
func TestPercentageRefusesToInventAScore(t *testing.T) {
	if percentage(0, 0) != nil {
		t.Error("an empty window reported a percentage")
	}
	if got := percentage(100, 1); got == nil || *got != 99 {
		t.Errorf("percentage(100, 1) = %v, want 99", got)
	}
}

// Ranging a map is randomised, so a breakdown with no total order reshuffles
// itself between two identical requests.
func TestByMonitorOrderIsTotal(t *testing.T) {
	entries := []IncidentMonitor{
		{MonitorID: 3, Incidents: 2, DowntimeSeconds: 10},
		{MonitorID: 1, Incidents: 2, DowntimeSeconds: 10},
		{MonitorID: 2, Incidents: 5, DowntimeSeconds: 1},
	}
	sortByLoad(entries)

	if got := []int64{entries[0].MonitorID, entries[1].MonitorID, entries[2].MonitorID}; got[0] != 2 || got[1] != 1 || got[2] != 3 {
		t.Errorf("order = %v, want [2 1 3]: most incidents first, then id", got)
	}
}
