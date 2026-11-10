package main

import (
	"cmp"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const maxDetailItems = 5

var (
	filetGate    = regexp.MustCompile(`FacileStudio/filet@|\bfilet\s+(?:check|test)\b`)
	douaneGate   = regexp.MustCompile(`FacileStudio/douane@|\bdouane\s+(?:scan|sweep)\b`)
	testsGate    = regexp.MustCompile(`\bgo test\b|\bbun test\b|\bcargo test\b|\bvitest\b`)
	raceGate     = regexp.MustCompile(`(?m)\bgo test\b.*\s-race\b`)
	sqlMigration = regexp.MustCompile(`(?:^|/)migrations/[^/]+\.sql$`)
	fuzzDir      = regexp.MustCompile(`(?:^|/)testdata/fuzz/|^fuzz/|/fuzz/`)
)

func (t repoTree) has(match func(string) bool) bool { return slices.ContainsFunc(t.paths, match) }

// gateSources are the files a gate can run from: the workflows, lefthook and
// scripts/check.sh, in path order.
func (t repoTree) gateSources() []string {
	var out []string
	for name := range t.files {
		if classify(name) == kindWorkflow || (slices.Contains(configPaths, name) && name != "filet.yml") {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

func (t repoTree) gate(id string, pattern *regexp.Regexp) fleet.CheckReport {
	var found []string
	for _, name := range t.gateSources() {
		if pattern.Match(t.files[name]) {
			found = append(found, name)
		}
	}
	if len(found) == 0 {
		return fleet.CheckReport{ID: id, Detail: "not found in workflows, lefthook.yml or scripts/check.sh"}
	}
	return fleet.CheckReport{ID: id, OK: true, Detail: "found in " + listed(found)}
}

func listed(items []string) string {
	if len(items) > maxDetailItems {
		return strings.Join(items[:maxDetailItems], ", ") + fmt.Sprintf(" and %d more", len(items)-maxDetailItems)
	}
	return strings.Join(items, ", ")
}

func present(id string, ok bool, yes, no string) fleet.CheckReport {
	if ok {
		return fleet.CheckReport{ID: id, OK: true, Detail: yes}
	}
	return fleet.CheckReport{ID: id, Detail: no}
}

// conformance asks a repository the suite's standards. A check that does not
// apply to it is left out, never failed: migrations only of a repository with
// apps/api/go.mod, the single Dockerfile only of one with apps/api, the race
// detector and fuzzing only of one with Go or Rust code, and current pins only
// of one with a tracked pin whose latest tag is known.
func conformance(tree repoTree, pins []fleet.PinReport, latest map[string]string) []fleet.CheckReport {
	hasGo := tree.has(func(p string) bool { return path.Base(p) == "go.mod" })
	hasRust := tree.has(func(p string) bool { return path.Base(p) == "Cargo.toml" })
	checks := []fleet.CheckReport{
		tree.gate("gate.filet", filetGate),
		tree.gate("gate.douane", douaneGate),
		tree.gate("gate.tests", testsGate),
		present("hooks.lefthook", tree.has(func(p string) bool { return p == "lefthook.yml" || p == "lefthook.yaml" }),
			"lefthook.yml at the root", "no lefthook.yml at the root"),
	}
	if hasGo {
		checks = append(checks, tree.gate("gate.race", raceGate))
	}
	if hasGo || hasRust {
		checks = append(checks, present("fuzz.present", tree.has(fuzzDir.MatchString),
			"a fuzz corpus or fuzz directory is committed", "no testdata/fuzz or fuzz directory"))
	}
	checks = append(checks, layoutChecks(tree)...)
	if check, ok := pinsCurrent(pins, latest); ok {
		checks = append(checks, check)
	}
	slices.SortFunc(checks, func(a, b fleet.CheckReport) int { return strings.Compare(a.ID, b.ID) })
	return checks
}

func layoutChecks(tree repoTree) []fleet.CheckReport {
	var checks []fleet.CheckReport
	if tree.has(func(p string) bool { return p == "apps/api/go.mod" }) {
		checks = append(checks, present("migrations.sql", tree.has(sqlMigration.MatchString),
			"ordered SQL migrations are committed", "no migrations/*.sql"))
	}
	if !tree.has(func(p string) bool { return strings.HasPrefix(p, "apps/api/") }) {
		return checks
	}
	var dockerfiles []string
	for _, p := range tree.paths {
		if isDockerfile(path.Base(p)) {
			dockerfiles = append(dockerfiles, p)
		}
	}
	single := len(dockerfiles) == 1 && dockerfiles[0] == "Dockerfile"
	return append(checks, present("docker.single", single, "one Dockerfile, at the root",
		fmt.Sprintf("%d Dockerfiles: %s", len(dockerfiles), listed(dockerfiles))))
}

func pinsCurrent(pins []fleet.PinReport, latest map[string]string) (fleet.CheckReport, bool) {
	var behind []string
	asked := 0
	for _, pin := range pins {
		want, known := latest[pin.Package]
		if !known {
			continue
		}
		asked++
		if pin.Resolved != want {
			behind = append(behind, fmt.Sprintf("%s %s in %s, latest %s", pin.Package, cmp.Or(pin.Resolved, "unpinned"), pin.Path, want))
		}
	}
	if asked == 0 {
		return fleet.CheckReport{}, false
	}
	return present("pins.current", len(behind) == 0, fmt.Sprintf("all %d suite pins on the latest tag", asked), "behind: "+listed(behind)), true
}
