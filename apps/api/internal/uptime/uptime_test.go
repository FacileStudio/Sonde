package uptime

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/testdb"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	url, ok := testdb.URL()
	if !ok {
		testdb.Announce("sh scripts/postgres-up.sh && export " + testdb.EnvVar + "=postgres://postgres:postgres@127.0.0.1:5603/sonde_test?sslmode=disable")
		os.Exit(0)
	}
	db, err := testdb.Open(url, testdb.Config{Prefix: "sonde_test", Migrate: applySchema})
	if err != nil {
		println("test setup failed: " + err.Error())
		os.Exit(1)
	}
	testDB = db
	os.Exit(m.Run())
}

func applySchema(db *gorm.DB) error {
	raw, err := os.ReadFile("../../migrations/00001_sonde.sql")
	if err != nil {
		return err
	}
	up := strings.Split(string(raw), "-- +goose Down")[0]
	up = strings.ReplaceAll(up, "CREATE TABLE ", "CREATE TABLE IF NOT EXISTS ")
	up = strings.ReplaceAll(up, "CREATE INDEX ", "CREATE INDEX IF NOT EXISTS ")
	for _, statement := range strings.Split(up, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func TestSummaryComputesWindowsOffHourlyRollups(t *testing.T) {
	if err := testdb.Truncate(testDB, testdb.Config{Prefix: "sonde_test"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	testDB.Exec(`INSERT INTO monitors (id, owner_id, slug, name, type) VALUES (1, 1, 'web', 'Web', 'http')`)

	buckets := []struct {
		age    time.Duration
		total  int64
		failed int64
	}{
		{age: time.Hour, total: 100, failed: 1},
		{age: 48 * time.Hour, total: 50, failed: 0},
		{age: 20 * 24 * time.Hour, total: 10, failed: 10},
		{age: 80 * 24 * time.Hour, total: 40, failed: 4},
	}
	for _, bucket := range buckets {
		if err := testDB.Exec(
			`INSERT INTO checks_hourly (monitor_id, bucket_start, total, failed, avg_latency_ms)
			 VALUES (1, ?, ?, ?, 100)`,
			now.Add(-bucket.age), bucket.total, bucket.failed,
		).Error; err != nil {
			t.Fatal(err)
		}
	}

	stats, err := NewService(testDB).Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("Summary() = %v", err)
	}

	want := map[string]struct {
		uptime float64
		total  int64
		failed int64
	}{
		"24h": {99, 100, 1},
		"7d":  {149.0 / 150 * 100, 150, 1},
		"30d": {149.0 / 160 * 100, 160, 11},
		"90d": {185.0 / 200 * 100, 200, 15},
	}
	if len(stats) != len(want) {
		t.Fatalf("got %d windows, want %d", len(stats), len(want))
	}
	for _, stat := range stats {
		expected := want[stat.Window]
		if stat.Uptime == nil || abs(*stat.Uptime-expected.uptime) > 0.001 {
			t.Errorf("%s uptime = %v, want %.2f", stat.Window, stat.Uptime, expected.uptime)
		}
		if stat.Total != expected.total || stat.Failed != expected.failed {
			t.Errorf("%s totals = (%d, %d), want (%d, %d)",
				stat.Window, stat.Total, stat.Failed, expected.total, expected.failed)
		}
	}
}

func TestEmptyWindowReportsNoPercentage(t *testing.T) {
	stats, err := NewService(testDB).Summary(context.Background(), 999)
	if err != nil {
		t.Fatalf("Summary() = %v", err)
	}
	for _, stat := range stats {
		if stat.Uptime != nil || stat.Total != 0 {
			t.Errorf("unknown monitor reported %v for window %s", stat.Uptime, stat.Window)
		}
	}
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
