package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
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

type douaneSweep struct {
	Repos []struct {
		Name     string          `json:"name"`
		Findings []douaneFinding `json:"findings"`
	} `json:"repos"`
}

const (
	exitFindings   = 1
	exitIncomplete = 3
)

func runDouane(ctx context.Context, bin, root string) (douaneSweep, error) {
	command := exec.CommandContext(ctx, bin, "sweep", root, "-format", "json", "-db", "")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == exitFindings || exit.ExitCode() == exitIncomplete) {
		err = nil
	}
	if err != nil {
		return douaneSweep{}, fmt.Errorf("douane sweep: %w: %s", err, strings.TrimSpace(tail(stderr.String())))
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

func tail(text string) string {
	const keep = 400
	if len(text) <= keep {
		return text
	}
	return text[len(text)-keep:]
}

func (sweep douaneSweep) byRepo() map[string][]fleet.FindingReport {
	out := make(map[string][]fleet.FindingReport, len(sweep.Repos))
	for _, repo := range sweep.Repos {
		findings := make([]fleet.FindingReport, 0, len(repo.Findings))
		for _, f := range repo.Findings {
			findings = append(findings, fleet.FindingReport{
				ID: f.ID, Aliases: f.Aliases, Summary: f.Summary, Severity: f.Severity, Package: f.Package,
				Ecosystem: f.Ecosystem, Installed: f.Installed, FixedIn: f.FixedIn, KEV: f.Exploit.KEV,
				EPSS: f.Exploit.EPSS, Scope: f.Exploit.Scope, Target: f.Target,
			})
		}
		out[repo.Name] = findings
	}
	return out
}
