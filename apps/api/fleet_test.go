package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const fleetToken = "fleet-token-for-tests-0123456789abcdef"

var fleetAdmins = []string{"admin@facile.studio", "listed@facile.studio"}

func fleetRouter(t *testing.T) (chi.Router, *gorm.DB) {
	t.Helper()
	return liveRouter(t, func(c *env.Config) {
		c.FleetToken = fleetToken
		c.FleetAdmins = fleetAdmins
	})
}

func gzipJSON(t *testing.T, value any) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip: %v", err)
	}
	return buffer.Bytes()
}

func deliver(t *testing.T, router chi.Router, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/fleet/reports", bytes.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func ingest(t *testing.T, router chi.Router, report fleet.Report) fleet.IngestResult {
	t.Helper()
	response := deliver(t, router, fleetToken, gzipJSON(t, report))
	if response.Code != http.StatusOK {
		t.Fatalf("ingest = %d: %s", response.Code, response.Body)
	}
	var result fleet.IngestResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return result
}

func nightly(projects ...fleet.ProjectReport) fleet.Report {
	now := time.Now().UTC()
	return fleet.Report{
		SchemaVersion: fleet.SchemaVersion,
		Run: fleet.RunInfo{
			StartedAt: now.Add(-time.Minute), FinishedAt: now, DouaneVersion: "0.6.1",
			ReposExpected: len(projects), ReposScanned: len(projects),
		},
		Projects: projects,
		Latest:   map[string]string{"@facile/muse": "v0.16.1"},
	}
}

func covered(id int64, name string, findings ...fleet.FindingReport) fleet.ProjectReport {
	return fleet.ProjectReport{GithubRepoID: id, Name: name, DefaultBranch: "main", Covered: true, Findings: findings}
}

func flaw(id string, aliases ...string) fleet.FindingReport {
	return fleet.FindingReport{
		ID: id, Aliases: aliases, Severity: "HIGH", Package: "next", Ecosystem: "npm",
		Installed: "16.2.4", FixedIn: "16.2.5", EPSS: 0.355,
	}
}

func fleetGet(t *testing.T, router chi.Router, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	return call(t, router, http.MethodGet, path, cookieHeader(token), nil)
}

func openActions(t *testing.T, router chi.Router, token string) fleet.ActionsResponse {
	t.Helper()
	response := fleetGet(t, router, "/api/fleet/actions", token)
	if response.Code != http.StatusOK {
		t.Fatalf("actions = %d: %s", response.Code, response.Body)
	}
	var out fleet.ActionsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode actions: %v", err)
	}
	return out
}

func TestEveryFleetRouteRefusesAnAnonymousCaller(t *testing.T) {
	router, _ := fleetRouter(t)
	checked := 0

	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/api/fleet/") {
			return nil
		}
		path := strings.ReplaceAll(route, "{id}", "1")
		response := call(t, router, method, path, nil, map[string]any{})
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous = %d, want 401", method, route, response.Code)
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if checked < 9 {
		t.Fatalf("checked %d fleet routes, want at least 9: the walk missed some", checked)
	}
}

func TestFleetReadsNeedAnAdminWhoIsAlsoAllowListed(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	listedButNotAdmin := signIn(t, db, "listed@facile.studio")
	stranger := signIn(t, db, "stranger@facile.studio")
	if !admin.User.IsAdmin || listedButNotAdmin.User.IsAdmin || stranger.User.IsAdmin {
		t.Fatalf("setup: admin flags are %v %v %v", admin.User.IsAdmin, listedButNotAdmin.User.IsAdmin, stranger.User.IsAdmin)
	}

	if got := fleetGet(t, router, "/api/fleet/actions", admin.Token).Code; got != http.StatusOK {
		t.Fatalf("allow-listed admin = %d, want 200", got)
	}
	if got := fleetGet(t, router, "/api/fleet/actions", listedButNotAdmin.Token).Code; got != http.StatusForbidden {
		t.Fatalf("allow-listed non-admin = %d, want 403", got)
	}
	if got := fleetGet(t, router, "/api/fleet/actions", stranger.Token).Code; got != http.StatusForbidden {
		t.Fatalf("plain user = %d, want 403", got)
	}
}

