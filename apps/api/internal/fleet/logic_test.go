package fleet

import (
	"strconv"
	"testing"
	"time"
)

func TestKeyIgnoresWhichAliasWon(t *testing.T) {
	asCVE := FindingReport{Ecosystem: "npm", Package: "devalue", Installed: "5.8.0", ID: "CVE-2026-1", Aliases: []string{"GHSA-aaaa"}}
	asGHSA := FindingReport{Ecosystem: "npm", Package: "devalue", Installed: "5.8.0", ID: "GHSA-aaaa", Aliases: []string{"CVE-2026-1"}}

	if asCVE.Key() != asGHSA.Key() {
		t.Fatalf("keys differ: %q vs %q", asCVE.Key(), asGHSA.Key())
	}
}

func TestSeverityOutsideTheScaleBecomesUnknown(t *testing.T) {
	for in, want := range map[string]string{"high": "HIGH", " Critical ": "CRITICAL", "moderate": "UNKNOWN", "": "UNKNOWN"} {
		if got := normalizeSeverity(in); got != want {
			t.Errorf("normalizeSeverity(%q) = %q, want %q", in, got, want)
		}
	}
}

func finding(project int64, tier Tier, pkg, fixed, severity string) OpenFinding {
	return OpenFinding{
		ProjectID: project, ProjectName: "p" + strconv.FormatInt(project, 10), Tier: tier, Advisory: "CVE-" + pkg + severity,
		Severity: severity, Package: pkg, Ecosystem: "npm", Installed: "1.0.0", FixedIn: fixed,
	}
}

func TestSeveralFindingsCollapseIntoOneUpgrade(t *testing.T) {
	actions := BuildActions([]OpenFinding{
		finding(1, TierDeployed, "next", "16.2.5", "HIGH"),
		finding(2, TierDeployed, "next", "16.2.5", "MEDIUM"),
		finding(2, TierDeployed, "next", "15.5.16", "MEDIUM"),
	})

	if len(actions) != 2 {
		t.Fatalf("actions = %d, want 2 (one per target version)", len(actions))
	}
	var merged Action
	for _, a := range actions {
		if a.FixedIn == "16.2.5" {
			merged = a
		}
	}
	if merged.Findings != 2 || len(merged.Projects) != 2 || merged.WorstSeverity != "HIGH" {
		t.Fatalf("merged = %+v", merged)
	}
}

