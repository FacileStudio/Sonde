package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

func TestAFindingStaysOpenWhileAReportKeepsNamingIt(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")

	first := ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1"))))
	second := ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1"))))

	if first.NewFindings != 1 || second.NewFindings != 0 || second.Resolved != 0 || second.FindingsOpen != 1 {
		t.Fatalf("first = %+v, second = %+v", first, second)
	}
	if got := openActions(t, router, admin.Token); len(got.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(got.Actions))
	}
}

func TestAFlawThatFlipsBetweenItsCVEAndGHSANameIsNotNewOrResolved(t *testing.T) {
	router, _ := fleetRouter(t)

	ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1", "GHSA-aaaa-bbbb-cccc"))))
	flipped := ingest(t, router, nightly(covered(1, "a", flaw("GHSA-aaaa-bbbb-cccc", "CVE-2026-1"))))

	if flipped.NewFindings != 0 || flipped.Resolved != 0 || flipped.FindingsOpen != 1 {
		t.Fatalf("a flip changed the picture: %+v", flipped)
	}
}

func TestAFindingResolvesOnlyForProjectsTheRunCovered(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1")), covered(2, "b", flaw("CVE-2026-1"))))

	missed := fleet.ProjectReport{GithubRepoID: 2, Name: "b", Covered: false, NotCovered: "fetch failed"}
	report := nightly(covered(1, "a"), missed)
	report.Run.ReposScanned = 1
	result := ingest(t, router, report)

	if result.Resolved != 1 || result.FindingsOpen != 1 || result.Complete {
		t.Fatalf("result = %+v: only a's finding should resolve, and the run is incomplete", result)
	}
	actions := openActions(t, router, admin.Token)
	if len(actions.Actions) != 1 || len(actions.Actions[0].Projects) != 1 || actions.Actions[0].Projects[0].Name != "b" {
		t.Fatalf("actions = %+v, want one action left on b", actions.Actions)
	}
}

func TestAResolvedFindingReopensWhenItReturns(t *testing.T) {
	router, _ := fleetRouter(t)
	ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1"))))
	ingest(t, router, nightly(covered(1, "a")))

	back := ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1"))))

	if back.FindingsOpen != 1 {
		t.Fatalf("open = %d, want 1 after the finding came back", back.FindingsOpen)
	}
}

func TestTheSameFlawInTwoLockfilesIsOneFinding(t *testing.T) {
	router, _ := fleetRouter(t)
	twice := covered(1, "a", flaw("CVE-2026-1"), flaw("CVE-2026-1"))

	if got := ingest(t, router, nightly(twice)); got.FindingsOpen != 1 {
		t.Fatalf("open = %d, want 1", got.FindingsOpen)
	}
}

func TestClassifyingAProjectSurvivesTheNextScan(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	ingest(t, router, nightly(covered(1, "a")))
	var id int64
	db.Table("fleet_projects").Select("id").Where("github_repo_id = 1").Scan(&id)

	invalid := call(t, router, http.MethodPatch, "/api/fleet/projects/"+itoa(id), csrfHeaders(admin.Token), map[string]any{"tier": "boss"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid tier = %d, want 400", invalid.Code)
	}
	plain := call(t, router, http.MethodPatch, "/api/fleet/projects/"+itoa(id), csrfHeaders(admin.Token), map[string]any{"version_url": "http://insecure"})
	if plain.Code != http.StatusBadRequest {
		t.Fatalf("http version_url = %d, want 400", plain.Code)
	}
	set := call(t, router, http.MethodPatch, "/api/fleet/projects/"+itoa(id), csrfHeaders(admin.Token), map[string]any{"tier": "deployed"})
	if set.Code != http.StatusOK {
		t.Fatalf("set tier = %d: %s", set.Code, set.Body)
	}
	ingest(t, router, nightly(covered(1, "renamed")))

	var tier, name string
	db.Table("fleet_projects").Select("tier").Where("id = ?", id).Scan(&tier)
	db.Table("fleet_projects").Select("name").Where("id = ?", id).Scan(&name)
	if tier != "deployed" || name != "renamed" {
		t.Fatalf("tier = %q, name = %q: a scan must refresh the name and keep the tier", tier, name)
	}
}

func TestACookieMutationOnAFleetRouteNeedsTheCSRFHeader(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")

	response := call(t, router, http.MethodPost, "/api/fleet/policies", cookieHeader(admin.Token), map[string]any{})

	if response.Code != http.StatusForbidden {
		t.Fatalf("cookie mutation without CSRF = %d, want 403", response.Code)
	}
}

func TestHoldsNeedAReasonAndAnEndWithinAYear(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	post := func(body map[string]any) int {
		return call(t, router, http.MethodPost, "/api/fleet/policies", csrfHeaders(admin.Token), body).Code
	}
	soon := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	far := time.Now().Add(3 * 365 * 24 * time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	if got := post(map[string]any{"package": "@facile/muse", "held_version": "v0.10.0", "reason": "", "expires_at": soon}); got != http.StatusBadRequest {
		t.Errorf("no reason = %d, want 400", got)
	}
	if got := post(map[string]any{"package": "@facile/muse", "held_version": "v0.10.0", "reason": "old tokens", "expires_at": far}); got != http.StatusBadRequest {
		t.Errorf("end in three years = %d, want 400", got)
	}
	if got := post(map[string]any{"package": "@facile/muse", "held_version": "v0.10.0", "reason": "old tokens", "expires_at": past}); got != http.StatusBadRequest {
		t.Errorf("end in the past = %d, want 400", got)
	}
	if got := post(map[string]any{"package": "@facile/muse", "held_version": "v0.10.0", "reason": "old tokens", "expires_at": soon}); got != http.StatusCreated {
		t.Errorf("valid hold = %d, want 201", got)
	}
}

func TestDriftReadsPinsFromTheLatestReport(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	current := covered(1, "current")
	current.Pins = []fleet.PinReport{{Path: "apps/client/package.json", Package: "@facile/muse", Spec: "github:FacileStudio/muse#v0.16.1", Resolved: "v0.16.1"}}
	behind := covered(2, "behind")
	behind.Pins = []fleet.PinReport{{Path: "apps/client/package.json", Package: "@facile/muse", Spec: "github:FacileStudio/muse#v0.10.0", Resolved: "v0.10.0"}}
	ingest(t, router, nightly(current, behind))

	response := fleetGet(t, router, "/api/fleet/drift", admin.Token)

	var drift fleet.DriftResponse
	if err := json.Unmarshal(response.Body.Bytes(), &drift); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(drift.Packages) != 1 || drift.Packages[0].Current != 1 || len(drift.Packages[0].Behind) != 1 || drift.Packages[0].Behind[0].Projects[0].Name != "behind" {
		t.Fatalf("drift = %+v", drift)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
