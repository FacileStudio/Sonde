package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

type douaneFinding struct {
	ID        string   `json:"id"`
	Aliases   []string `json:"aliases"`
	Summary   string   `json:"summary"`
	Severity  string   `json:"severity"`
	Package   string   `json:"package"`
	Ecosystem string   `json:"ecosystem"`
	Installed string   `json:"installed"`
	FixedIn   string   `json:"fixed_in"`
	Target    string   `json:"target"`
	Exploit   struct {
		KEV   bool    `json:"kev"`
		EPSS  float64 `json:"epss"`
		Scope string  `json:"scope"`
	} `json:"exploit"`
}

type douaneGap struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`
}

type douaneRepo struct {
	Name     string          `json:"name"`
	Packages int             `json:"packages"`
	Failed   bool            `json:"failed"`
	Findings []douaneFinding `json:"findings"`
	Gaps     []douaneGap     `json:"gaps"`
}

type douaneSweep struct {
	Repos []douaneRepo `json:"repos"`
	Gaps  []douaneGap  `json:"gaps"`
}

const (
	exitFindings   = 1
	exitIncomplete = 3
)

// blockingKinds are the gaps that can hide a finding. A severity or range gap
// only qualifies a finding douane did report, so it leaves the scan whole;
// douane's own `complete` flag counts those too, which is why it is not read.
// An unsupported lockfile blocks only a repository douane read nothing from:
// it was never readable, so no finding from it exists to be wrongly resolved,
// while a repository with nothing readable would otherwise pass as clean.
var blockingKinds = []string{"unreadable", "unsupported", "upstream", "unresolved"}

// rankingFeeds raise upstream gaps that cost the ranking, never a finding.
var rankingFeeds = []string{"kev", "epss"}

func runDouane(ctx context.Context, bin, root string) (douaneSweep, error) {
	command := exec.CommandContext(ctx, bin, "sweep", root, "-format", "json", "-db", "")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	err := command.Run()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit) && (exit.ExitCode() == exitFindings || exit.ExitCode() == exitIncomplete):
	case errors.As(err, &exit):
		return douaneSweep{}, fmt.Errorf("douane sweep exited %d", exit.ExitCode())
	case err != nil:
		return douaneSweep{}, fmt.Errorf("douane sweep: %w", err)
	}
	var sweep douaneSweep
	if err := json.Unmarshal(stdout.Bytes(), &sweep); err != nil {
		return douaneSweep{}, fmt.Errorf("douane output is not the expected json: %w", err)
	}
	return sweep, nil
}

func douaneVersion(ctx context.Context, bin string) string {
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "douane ")
}

func blocking(gaps []douaneGap, packages int) []string {
	var kinds []string
	for _, gap := range gaps {
		if harmless(gap, packages) {
			continue
		}
		kinds = append(kinds, gap.Kind)
	}
	return kinds
}

// harmless reports a gap that cannot hide a finding: a kind outside the
// blocking list, a ranking feed being down, a Dockerfile whose Go module could
// not be named while the module's own go.mod was read, or an unsupported
// lockfile beside packages douane did read.
func harmless(gap douaneGap, packages int) bool {
	switch {
	case !slices.Contains(blockingKinds, gap.Kind):
		return true
	case gap.Kind == "upstream":
		return slices.Contains(rankingFeeds, gap.Subject)
	case gap.Kind == "unresolved":
		return isDockerfile(filepath.Base(gap.Subject))
	case gap.Kind == "unsupported":
		return packages > 0
	}
	return false
}

type verdict struct {
	covered  bool
	reason   string
	findings []fleet.FindingReport
}

// verdicts decides, per repository directory, whether the scan can be trusted.
// A gap at the sweep level comes from the shared OSV resolution and cannot be
// pinned on one repository, so it withholds every repository that has packages.
func (sweep douaneSweep) verdicts() map[string]verdict {
	shared := blocking(sweep.Gaps, 1)
	out := make(map[string]verdict, len(sweep.Repos))
	for _, repo := range sweep.Repos {
		kinds := blocking(repo.Gaps, repo.Packages)
		if repo.Packages > 0 {
			kinds = append(kinds, shared...)
		}
		if repo.Failed {
			kinds = append(kinds, "failed")
		}
		slices.Sort(kinds)
		kinds = slices.Compact(kinds)
		if len(kinds) > 0 {
			out[repo.Name] = verdict{reason: "douane could not finish: " + strings.Join(kinds, ", ")}
			continue
		}
		out[repo.Name] = verdict{covered: true, findings: repo.mapped()}
	}
	return out
}

func (repo douaneRepo) mapped() []fleet.FindingReport {
	findings := make([]fleet.FindingReport, 0, len(repo.Findings))
	for _, f := range repo.Findings {
		findings = append(findings, fleet.FindingReport{
			ID: f.ID, Aliases: f.Aliases, Summary: f.Summary, Severity: f.Severity, Package: f.Package,
			Ecosystem: f.Ecosystem, Installed: f.Installed, FixedIn: f.FixedIn, KEV: f.Exploit.KEV,
			EPSS: f.Exploit.EPSS, Scope: f.Exploit.Scope, Target: f.Target,
		})
	}
	return findings
}
