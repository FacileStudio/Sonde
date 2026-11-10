package retention

import (
	"context"
	"log/slog"
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
	for _, file := range []string{"00001_sonde.sql", "00006_fleet.sql"} {
		raw, err := os.ReadFile("../../migrations/" + file)
		if err != nil {
			return err
		}
		if err := applyUp(db, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func applyUp(db *gorm.DB, script string) error {
	up := strings.Split(script, "-- +goose Down")[0]
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

func mustExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if err := testDB.Exec(query, args...).Error; err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func count(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := testDB.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func TestRunPrunesRawAndRollsUp(t *testing.T) {
	now := seedChecks(t)

	job := New(testDB, DefaultRawRetentionDays, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v", err)
	}

	checkRawPruned(t)
	checkHourlyRollup(t, now)
	checkDailyRollup(t, now)
}

func seedChecks(t *testing.T) time.Time {
	t.Helper()
	testdb.Truncate(testDB, testdb.Config{Prefix: "sonde_test"})
	now := time.Now()
	mustExec(t, `INSERT INTO monitors (id, owner_id, slug, name, type) VALUES (1, 1, 'web', 'Web', 'http')`)
	mustExec(t, `INSERT INTO checks_raw (monitor_id, ok, latency_ms, created_at)
	             SELECT 1, true, 100, ? FROM generate_series(1, 5)`, now.AddDate(0, 0, -40))
	mustExec(t, `INSERT INTO checks_raw (monitor_id, ok, latency_ms, created_at) VALUES (1, false, 900, ?)`, now.Add(-2*time.Hour))
	mustExec(t, `INSERT INTO checks_raw (monitor_id, ok, latency_ms, created_at) VALUES (1, true, 50, ?)`, now)
	mustExec(t, `INSERT INTO checks_hourly (monitor_id, bucket_start, total, failed, avg_latency_ms)
	             VALUES (1, ?, 10, 4, 200)`, now.AddDate(-1, -2, 0))
	return now
}

func checkRawPruned(t *testing.T) {
	t.Helper()
	if got := count(t, `SELECT COUNT(*) FROM checks_raw WHERE created_at < date_trunc('hour', now()) - make_interval(days => 30)`); got != 0 {
		t.Fatalf("old raw rows survived: %d", got)
	}
	if got := count(t, `SELECT COUNT(*) FROM checks_raw`); got != 2 {
		t.Fatalf("raw rows = %d, want only the two within retention", got)
	}
}

func checkHourlyRollup(t *testing.T, now time.Time) {
	t.Helper()
	hourlyBucket := now.AddDate(0, 0, -40).Truncate(time.Hour)
	if got := count(t, `SELECT total FROM checks_hourly WHERE monitor_id = 1 AND bucket_start = ?`, hourlyBucket); got != 5 {
		t.Fatalf("hourly rollup from raw = %d, want 5", got)
	}
	failed := count(t, `SELECT failed FROM checks_hourly WHERE bucket_start >= date_trunc('hour', now()) - interval '3 hours' AND failed > 0`)
	if failed == 0 {
		t.Fatal("the failing check was not aggregated into its hourly bucket")
	}
}

func checkDailyRollup(t *testing.T, now time.Time) {
	t.Helper()
	dailyBucket := now.AddDate(-1, -2, 0)
	if got := count(t, `SELECT total FROM checks_daily WHERE monitor_id = 1
	                    AND bucket_start >= date_trunc('day', ?::timestamptz)
	                    AND bucket_start < date_trunc('day', ?::timestamptz) + interval '1 day'`,
		dailyBucket, dailyBucket); got != 10 {
		t.Fatalf("daily rollup from old hourly = %d, want 10", got)
	}
	cutoff := now.AddDate(0, -13, 0)
	if got := count(t, `SELECT COUNT(*) FROM checks_hourly WHERE bucket_start < ?`, cutoff); got != 0 {
		t.Fatalf("13-month-old hourly rollups survived: %d", got)
	}
}

func TestRunPrunesResolvedFleetFindingsAndOldRunsOnly(t *testing.T) {
	mustExec(t, `TRUNCATE fleet_findings, fleet_runs, fleet_projects RESTART IDENTITY CASCADE`)
	mustExec(t, `INSERT INTO fleet_projects (github_repo_id, name) VALUES (1, 'a')`)
	insert := `INSERT INTO fleet_findings (project_id, key, advisory, package, ecosystem, first_seen, last_seen, resolved_at)
		VALUES (1, ?, 'CVE', 'p', 'npm', now(), now(), ?)`
	mustExec(t, insert, "resolved-long-ago", gorm.Expr("now() - interval '100 days'"))
	mustExec(t, insert, "resolved-recently", gorm.Expr("now() - interval '10 days'"))
	mustExec(t, insert, "open-and-old", nil)
	mustExec(t, `UPDATE fleet_findings SET first_seen = now() - interval '400 days' WHERE key = 'open-and-old'`)
	run := `INSERT INTO fleet_runs (started_at, finished_at, repos_expected, repos_scanned, complete, created_at)
		VALUES (now(), now(), 1, 1, true, ?)`
	mustExec(t, run, gorm.Expr("now() - interval '100 days'"))
	mustExec(t, run, gorm.Expr("now() - interval '1 day'"))

	if err := New(testDB, 30, nil).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := count(t, `SELECT count(*) FROM fleet_findings`); got != 2 {
		t.Fatalf("findings = %d, want 2: the old resolved one goes, the recent and the open ones stay", got)
	}
	if got := count(t, `SELECT count(*) FROM fleet_findings WHERE key = 'open-and-old'`); got != 1 {
		t.Fatal("an open finding was pruned for being old")
	}
	if got := count(t, `SELECT count(*) FROM fleet_runs`); got != 1 {
		t.Fatalf("runs = %d, want 1", got)
	}
}
