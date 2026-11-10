package main

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

var fixtureRepos = []string{
	"Agenda", "Antenne", "antenne-client", "Ardoise", "Courrier", "Glouton", "Journal", "muse",
	"Nuage", "Opus", "porte", "Registre", "Sablier", "tronc", "vitrine",
}

var fixtureLatest = map[string]string{
	"github.com/FacileStudio/tronc": "v0.17.1",
	"github.com/FacileStudio/porte": "v0.13.0",
	"@facile/muse":                  "v0.16.1",
}

func loadFixture(t *testing.T, name string) fetchedRepo {
	t.Helper()
	root := filepath.Join("testdata", "repos", name)
	files := map[string][]byte{}
	var paths []string
	err := filepath.WalkDir(root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		rel := filepath.ToSlash(strings.TrimPrefix(p, root+string(filepath.Separator)))
		files[rel], paths = data, append(paths, rel)
		return err
	})
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return readState(repoInfo{Name: name}, files, paths)
}

func TestEdgesAreReadFromPinsAndWiredVariables(t *testing.T) {
	repo := loadFixture(t, "Courrier")

	got := deriveEdges("Courrier", repo.tree, repo.libs, fixtureRepos)

	want := []fleet.EdgeReport{
		{To: "Agenda", Kind: "call", Auth: "webhook", Evidence: "AGENDA_WEBHOOK_SECRET"},
		{To: "Glouton", Kind: "call", Auth: "audience", Evidence: "GLOUTON_AUDIENCE"},
		{To: "Nuage", Kind: "call", Auth: "static_key", Evidence: "NUAGE_SERVICE_KEY"},
		{To: "Opus", Kind: "call", Auth: "static_key", Evidence: "OPUS_INTEGRATION_KEY"},
		{To: "Registre", Kind: "call", Auth: "static_key", Evidence: "REGISTRE_API_TOKEN"},
		{To: "Sablier", Kind: "call", Auth: "url", Evidence: "SABLIER_API_URL"},
		{To: "antenne-client", Kind: "call", Auth: "url", Evidence: "ANTENNE_CLIENT_URL"},
		{To: "Antenne", Kind: "event", Evidence: "ANTENNE_INSTANCE"},
		{To: "Registre", Kind: "identity", Evidence: "OIDC_ISSUER"},
		{To: "antenne-client", Kind: "lib", Evidence: "v0.0.0-20261101000000-abcdef123456"},
		{To: "muse", Kind: "lib", Evidence: "github:FacileStudio/muse#v0.16.1"},
		{To: "porte", Kind: "lib", Evidence: "v0.12.0"},
		{To: "tronc", Kind: "lib", Evidence: "v0.17.1"},
		{To: "Journal", Kind: "logs", Evidence: "JOURNAL_URL"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("edges:\n got  %+v\n want %+v", got, want)
	}
}

func TestAProseMentionOrACommentedComposeLineIsNoEdge(t *testing.T) {
	tree := repoTree{files: map[string][]byte{
		"docker-compose.yml":    []byte("# OPUS_URL: ${OPUS_URL}\nservices: {}\n"),
		"apps/api/.env.example": []byte("# Point OPUS_URL at Opus.\nSELF_URL=x\n"),
	}}

	if got := deriveEdges("Self", tree, nil, []string{"Opus", "Self"}); len(got) != 0 {
		t.Fatalf("edges = %+v, want none", got)
	}
}

func checkMap(checks []fleet.CheckReport) map[string]bool {
	out := map[string]bool{}
	for _, check := range checks {
		out[check.ID] = check.OK
	}
	return out
}

func TestConformanceAsksOnlyTheChecksThatApply(t *testing.T) {
	cases := []struct {
		repo string
		want map[string]bool
	}{
		{"Courrier", map[string]bool{
			"docker.single": true, "fuzz.present": true, "gate.douane": true, "gate.filet": true, "gate.race": true,
			"gate.tests": true, "hooks.lefthook": true, "license.present": true, "migrations.sql": true, "pins.current": false,
		}},
		{"Ardoise", map[string]bool{
			"docker.single": false, "fuzz.present": false, "gate.douane": false, "gate.filet": false, "gate.race": false,
			"gate.tests": true, "hooks.lefthook": false, "license.present": false, "migrations.sql": false, "pins.current": false,
		}},
		{"vitrine", map[string]bool{"gate.douane": false, "gate.filet": false, "gate.tests": false, "hooks.lefthook": false, "license.present": true}},
	}
	for _, c := range cases {
		repo := loadFixture(t, c.repo)
		got := conformance(repo.tree, repo.pins, fixtureLatest)
		if !maps.Equal(checkMap(got), c.want) {
			t.Errorf("%s checks = %+v, want %v", c.repo, got, c.want)
		}
	}
}

func TestAStalePinNamesWhatIsBehind(t *testing.T) {
	repo := loadFixture(t, "Courrier")

	check, ok := pinsCurrent(repo.pins, fixtureLatest)

	want := "behind: github.com/FacileStudio/porte v0.12.0 in apps/api/go.mod, latest v0.13.0"
	if !ok || check.OK || check.Detail != want {
		t.Fatalf("pins.current = %+v, want %q", check, want)
	}
}

func TestAScanReportsEdgesAndChecksForCoveredReposOnly(t *testing.T) {
	quietLogs(t)
	sonde, _ := scanWith(t, treeChecks+douaneEmitting(goodSweep, 1), 0)
	report := onlyReport(t, sonde)

	api := projectNamed(t, report, "api")
	if !slices.Contains(api.Edges, fleet.EdgeReport{To: "Journal", Kind: "logs", Evidence: "JOURNAL_URL"}) ||
		!slices.Contains(api.Edges, fleet.EdgeReport{To: "tronc", Kind: "lib", Evidence: "v0.16.0"}) {
		t.Fatalf("api edges = %+v", api.Edges)
	}
	if !checkMap(api.Conformance)["gate.filet"] || len(api.Conformance) == 0 {
		t.Fatalf("api checks = %+v", api.Conformance)
	}
	for _, name := range []string{"yarny", "broken", "old", "docs"} {
		if got := projectNamed(t, report, name); len(got.Edges)+len(got.Conformance) != 0 {
			t.Fatalf("%s carries edges or checks without a covering scan: %+v", name, got)
		}
	}
}
