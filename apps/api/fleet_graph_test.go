package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func graphFixture(t *testing.T) fleet.Report {
	t.Helper()
	data, err := os.ReadFile("testdata/fleet-graph-projects.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var projects []fleet.ProjectReport
	if err := json.Unmarshal(data, &projects); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	report := nightly(projects...)
	report.Latest["github.com/FacileStudio/tronc"] = "v0.17.1"
	return report
}

func readFleet[T any](t *testing.T, router chi.Router, path, token string) T {
	t.Helper()
	response := fleetGet(t, router, path, token)
	if response.Code != http.StatusOK {
		t.Fatalf("%s = %d: %s", path, response.Code, response.Body)
	}
	var out T
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

func nodeNamed(t *testing.T, graph fleet.GraphResponse, name string) fleet.GraphNode {
	t.Helper()
	for _, node := range graph.Nodes {
		if node.Name == name {
			return node
		}
	}
	t.Fatalf("no node %q in %+v", name, graph.Nodes)
	return fleet.GraphNode{}
}

func downMonitor(t *testing.T, db *gorm.DB, owner int64, name string) {
	t.Helper()
	var id int64
	if err := db.Raw(`INSERT INTO monitors (owner_id, slug, name, type) VALUES (?, ?, ?, 'http') RETURNING id`, owner, name, name).Scan(&id).Error; err != nil {
		t.Fatalf("monitor: %v", err)
	}
	if err := db.Exec(`INSERT INTO checks_raw (monitor_id, ok) VALUES (?, false)`, id).Error; err != nil {
		t.Fatalf("check: %v", err)
	}
}

func TestTheGraphAndConformanceReadBackWhatTheRunnerDerived(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	downMonitor(t, db, admin.User.ID, "courrier")
	ingest(t, router, graphFixture(t))

	graph := readFleet[fleet.GraphResponse](t, router, "/api/fleet/graph", admin.Token)

	if len(graph.Edges) != 4 || len(graph.Nodes) != 4 {
		t.Fatalf("graph = %+v: three projects plus Antenne, four distinct edges", graph)
	}
	courrier := nodeNamed(t, graph, "Courrier")
	if courrier.ChecksOK != 1 || courrier.ChecksTotal != 2 || courrier.OpenAct != 1 || courrier.UptimeStatus != "down" {
		t.Fatalf("Courrier = %+v", courrier)
	}
	if antenne := nodeNamed(t, graph, "Antenne"); antenne.ID != 0 {
		t.Fatalf("Antenne = %+v: a target the runner never listed has no project id", antenne)
	}
	if tronc := nodeNamed(t, graph, "tronc"); tronc.LatestTag != "v0.17.1" || tronc.ID == 0 {
		t.Fatalf("tronc = %+v", tronc)
	}

	matrix := readFleet[fleet.ConformanceResponse](t, router, "/api/fleet/conformance", admin.Token)
	if len(matrix.Projects) != 3 || matrix.Projects[0].Name != "Courrier" || matrix.Projects[2].Name != "Nuage" || len(matrix.Checks) != 3 {
		t.Fatalf("matrix = %+v: worst first, then by name", matrix)
	}
}

func TestACoveredProjectReplacesItsGraphAndAnUncoveredOneKeepsIt(t *testing.T) {
	router, db := fleetRouter(t)
	admin := signIn(t, db, "admin@facile.studio")
	ingest(t, router, graphFixture(t))
	next := graphFixture(t)
	next.Projects[0] = fleet.ProjectReport{GithubRepoID: 201, Name: "Courrier", Covered: false, NotCovered: "fetch failed"}
	next.Projects[2].Conformance = nil
	next.Run.ReposScanned = 2
	ingest(t, router, next)

	matrix := readFleet[fleet.ConformanceResponse](t, router, "/api/fleet/conformance", admin.Token)
	graph := readFleet[fleet.GraphResponse](t, router, "/api/fleet/graph", admin.Token)

	if len(matrix.Projects) != 2 || matrix.Projects[0].Name != "Courrier" || matrix.Projects[1].Name != "Nuage" {
		t.Fatalf("matrix = %+v: Courrier keeps its checks, tronc lost them", matrix)
	}
	if len(graph.Edges) != 4 {
		t.Fatalf("edges = %+v: an uncovered project keeps its edges", graph.Edges)
	}
}

func TestAnEdgeOrCheckTheServerCannotStoreIsRefused(t *testing.T) {
	router, _ := fleetRouter(t)
	cases := map[string]fleet.ProjectReport{
		"unknown kind":      {Edges: []fleet.EdgeReport{{To: "x", Kind: "friend"}}},
		"call without auth": {Edges: []fleet.EdgeReport{{To: "x", Kind: "call"}}},
		"lib with auth":     {Edges: []fleet.EdgeReport{{To: "x", Kind: "lib", Auth: "url"}}},
		"NUL evidence":      {Edges: []fleet.EdgeReport{{To: "x", Kind: "lib", Evidence: "a\x00b"}}},
		"empty check id":    {Conformance: []fleet.CheckReport{{ID: " "}}},
	}
	for name, project := range cases {
		project.GithubRepoID, project.Name, project.Covered = 1, "a", true
		if got := deliver(t, router, fleetToken, gzipJSON(t, nightly(project))).Code; got != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, got)
		}
	}
	uncovered := fleet.ProjectReport{GithubRepoID: 1, Name: "a", Conformance: []fleet.CheckReport{{ID: "gate.filet"}}}
	if got := deliver(t, router, fleetToken, gzipJSON(t, nightly(uncovered))).Code; got != http.StatusBadRequest {
		t.Errorf("checks on an uncovered project = %d, want 400", got)
	}
}
