package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const fleetFinding = `{"id":"CVE-2026-1","aliases":["GHSA-aaaa"],"summary":"boom","severity":"HIGH","package":"next","ecosystem":"npm","installed":"16.2.4","fixed_in":"16.2.5","target":"apps/client/bun.lock","exploit":{"kev":false,"epss":0.355,"scope":"prod"}}`

const goodSweep = `{"repos":[
 {"name":"11","packages":5,"findings":[` + fleetFinding + `],"gaps":[{"kind":"range","subject":"next"}]},
 {"name":"15","packages":0,"findings":[],"gaps":[{"kind":"unsupported","subject":"yarn.lock"}]},
 {"name":"16","packages":1,"findings":[],"gaps":[]},
 {"name":"17","packages":1,"findings":[],"gaps":[]}
],"gaps":[{"kind":"severity","subject":"GO-1"},{"kind":"upstream","subject":"epss"}]}`

const treeChecks = `[ -d "$2/11/.git" ] && [ -f "$2/11/go.mod" ] && [ -f "$2/11/apps/client/bun.lock" ] || exit 2
[ -e "$2/11/node_modules" ] && exit 2
[ -e "$2/11/docker-compose.yml" ] && exit 2
[ -f "$2/15/yarn.lock" ] && [ -f "$2/16/go.mod" ] && [ -f "$2/17/go.mod" ] || exit 2
`

func scanWith(t *testing.T, douaneBody string, failures int32) (*fakeSonde, error) {
	t.Helper()
	github, sonde := fakeGithub(t, fleetRepos()), newFakeSonde(t, failures)
	err := scan(context.Background(), testConfig(github, sonde, fakeDouane(t, douaneBody)))
	return sonde, err
}

func onlyReport(t *testing.T, sonde *fakeSonde) fleet.Report {
	t.Helper()
	if len(sonde.received) != 1 {
		t.Fatalf("reports delivered = %d, want 1", len(sonde.received))
	}
	return sonde.received[0]
}

func TestAScanDeliversAReportTheServerAccepts(t *testing.T) {
	logs := quietLogs(t)
	sonde, err := scanWith(t, treeChecks+douaneEmitting(goodSweep, 1), 0)
	if !errors.Is(err, errIncomplete) || !strings.Contains(err.Error(), "scanned 4 of 6") {
		t.Fatalf("scan: %v: an incomplete run is delivered, then fails the job", err)
	}
	report := onlyReport(t, sonde)

	checkRunAndAPI(t, report)
	checkEveryOtherRepo(t, report)
	if strings.Contains(logs.String(), "CVE-") || strings.Contains(logs.String(), "next") {
		t.Fatalf("the log carries finding detail:\n%s", logs.String())
	}
}

func checkRunAndAPI(t *testing.T, report fleet.Report) {
	t.Helper()
	if report.Run.DouaneVersion != "9.9.9" || report.Run.ReposExpected != 6 || report.Run.ReposScanned != 4 {
		t.Fatalf("run = %+v: six live repos, four covered", report.Run)
	}
	api := projectNamed(t, report, "api")
	if !api.Covered || len(api.Findings) != 1 || api.CIConclusion != "success" || !api.FiletWired || api.FiletPin != "v0.3.0" || api.GoDirective != "1.26" || len(api.Pins) != 3 {
		t.Fatalf("api = %+v", api)
	}
	if report.Latest["@facile/muse"] != "v0.16.1" {
		t.Fatalf("latest = %v", report.Latest)
	}
}

func checkEveryOtherRepo(t *testing.T, report fleet.Report) {
	t.Helper()
	if got := projectNamed(t, report, "docs"); !got.Covered || got.NotCovered != "" || len(got.Findings) != 0 {
		t.Fatalf("docs = %+v: nothing to scan is a clean scan", got)
	}
	if got := projectNamed(t, report, "yarny"); got.Covered || !strings.Contains(got.NotCovered, "unsupported") {
		t.Fatalf("yarny = %+v: a lockfile douane cannot read must not read as clean", got)
	}
	for _, name := range []string{".github", "slashy"} {
		if got := projectNamed(t, report, name); !got.Covered {
			t.Fatalf("%s = %+v", name, got)
		}
	}
	if got := projectNamed(t, report, "broken"); got.Covered || !strings.HasPrefix(got.NotCovered, "tree failed") {
		t.Fatalf("broken = %+v", got)
	}
	if got := projectNamed(t, report, "old"); got.Covered || got.NotCovered != archivedReason || len(got.Findings) != 0 {
		t.Fatalf("old = %+v: an archived repository is reported, not covered and counted nowhere", got)
	}
}

func TestARepoDouaneFailedOrCouldNotReadIsNotCovered(t *testing.T) {
	quietLogs(t)
	sweep := `{"repos":[
 {"name":"11","packages":5,"failed":true,"findings":[` + fleetFinding + `],"gaps":[]},
 {"name":"16","packages":1,"findings":[],"gaps":[{"kind":"unreadable","subject":"go.mod"}]}
],"gaps":[]}`
	sonde, err := scanWith(t, douaneEmitting(sweep, 3), 0)
	if err != nil && !errors.Is(err, errIncomplete) {
		t.Fatalf("scan: %v", err)
	}
	report := onlyReport(t, sonde)

	if got := projectNamed(t, report, "api"); got.Covered || len(got.Findings) != 0 || !strings.Contains(got.NotCovered, "failed") {
		t.Fatalf("api = %+v: a failed repo must keep its previous findings", got)
	}
	if got := projectNamed(t, report, ".github"); got.Covered || !strings.Contains(got.NotCovered, "unreadable") {
		t.Fatalf(".github = %+v", got)
	}
}