func TestLevelFollowsTheRulesInOrder(t *testing.T) {
	kev := finding(1, TierLibrary, "x", "2", "LOW")
	kev.KEV = true
	likely := finding(1, TierClient, "y", "2", "LOW")
	likely.EPSS = 0.35

	cases := []struct {
		name string
		in   OpenFinding
		want Level
	}{
		{"kev anywhere live", kev, LevelAct},
		{"likely exploitation", likely, LevelAct},
		{"critical in deployed", finding(1, TierDeployed, "p", "2", "CRITICAL"), LevelAct},
		{"high in deployed", finding(1, TierDeployed, "q", "2", "HIGH"), LevelSchedule},
		{"critical in a library", finding(1, TierLibrary, "r", "2", "CRITICAL"), LevelSchedule},
		{"high in a library", finding(1, TierLibrary, "s", "2", "HIGH"), LevelTrack},
		{"unclassified high", finding(1, TierUnclassified, "t", "2", "HIGH"), LevelTrack},
	}
	for _, c := range cases {
		got := BuildActions([]OpenFinding{c.in})[0].Level
		if got != c.want {
			t.Errorf("%s: level = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestDormantProjectsNeverAskForAPerson(t *testing.T) {
	archived := finding(1, TierArchive, "p", "2", "CRITICAL")
	archived.KEV = true
	experiment := finding(2, TierExperiment, "p", "2", "CRITICAL")
	experiment.EPSS = 0.9

	if got := BuildActions([]OpenFinding{archived, experiment})[0].Level; got != LevelTrack {
		t.Fatalf("level = %s, want track", got)
	}
}

func TestOneLiveProjectLiftsAGroupThatDormantOnesShare(t *testing.T) {
	dormant := finding(1, TierExperiment, "p", "2", "CRITICAL")
	live := finding(2, TierDeployed, "p", "2", "CRITICAL")

	if got := BuildActions([]OpenFinding{dormant, live})[0].Level; got != LevelAct {
		t.Fatalf("level = %s, want act", got)
	}
}

func TestActionsAreRankedByLevelThenExploitSignals(t *testing.T) {
	low := finding(1, TierDeployed, "low", "2", "HIGH")
	likely := finding(2, TierDeployed, "likely", "2", "HIGH")
	likely.EPSS = 0.5
	crit := finding(3, TierDeployed, "crit", "2", "CRITICAL")

	got := BuildActions([]OpenFinding{low, crit, likely})

	order := []string{got[0].Package, got[1].Package, got[2].Package}
	if order[0] != "likely" || order[1] != "crit" || order[2] != "low" {
		t.Fatalf("order = %v, want [likely crit low]", order)
	}
}

func TestNoFixIsAGroupOfItsOwn(t *testing.T) {
	got := BuildActions([]OpenFinding{
		finding(1, TierDeployed, "p", "", "HIGH"),
		finding(1, TierDeployed, "p", "2", "HIGH"),
	})

	if len(got) != 2 {
		t.Fatalf("groups = %d, want 2", len(got))
	}
}

func TestRankingIsDeterministic(t *testing.T) {
	in := []OpenFinding{
		finding(1, TierDeployed, "m", "2", "HIGH"),
		finding(2, TierDeployed, "n", "2", "HIGH"),
		finding(3, TierDeployed, "o", "2", "HIGH"),
	}
	first := BuildActions(in)
	for range 20 {
		again := BuildActions(in)
		for i := range first {
			if again[i].Package != first[i].Package {
				t.Fatalf("order changed between calls: %v vs %v", again[i].Package, first[i].Package)
			}
		}
	}
}

func TestDriftSeparatesCurrentBehindAndHeld(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{
		{ProjectID: 1, ProjectName: "a", Package: "muse", Resolved: "v0.16.1"},
		{ProjectID: 2, ProjectName: "b", Package: "muse", Resolved: "v0.10.0"},
		{ProjectID: 3, ProjectName: "c", Package: "muse", Resolved: "v0.10.0"},
		{ProjectID: 4, ProjectName: "d", Package: "muse", Resolved: ""},
		{ProjectID: 5, ProjectName: "e", Package: "untracked", Resolved: "v1"},
	}
	holdB := int64(2)
	policies := []Policy{{Package: "muse", ProjectID: &holdB, HeldVersion: "v0.10.0", Reason: "needs the old tokens", ExpiresAt: now.Add(time.Hour)}}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.16.1"}, policies, now)

	if len(rows) != 1 || rows[0].Current != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	behind := rows[0].Behind
	if len(behind) != 2 || behind[0].Resolved != "v0.10.0" || len(behind[0].Projects) != 2 {
		t.Fatalf("behind = %+v", behind)
	}
	if !behind[0].Projects[0].Held || behind[0].Projects[1].Held {
		t.Fatalf("only project b is held: %+v", behind[0].Projects)
	}
	if behind[1].Resolved != unpinned {
		t.Fatalf("an empty pin must read as %q, got %q", unpinned, behind[1].Resolved)
	}
}

func TestAnExpiredHoldNoLongerHides(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{{ProjectID: 2, ProjectName: "b", Package: "muse", Resolved: "v0.10.0"}}
	policies := []Policy{{Package: "muse", HeldVersion: "v0.10.0", Reason: "x", ExpiresAt: now.Add(-time.Second)}}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.16.1"}, policies, now)

	if rows[0].Behind[0].Projects[0].Held {
		t.Fatal("a lapsed hold still hides the project")
	}
}

func TestAHoldOnAnotherVersionDoesNotApply(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{{ProjectID: 2, ProjectName: "b", Package: "muse", Resolved: "v0.9.0"}}
	policies := []Policy{{Package: "muse", HeldVersion: "v0.10.0", Reason: "x", ExpiresAt: now.Add(time.Hour)}}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.16.1"}, policies, now)

	if rows[0].Behind[0].Projects[0].Held {
		t.Fatal("a hold on v0.10.0 hid a project on v0.9.0")
	}
}

func validReport(now time.Time) Report {
	return Report{
		SchemaVersion: SchemaVersion,
		Run:           RunInfo{StartedAt: now.Add(-time.Minute), FinishedAt: now, ReposExpected: 2, ReposScanned: 2},
		Projects: []ProjectReport{
			{GithubRepoID: 1, Name: "a", Covered: true},
			{GithubRepoID: 2, Name: "b", Covered: false, NotCovered: "no lockfile"},
		},
	}
}

func TestValidateAcceptsAnOrdinaryReport(t *testing.T) {
	now := time.Now()
	if err := validReport(now).Validate(now); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateRefusesWhatTheStoreMustNotTouch(t *testing.T) {
	now := time.Now()
	mutations := map[string]func(*Report){
		"wrong schema":            func(r *Report) { r.SchemaVersion = 99 },
		"finished before started": func(r *Report) { r.Run.FinishedAt = r.Run.StartedAt.Add(-time.Second) },
		"finished in the future":  func(r *Report) { r.Run.FinishedAt = now.Add(48 * time.Hour) },
		"scanned above expected":  func(r *Report) { r.Run.ReposScanned = 5 },
		"duplicate repo":          func(r *Report) { r.Projects[1].GithubRepoID = 1 },
		"no repo id":              func(r *Report) { r.Projects[0].GithubRepoID = 0 },
		"uncovered with findings": func(r *Report) { r.Projects[1].Findings = []FindingReport{{ID: "x", Package: "p", Ecosystem: "npm"}} },
		"finding without package": func(r *Report) { r.Projects[0].Findings = []FindingReport{{ID: "x", Ecosystem: "npm"}} },
		"epss out of range": func(r *Report) {
			r.Projects[0].Findings = []FindingReport{{ID: "x", Package: "p", Ecosystem: "npm", EPSS: 4}}
		},
		"empty latest version": func(r *Report) { r.Latest = map[string]string{"muse": ""} },
	}
	for name, mutate := range mutations {
		report := validReport(now)
		mutate(&report)
		if err := report.Validate(now); err == nil {
			t.Errorf("%s: report accepted", name)
		}
	}
}
