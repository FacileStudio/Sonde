package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

type progressSink struct {
	mu      sync.Mutex
	updates []fleet.ProgressUpdate
	invalid int
}

func (s *progressSink) handle(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var update fleet.ProgressUpdate
	s.mu.Lock()
	if decoder.Decode(&update) != nil || update.Validate() != nil || r.Header.Get("Authorization") != "Bearer "+testToken {
		s.invalid++
	}
	s.updates = append(s.updates, update)
	s.mu.Unlock()
	http.Error(w, "progress is down", http.StatusInternalServerError)
}

func TestAFailingProgressEndpointNeverFailsTheScan(t *testing.T) {
	logs := quietLogs(t)
	github, sonde, sink := fakeGithub(t, completeFleet()), newFakeSonde(t, 0), &progressSink{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/fleet/scans/progress", sink.handle)
	mux.HandleFunc("/", sonde.handle)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	cfg := testConfig(github, sonde, fakeDouane(t, douaneEmitting(coveringSweep, 1)))
	cfg.sondeURL, cfg.runID, cfg.progressEvery = server.URL, 42, time.Millisecond

	if err := scan(context.Background(), cfg); err != nil {
		t.Fatalf("scan: %v: a progress endpoint answering 500 must not fail the run", err)
	}
	onlyReport(t, sonde)
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.updates) == 0 || sink.invalid != 0 || sink.updates[0].RunID != 42 {
		t.Fatalf("updates = %+v, %d invalid: the runner posts valid updates for its run", sink.updates, sink.invalid)
	}
	if strings.Contains(logs.String(), "progress") || strings.Contains(logs.String(), "docs") {
		t.Fatalf("progress or a repository name reached the public log:\n%s", logs.String())
	}
}

func TestProgressIsOffOutsideAWorkflowRun(t *testing.T) {
	cfg := config{sondeURL: "http://sonde", fleetToken: testToken}
	if startProgress(context.Background(), cfg) != nil {
		t.Fatal("no GITHUB_RUN_ID: progress must be off")
	}
	cfg.runID, cfg.out = 1, "report.json"
	if startProgress(context.Background(), cfg) != nil {
		t.Fatal("-out: progress must be off")
	}
	var off *progress
	off.phase(fleet.PhaseScanning, 3)
	off.started("x")
	off.finished()
}
