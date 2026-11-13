package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

func TestACompleteRunDeletesAProjectItNoLongerLists(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	chat := covered(7, "Chat", flaw("GHSA-chat"))
	chat.Edges = []fleet.EdgeReport{{To: "repo-0", Kind: "call", Auth: "url", Evidence: "REPO_0_URL"}}
	rest := fleetOf(9)
	for i := range rest {
		rest[i].Findings = nil
	}
	rest[0].Edges = []fleet.EdgeReport{{To: "Chat", Kind: "call", Auth: "url", Evidence: "CHAT_URL"}}
	ingest(t, router, nightly(append([]fleet.ProjectReport{chat}, rest...)...))

	result := ingest(t, router, nightly(rest...))

	if result.Resolved != 1 || len(result.Removed) != 1 || result.Removed[0] != "Chat" {
		t.Fatalf("result = %+v: Chat left the listing, so it is removed and its finding counted resolved", result)
	}
	if got := openActions(t, router, admin.Token); len(got.Actions) != 0 {
		t.Fatalf("actions = %+v: nothing of Chat stays in the inbox", got.Actions)
	}
	var projects, findings, edges int64
	db.Table("projects").Where("github_repo_id = 7").Count(&projects)
	db.Table("fleet_findings").Count(&findings)
	db.Table("fleet_edges").Where("lower(to_name) = 'chat'").Count(&edges)
	if projects+findings+edges != 0 {
		t.Fatalf("left behind: %d projects, %d findings, %d edges to Chat", projects, findings, edges)
	}
}

func TestTheRunnersProgressReachesTheCurrentScan(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	update := map[string]any{"run_id": 77, "phase": "scanning", "repos_done": 12, "repos_total": 94, "current": "Sonde"}
	anonymous := call(t, router, http.MethodPost, "/api/fleet/scans/progress", nil, update)
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("progress without the token = %d, want 401", anonymous.Code)
	}
	posted := call(t, router, http.MethodPost, "/api/fleet/scans/progress", map[string]string{"Authorization": "Bearer " + fleetToken}, update)
	if posted.Code != http.StatusNoContent {
		t.Fatalf("progress = %d: %s", posted.Code, posted.Body)
	}

	response := fleetGet(t, router, "/api/fleet/scans/current", admin.Token)
	var status fleet.ScanStatus
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil || response.Code != http.StatusOK {
		t.Fatalf("current = %d %s (%v)", response.Code, response.Body, err)
	}
	if status.State != fleet.ScanInProgress || status.Progress == nil || status.Progress.ReposDone != 12 || status.Progress.Current != "Sonde" {
		t.Fatalf("status = %+v: without the App the runner's progress still shows", status)
	}
}
