package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const testToken = "fleet-token-for-tests-0123456789abcdef"

var apiFiles = map[string]string{
	"go.mod":                      "module x\n\ngo 1.26\n\nrequire (\n\tgithub.com/FacileStudio/tronc v0.16.0\n\tgithub.com/FacileStudio/porte v0.13.0\n)\n",
	"apps/client/package.json":    `{"dependencies":{"@facile/muse":"github:FacileStudio/muse#v0.10.0"}}`,
	"apps/client/bun.lock":        "{}",
	".github/workflows/ci.yml":    "jobs:\n  lint:\n    steps:\n      - uses: FacileStudio/filet@v0.3.0\n",
	"node_modules/x/package.json": "{}",
	"README.md":                   "# api",
}

func fakeGithub(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/Acme/repos", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]any{
			{"id": 11, "name": "api", "full_name": "Acme/api", "default_branch": "main"},
			{"id": 12, "name": "docs", "full_name": "Acme/docs", "default_branch": "main"},
			{"id": 13, "name": "old", "full_name": "Acme/old", "default_branch": "main", "archived": true},
			{"id": 14, "name": "broken", "full_name": "Acme/broken", "default_branch": "main"},
		})
	})
	mux.HandleFunc("/repos/Acme/api/git/trees/main", func(w http.ResponseWriter, _ *http.Request) {
		var tree []map[string]string
		for path := range apiFiles {
			tree = append(tree, map[string]string{"path": path, "type": "blob"})
		}
		writeJSON(w, map[string]any{"tree": tree})
	})
	mux.HandleFunc("/repos/Acme/api/contents/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/repos/Acme/api/contents/")
		content, ok := apiFiles[path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(content))
	})
	mux.HandleFunc("/repos/Acme/api/actions/runs", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"workflow_runs": []map[string]string{{"conclusion": "success"}}})
	})
	mux.HandleFunc("/repos/Acme/docs/git/trees/main", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"tree": []map[string]string{{"path": "README.md", "type": "blob"}}})
	})
	mux.HandleFunc("/repos/FacileStudio/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]string{{"name": "v0.9.0"}, {"name": "v0.16.1"}, {"name": "v0.17.0-rc1"}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

type fakeSonde struct {
	server   *httptest.Server
	received []fleet.Report
}

func newFakeSonde(t *testing.T) *fakeSonde {
	t.Helper()
	sonde := &fakeSonde{}
	sonde.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "not gzip", http.StatusBadRequest)
			return
		}
		defer func() { _ = reader.Close() }()
		decoder := json.NewDecoder(reader)
		decoder.DisallowUnknownFields()
		var report fleet.Report
		if err := decoder.Decode(&report); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := report.Validate(time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sonde.received = append(sonde.received, report)
		writeJSON(w, fleet.IngestResult{RunID: 7, Complete: report.Run.ReposScanned == report.Run.ReposExpected})
	}))
	t.Cleanup(sonde.server.Close)
	return sonde
}

func fakeDouane(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "douane")
	script := "#!/bin/sh\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

const goodDouane = `
if [ "$1" = "--version" ]; then echo "douane 9.9.9"; exit 0; fi
[ -d "$2/api/.git" ] && [ -f "$2/api/go.mod" ] && [ -f "$2/api/apps/client/bun.lock" ] || exit 2
[ -e "$2/api/node_modules" ] && exit 2
cat <<'JSON'
{"repos":[{"name":"api","findings":[{"id":"CVE-2026-1","aliases":["GHSA-aaaa"],"summary":"boom","severity":"HIGH","package":"next","ecosystem":"npm","installed":"16.2.4","fixed_in":"16.2.5","target":"apps/client/bun.lock","exploit":{"kev":false,"epss":0.355,"scope":"prod"}}]}]}
JSON
exit 1
`

func testConfig(github *httptest.Server, sonde *fakeSonde, douane string) config {
	cfg := config{
		org: "Acme", githubAPI: github.URL, githubToken: "gh-token", douaneBin: douane, workers: 2,
	}
	if sonde != nil {
		cfg.sondeURL, cfg.fleetToken = sonde.server.URL, testToken
	}
	return cfg
}

func quietLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	log.SetOutput(&buffer)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buffer
}

func projectNamed(t *testing.T, report fleet.Report, name string) fleet.ProjectReport {
	t.Helper()
	for _, project := range report.Projects {
		if project.Name == name {
			return project
		}
	}
	t.Fatalf("no project %q in %+v", name, report.Projects)
	return fleet.ProjectReport{}
}

