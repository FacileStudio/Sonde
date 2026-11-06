package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
)

// RecordCheck writes the raw row and the hourly bucket in one transaction, and
// the bucket is the half that only Postgres can judge: date_trunc is overloaded,
// so two untyped arguments make the call ambiguous and the whole transaction
// rolls back, raw row included. That shipped, because every other test in the
// package drives the engine through a fake recorder and this statement had never
// reached a database. A probe ran every minute against nine monitors and stored
// nothing, and the API reported an honest zero the whole time.
func TestRecordCheckStoresTheRawRowAndTheHourlyBucket(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, db)

	created := call(t, router, http.MethodPost, "/api/monitors", csrfHeaders(session.Token), map[string]any{
		"slug": "website", "name": "Website", "type": "http", "target": "https://facile.studio",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body)
	}
	var monitor struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &monitor); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	recorder := checker.NewGormRecorder(db)
	if err := recorder.RecordCheck(context.Background(), monitor.ID, true, 42, "", nil); err != nil {
		t.Fatalf("RecordCheck: %v", err)
	}

	var raw int64
	if err := db.Raw(`SELECT count(*) FROM checks_raw WHERE monitor_id = ?`, monitor.ID).Scan(&raw).Error; err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if raw != 1 {
		t.Errorf("checks_raw rows = %d, want 1", raw)
	}

	var total, failed int64
	if err := db.Raw(
		`SELECT total, failed FROM checks_hourly WHERE monitor_id = ?`, monitor.ID,
	).Row().Scan(&total, &failed); err != nil {
		t.Fatalf("read the hourly bucket: %v", err)
	}
	if total != 1 || failed != 0 {
		t.Errorf("hourly bucket = (total %d, failed %d), want (1, 0)", total, failed)
	}

	if err := recorder.RecordCheck(context.Background(), monitor.ID, false, 0, "boom", nil); err != nil {
		t.Fatalf("RecordCheck on a failure: %v", err)
	}
	if err := db.Raw(
		`SELECT total, failed FROM checks_hourly WHERE monitor_id = ?`, monitor.ID,
	).Row().Scan(&total, &failed); err != nil {
		t.Fatalf("re-read the hourly bucket: %v", err)
	}
	if total != 2 || failed != 1 {
		t.Errorf("hourly bucket after a failure = (total %d, failed %d), want (2, 1)", total, failed)
	}
}