func TestAnOSVFailureWithholdsEveryRepoWithPackages(t *testing.T) {
	quietLogs(t)
	sweep := strings.Replace(goodSweep, `{"kind":"upstream","subject":"epss"}`, `{"kind":"upstream","subject":"next, vite"}`, 1)
	sonde, err := scanWith(t, douaneEmitting(sweep, 3), 0)
	if err != nil && !errors.Is(err, errIncomplete) {
		t.Fatalf("scan: %v", err)
	}
	report := onlyReport(t, sonde)

	for _, name := range []string{"api", ".github", "slashy"} {
		if got := projectNamed(t, report, name); got.Covered || !strings.Contains(got.NotCovered, "upstream") {
			t.Fatalf("%s = %+v: an OSV gap cannot be pinned on one repo, so none is clean", name, got)
		}
	}
	if got := projectNamed(t, report, "docs"); !got.Covered {
		t.Fatalf("docs = %+v: a repo with nothing to scan is unaffected", got)
	}
}

func TestAFailingScannerDeliversNothing(t *testing.T) {
	quietLogs(t)
	sonde, err := scanWith(t, douaneEmitting(`{"repos":[{"name":"11","packages":1,"findings":[],"gaps":[]}],"gaps":[]}`, 2), 0)
	if err == nil {
		t.Fatal("a scanner that crashed was treated as a clean scan")
	}
	if strings.Contains(err.Error(), "11") || len(sonde.received) != 0 {
		t.Fatalf("err = %v, delivered = %d", err, len(sonde.received))
	}
}

func TestARepoTheScannerSkippedIsNotReportedAsClean(t *testing.T) {
	quietLogs(t)
	sonde, err := scanWith(t, douaneEmitting(`{"repos":[],"gaps":[]}`, 0), 0)
	if err != nil && !errors.Is(err, errIncomplete) {
		t.Fatalf("scan: %v", err)
	}
	if api := projectNamed(t, onlyReport(t, sonde), "api"); api.Covered || !strings.Contains(api.NotCovered, "douane returned nothing") {
		t.Fatalf("api = %+v", api)
	}
}

func TestAnInvalidProjectIsDemotedInsteadOfLosingTheNight(t *testing.T) {
	quietLogs(t)
	sweep := strings.Replace(goodSweep, `"package":"next"`, `"package":""`, 1)
	sonde, err := scanWith(t, douaneEmitting(sweep, 1), 0)
	if err != nil && !errors.Is(err, errIncomplete) {
		t.Fatalf("scan: %v", err)
	}
	report := onlyReport(t, sonde)

	if api := projectNamed(t, report, "api"); api.Covered || api.NotCovered != invalidScan || len(api.Findings) != 0 || len(api.Pins) != 0 {
		t.Fatalf("api = %+v", api)
	}
	if got := projectNamed(t, report, ".github"); !got.Covered {
		t.Fatalf(".github = %+v: one bad project must not cost the others", got)
	}
}

func TestDeliveryRetriesServerErrorsButNotRefusals(t *testing.T) {
	quietLogs(t)
	sonde, err := scanWith(t, douaneEmitting(goodSweep, 1), 2)
	if !errors.Is(err, errIncomplete) || len(sonde.received) != 1 || sonde.calls.Load() != 3 {
		t.Fatalf("err = %v, delivered = %d, calls = %d: two 503s then a success", err, len(sonde.received), sonde.calls.Load())
	}

	github, refusing := fakeGithub(t, fleetRepos()), newFakeSonde(t, 0)
	cfg := testConfig(github, refusing, fakeDouane(t, douaneEmitting(goodSweep, 1)))
	cfg.fleetToken = "wrong-token-wrong-token-wrong-token-1"
	if err := scan(context.Background(), cfg); err == nil || refusing.calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d: a 401 is final", err, refusing.calls.Load())
	}
}

func TestOutWritesTheReportInsteadOfDeliveringIt(t *testing.T) {
	quietLogs(t)
	github, sonde := fakeGithub(t, fleetRepos()), newFakeSonde(t, 0)
	cfg := testConfig(github, sonde, fakeDouane(t, douaneEmitting(goodSweep, 1)))
	cfg.out = filepath.Join(t.TempDir(), "report.json")

	if err := scan(context.Background(), cfg); err != nil {
		t.Fatalf("scan: %v", err)
	}
	var written fleet.Report
	data, err := os.ReadFile(cfg.out)
	if err != nil || json.Unmarshal(data, &written) != nil || len(written.Projects) != 7 || sonde.calls.Load() != 0 {
		t.Fatalf("written report wrong (err %v) or delivered anyway (%d calls)", err, sonde.calls.Load())
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