func TestAScanDeliversAReportTheServerAccepts(t *testing.T) {
	logs := quietLogs(t)
	github, sonde := fakeGithub(t), newFakeSonde(t)

	if err := scan(context.Background(), testConfig(github, sonde, fakeDouane(t, goodDouane))); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(sonde.received) != 1 {
		t.Fatalf("reports delivered = %d, want 1", len(sonde.received))
	}
	report := sonde.received[0]
	if report.Run.DouaneVersion != "9.9.9" || report.Run.ReposExpected != 2 || report.Run.ReposScanned != 1 {
		t.Fatalf("run = %+v: archived and nothing-to-scan repos must not be expected, a failed one must be", report.Run)
	}
	api := projectNamed(t, report, "api")
	if !api.Covered || len(api.Findings) != 1 || api.CIConclusion != "success" || !api.FiletWired || api.FiletPin != "v0.3.0" || api.GoDirective != "1.26" {
		t.Fatalf("api = %+v", api)
	}
	if len(api.Pins) != 3 {
		t.Fatalf("pins = %+v, want muse, tronc and porte", api.Pins)
	}
	if got := projectNamed(t, report, "docs"); got.Covered || got.NotCovered != "no lockfile" {
		t.Fatalf("docs = %+v", got)
	}
	if got := projectNamed(t, report, "broken"); got.Covered || !strings.HasPrefix(got.NotCovered, "tree failed") {
		t.Fatalf("broken = %+v", got)
	}
	for _, project := range report.Projects {
		if project.Name == "old" {
			t.Fatal("an archived repository was reported")
		}
	}
	if report.Latest["@facile/muse"] != "v0.16.1" {
		t.Fatalf("latest = %v, want the highest release tag", report.Latest)
	}
	if strings.Contains(logs.String(), "CVE-") || strings.Contains(logs.String(), "next") {
		t.Fatalf("the log carries finding detail:\n%s", logs.String())
	}
}

func TestAFailingScannerDeliversNothing(t *testing.T) {
	quietLogs(t)
	github, sonde := fakeGithub(t), newFakeSonde(t)

	crashed := fakeDouane(t, `if [ "$1" = "--version" ]; then echo "douane 9.9.9"; exit 0; fi
echo '{"repos":[{"name":"api","findings":[]}]}'
exit 2
`)

	err := scan(context.Background(), testConfig(github, sonde, crashed))

	if err == nil {
		t.Fatal("a scanner that crashed was treated as a clean scan")
	}
	if len(sonde.received) != 0 {
		t.Fatalf("a report was delivered after the scanner failed: %d", len(sonde.received))
	}
}

func TestARepoTheScannerSkippedIsNotReportedAsClean(t *testing.T) {
	quietLogs(t)
	github, sonde := fakeGithub(t), newFakeSonde(t)
	silent := fakeDouane(t, `if [ "$1" = "--version" ]; then echo "douane 9.9.9"; exit 0; fi
echo '{"repos":[]}'
`)

	if err := scan(context.Background(), testConfig(github, sonde, silent)); err != nil {
		t.Fatalf("scan: %v", err)
	}

	api := projectNamed(t, sonde.received[0], "api")
	if api.Covered || !strings.Contains(api.NotCovered, "douane returned nothing") {
		t.Fatalf("api = %+v: a repo douane never mentioned must read as not covered", api)
	}
	if sonde.received[0].Run.ReposScanned != 0 {
		t.Fatalf("scanned = %d, want 0", sonde.received[0].Run.ReposScanned)
	}
}

func TestOutWritesTheReportInsteadOfDeliveringIt(t *testing.T) {
	quietLogs(t)
	github, sonde := fakeGithub(t), newFakeSonde(t)
	cfg := testConfig(github, sonde, fakeDouane(t, goodDouane))
	cfg.out = filepath.Join(t.TempDir(), "report.json")

	if err := scan(context.Background(), cfg); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(sonde.received) != 0 {
		t.Fatal("-out still delivered the report")
	}
	var written fleet.Report
	data, err := os.ReadFile(cfg.out)
	if err != nil || json.Unmarshal(data, &written) != nil || len(written.Projects) != 3 {
		t.Fatalf("written report unreadable or wrong: %v", err)
	}
}

func TestADeliverySondeRefusesIsAFailure(t *testing.T) {
	quietLogs(t)
	github, sonde := fakeGithub(t), newFakeSonde(t)
	cfg := testConfig(github, sonde, fakeDouane(t, goodDouane))
	cfg.fleetToken = "wrong-token-wrong-token-wrong-token-1"

	if err := scan(context.Background(), cfg); err == nil {
		t.Fatal("a refused delivery was reported as success")
	}
}

func TestConfigRefusesToStartWithoutWhatItNeeds(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("SONDE_URL", "")
	t.Setenv("SONDE_FLEET_TOKEN", "")
	if _, err := loadConfig(nil); err == nil {
		t.Error("no GitHub token was accepted")
	}
	t.Setenv("GITHUB_TOKEN", "x")
	if _, err := loadConfig(nil); err == nil {
		t.Error("no Sonde target and no -out was accepted")
	}
	if _, err := loadConfig([]string{"-out", "report.json"}); err != nil {
		t.Errorf("-out alone should be enough: %v", err)
	}
	if _, err := loadConfig([]string{"-out", "report.json", "-workers", "0"}); err == nil {
		t.Error("zero workers was accepted")
	}
}
