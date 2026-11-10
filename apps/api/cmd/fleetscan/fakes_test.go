package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const testToken = "fleet-token-for-tests-0123456789abcdef"

type fakeRepo struct {
	id       int64
	name     string
	branch   string
	archived bool
	files    map[string]string
	missing  bool
}

var goModFile = "module x\n\ngo 1.26\n\nrequire (\n\tgithub.com/FacileStudio/tronc v0.16.0\n\tgithub.com/FacileStudio/porte v0.13.0\n)\n"

func fleetRepos() []fakeRepo {
	return []fakeRepo{
		{id: 11, name: "api", branch: "main", files: map[string]string{
			"go.mod":                      goModFile,
			"apps/client/package.json":    `{"dependencies":{"@facile/muse":"github:FacileStudio/muse#v0.10.0"}}`,
			"apps/client/bun.lock":        "{}",
			".github/workflows/ci.yml":    "jobs:\n  lint:\n    steps:\n      - uses: FacileStudio/filet@v0.3.0\n",
			"node_modules/x/package.json": "{}",
			"README.md":                   "# api",
			"docker-compose.yml":          "services:\n  api:\n    environment:\n      JOURNAL_URL: http://journal\n",
		}},
		{id: 12, name: "docs", branch: "main", files: map[string]string{"README.md": "# docs"}},
		{id: 13, name: "old", branch: "main", archived: true, files: map[string]string{"go.mod": goModFile}},
		{id: 14, name: "broken", branch: "main", missing: true},
		{id: 15, name: "yarny", branch: "main", files: map[string]string{"yarn.lock": "# yarn"}},
		{id: 16, name: ".github", branch: "main", files: map[string]string{"go.mod": goModFile}},
		{id: 17, name: "slashy", branch: "release/x", files: map[string]string{"go.mod": goModFile}},
	}
}

func fakeGithub(t *testing.T, repos []fakeRepo) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	listing := make([]map[string]any, 0, len(repos))
	for _, repo := range repos {
		listing = append(listing, map[string]any{
			"id": repo.id, "name": repo.name, "full_name": "Acme/" + repo.name,
			"default_branch": repo.branch, "archived": repo.archived,
		})
		serveRepo(mux, repo)
	}
	mux.HandleFunc("/orgs/Acme/repos", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, listing) })
	mux.HandleFunc("/repos/FacileStudio/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]string{{"name": "v0.9.0"}, {"name": "v0.16.1"}, {"name": "v0.17.0-rc1"}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func serveRepo(mux *http.ServeMux, repo fakeRepo) {
	base := "/repos/Acme/" + repo.name
	if repo.missing {
		return
	}
	mux.HandleFunc(base+"/git/trees/"+repo.branch, func(w http.ResponseWriter, _ *http.Request) {
		var tree []map[string]string
		for name := range repo.files {
			tree = append(tree, map[string]string{"path": name, "type": "blob"})
		}
		writeJSON(w, map[string]any{"tree": tree})
	})
	mux.HandleFunc(base+"/contents/", func(w http.ResponseWriter, r *http.Request) {
		content, ok := repo.files[strings.TrimPrefix(r.URL.Path, base+"/contents/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(content))
	})
	mux.HandleFunc(base+"/actions/runs", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"workflow_runs": []map[string]string{{"conclusion": "success"}}})
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

type fakeSonde struct {
	server    *httptest.Server
	calls     atomic.Int32
	failures  int32
	conflicts int32
	received  []fleet.Report
}

func newFakeSonde(t *testing.T, failures int32) *fakeSonde {
	t.Helper()
	sonde := &fakeSonde{failures: failures}
	sonde.server = httptest.NewServer(http.HandlerFunc(sonde.handle))
	t.Cleanup(sonde.server.Close)
	return sonde
}

func (s *fakeSonde) handle(w http.ResponseWriter, r *http.Request) {
	call := s.calls.Add(1)
	if call <= s.failures {
		http.Error(w, "busy", http.StatusServiceUnavailable)
		return
	}
	if call <= s.failures+s.conflicts {
		http.Error(w, "already stored", http.StatusConflict)
		return
	}
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
	if err := decoder.Decode(&report); err != nil || report.Validate(time.Now()) != nil {
		http.Error(w, "invalid report", http.StatusBadRequest)
		return
	}
	s.received = append(s.received, report)
	writeJSON(w, fleet.IngestResult{RunID: 7, Complete: report.Run.ReposScanned == report.Run.ReposExpected})
}

func fakeDouane(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "douane")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"douane 9.9.9\"; exit 0; fi\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func douaneEmitting(sweep string, exit int) string {
	return "cat <<'JSON'\n" + sweep + "\nJSON\nexit " + string(rune('0'+exit)) + "\n"
}

func testConfig(github *httptest.Server, sonde *fakeSonde, douane string) config {
	cfg := config{org: "Acme", githubAPI: github.URL, githubToken: "gh-token", douaneBin: douane, workers: 2, retryWait: time.Millisecond}
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