func TestPromotingAStrangerToAdminDoesNotOpenFleetData(t *testing.T) {
	router, db := fleetRouter(t)
	signIn(t, db, "admin@facile.studio")
	stranger := signIn(t, db, "stranger@facile.studio")
	if err := db.Exec(`UPDATE users SET is_admin = true WHERE id = ?`, stranger.User.ID).Error; err != nil {
		t.Fatalf("promote: %v", err)
	}

	for _, path := range []string{"/api/fleet/actions", "/api/fleet/drift", "/api/fleet/graph", "/api/fleet/conformance", "/api/fleet/projects", "/api/fleet/runs", "/api/fleet/policies"} {
		if got := fleetGet(t, router, path, stranger.Token).Code; got != http.StatusForbidden {
			t.Errorf("promoted stranger on %s = %d, want 403", path, got)
		}
	}
}

func TestAnEmptyAllowListClosesTheReadRoutes(t *testing.T) {
	router, db := liveRouter(t, func(c *env.Config) { c.FleetToken = fleetToken })
	admin := signIn(t, db, "admin@facile.studio")

	if got := fleetGet(t, router, "/api/fleet/actions", admin.Token).Code; got != http.StatusForbidden {
		t.Fatalf("admin with no allow-list = %d, want 403", got)
	}
}

func TestIngestNeedsTheTokenAndGzip(t *testing.T) {
	router, _ := fleetRouter(t)
	body := gzipJSON(t, nightly(covered(1, "a")))

	if got := deliver(t, router, "", body).Code; got != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got)
	}
	if got := deliver(t, router, "wrong-token-wrong-token-wrong-token-1", body).Code; got != http.StatusUnauthorized {
		t.Errorf("wrong token = %d, want 401", got)
	}
	plain, _ := json.Marshal(nightly(covered(1, "a")))
	if got := deliver(t, router, fleetToken, plain).Code; got != http.StatusBadRequest {
		t.Errorf("uncompressed body = %d, want 400", got)
	}
	if got := deliver(t, router, fleetToken, body).Code; got != http.StatusOK {
		t.Errorf("valid delivery = %d, want 200", got)
	}
}

func TestIngestIsOffUntilATokenIsConfigured(t *testing.T) {
	router, _ := liveRouter(t, nil)

	if got := deliver(t, router, fleetToken, gzipJSON(t, nightly())).Code; got != http.StatusNotFound {
		t.Fatalf("ingest with no token configured = %d, want 404", got)
	}
}

func TestAnInvalidReportChangesNothing(t *testing.T) {
	router, db := fleetRouter(t)
	report := nightly(covered(1, "a", flaw("CVE-2026-1")))
	report.Projects = append(report.Projects, fleet.ProjectReport{GithubRepoID: 1, Name: "duplicate", Covered: true})

	if got := deliver(t, router, fleetToken, gzipJSON(t, report)).Code; got != http.StatusBadRequest {
		t.Fatalf("duplicate repo = %d, want 400", got)
	}
	var rows int64
	db.Table("fleet_projects").Count(&rows)
	if rows != 0 {
		t.Fatalf("a refused report wrote %d projects", rows)
	}
}

func TestAnArchivedRepositoryLeavesTheInbox(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	ingest(t, router, nightly(covered(7, "Chat", flaw("GHSA-chat"))))
	if got := openActions(t, router, admin.Token).Counts; got.Act+got.Schedule != 1 {
		t.Fatalf("before archiving: %+v, want one act or schedule", got)
	}

	archived := fleet.ProjectReport{GithubRepoID: 7, Name: "Chat", DefaultBranch: "main", NotCovered: "archived"}
	ingest(t, router, nightly(archived))

	if got := openActions(t, router, admin.Token).Counts; got.Act+got.Schedule != 0 || got.Track != 1 {
		t.Fatalf("after archiving: %+v, want only one track", got)
	}
	var tier string
	if err := db.Raw(`SELECT tier FROM fleet_projects WHERE github_repo_id = 7`).Scan(&tier).Error; err != nil || tier != "archive" {
		t.Fatalf("tier = %q (%v), want archive", tier, err)
	}
}
