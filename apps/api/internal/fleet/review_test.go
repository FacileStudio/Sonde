package fleet

import (
	"testing"
	"time"
)

func TestBlankAliasesNeverBecomeTheKey(t *testing.T) {
	a := FindingReport{Ecosystem: "npm", Package: "p", Installed: "1", ID: "CVE-2026-1", Aliases: []string{"", " "}}
	b := FindingReport{Ecosystem: "npm", Package: "p", Installed: "1", ID: "CVE-2026-2", Aliases: []string{""}}

	if a.Key() == b.Key() {
		t.Fatalf("two flaws share the key %q", a.Key())
	}
	if got := a.Key(); got != "npm|p|1|CVE-2026-1" {
		t.Fatalf("key = %q", got)
	}
}

func TestNamesAreTrimmed(t *testing.T) {
	f := FindingReport{ID: " GHSA-x ", Aliases: []string{"CVE-1 ", "GHSA-x"}}

	names := f.Names()
	if len(names) != 2 || names[0] != "CVE-1" || names[1] != "GHSA-x" {
		t.Fatalf("names = %q", names)
	}
}

func TestCanonicalNameIsTheSmallestStoredName(t *testing.T) {
	if got := canonicalName("GHSA-x", "CVE-1,,GO-2"); got != "CVE-1" {
		t.Fatalf("canonical = %q", got)
	}
	if got := canonicalName("GHSA-x", ""); got != "GHSA-x" {
		t.Fatalf("canonical without aliases = %q", got)
	}
}

func TestEveryInstalledVersionOfAProjectIsListed(t *testing.T) {
	newer := finding(1, TierDeployed, "p", "2", "HIGH")
	newer.Installed = "1.1.0"

	got := BuildActions([]OpenFinding{finding(1, TierDeployed, "p", "2", "HIGH"), newer})

	if len(got) != 1 || len(got[0].Projects) != 1 || got[0].Projects[0].Installed != "1.0.0, 1.1.0" {
		t.Fatalf("actions = %+v", got)
	}
}

func TestValidateRefusesNULBytes(t *testing.T) {
	now := time.Now()
	mutations := map[string]func(*Report){
		"finding summary": func(r *Report) {
			r.Projects[0].Findings = []FindingReport{{ID: "x", Package: "p", Ecosystem: "npm", Summary: "a\x00b"}}
		},
		"finding alias": func(r *Report) {
			r.Projects[0].Findings = []FindingReport{{ID: "x", Package: "p", Ecosystem: "npm", Aliases: []string{"\x00"}}}
		},
		"pin":     func(r *Report) { r.Projects[0].Pins = []PinReport{{Path: "p", Package: "m", Resolved: "v\x00"}} },
		"project": func(r *Report) { r.Projects[0].DefaultBranch = "ma\x00in" },
		"latest":  func(r *Report) { r.Latest = map[string]string{"muse": "v1\x00"} },
		"run":     func(r *Report) { r.Run.DouaneVersion = "0.6\x00" },
	}
	for name, mutate := range mutations {
		report := validReport(now)
		mutate(&report)
		if err := report.Validate(now); err == nil {
			t.Errorf("%s: a NUL byte was accepted", name)
		}
	}
}
