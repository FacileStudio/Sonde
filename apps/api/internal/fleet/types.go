// Package fleet keeps the maintenance picture of every repository: open
// vulnerabilities, version drift, CI state. It stores what a scheduled runner
// reports and ranks it; it runs no scanner and touches no repository.
package fleet

import (
	"slices"
	"strings"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

// SchemaVersion names the shape of Report. The runner and the server ship from
// one repository, but a runner can lag a deploy by a night.
const SchemaVersion = 1

const (
	maxProjects             = 1000
	maxFindingsPerProject   = 20000
	maxPinsPerProject       = 5000
	maxLatestPackages       = 500
	maxFieldLength          = 2000
	maxReportSkewIntoFuture = time.Hour
)

// Report is one night's delivery from the runner.
type Report struct {
	SchemaVersion int               `json:"schema_version"`
	Run           RunInfo           `json:"run"`
	Projects      []ProjectReport   `json:"projects"`
	Latest        map[string]string `json:"latest"`
}

// RunInfo says how complete the delivery is. ReposExpected counts the
// repositories the runner should have been able to scan: one with no lockfile
// has nothing to scan and is left out, while one whose fetch failed is counted,
// so a run that listed 88 and scanned 80 says so. The server calls a run
// complete when the two numbers match.
type RunInfo struct {
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	DouaneVersion string    `json:"douane_version"`
	ReposExpected int       `json:"repos_expected"`
	ReposScanned  int       `json:"repos_scanned"`
}

// ProjectReport is what the runner learned about one repository. Covered false
// means the runner could not scan it, which is different from a clean scan: the
// project's findings are left alone and the reason is shown.
type ProjectReport struct {
	GithubRepoID    int64           `json:"github_repo_id"`
	Name            string          `json:"name"`
	DefaultBranch   string          `json:"default_branch"`
	Covered         bool            `json:"covered"`
	NotCovered      string          `json:"not_covered"`
	GoDirective     string          `json:"go_directive"`
	CIConclusion    string          `json:"ci_conclusion"`
	FiletWired      bool            `json:"filet_wired"`
	FiletPin        string          `json:"filet_pin"`
	DeployedVersion string          `json:"deployed_version"`
	Findings        []FindingReport `json:"findings"`
	Pins            []PinReport     `json:"pins"`
}

// FindingReport is one Douane finding as the runner forwards it.
type FindingReport struct {
	ID        string   `json:"id"`
	Aliases   []string `json:"aliases"`
	Summary   string   `json:"summary"`
	Severity  string   `json:"severity"`
	Package   string   `json:"package"`
	Ecosystem string   `json:"ecosystem"`
	Installed string   `json:"installed"`
	FixedIn   string   `json:"fixed_in"`
	KEV       bool     `json:"kev"`
	EPSS      float64  `json:"epss"`
	Scope     string   `json:"scope"`
	Target    string   `json:"target"`
}

// PinReport is a dependency on a package the suite releases itself.
type PinReport struct {
	Path     string `json:"path"`
	Package  string `json:"package"`
	Spec     string `json:"spec"`
	Resolved string `json:"resolved"`
}

var knownSeverities = []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "UNKNOWN"}

func normalizeSeverity(raw string) string {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	if slices.Contains(knownSeverities, upper) {
		return upper
	}
	return "UNKNOWN"
}

// Names lists every identifier the flaw goes by, sorted and without repeats.
func (f FindingReport) Names() []string {
	names := append([]string{f.ID}, f.Aliases...)
	slices.Sort(names)
	return slices.Compact(names)
}

// Key identifies a finding across nights. It is built from the smallest name the
// flaw has, not from the identifier the scanner reported it under, because that
// choice moves between a CVE and its GHSA alias and a key that followed it would
// read every move as one finding resolved and one new. Deriving the key here
// keeps Sonde correct whichever scanner version delivered the report.
func (f FindingReport) Key() string {
	return f.Ecosystem + "|" + f.Package + "|" + f.Installed + "|" + f.Names()[0]
}

// Validate refuses a report the store should not touch. It reports the first
// problem, naming the project, so the runner's log points at the repository.
func (r Report) Validate(now time.Time) error {
	if r.SchemaVersion != SchemaVersion {
		return errors.Invalid("unsupported schema_version")
	}
	if r.Run.StartedAt.IsZero() || r.Run.FinishedAt.IsZero() || r.Run.FinishedAt.Before(r.Run.StartedAt) {
		return errors.Invalid("run needs started_at and a finished_at that is not earlier")
	}
	if r.Run.FinishedAt.After(now.Add(maxReportSkewIntoFuture)) {
		return errors.Invalid("run finished_at is in the future")
	}
	if r.Run.ReposExpected < 0 || r.Run.ReposScanned < 0 || r.Run.ReposScanned > r.Run.ReposExpected {
		return errors.Invalid("repos_scanned must be between 0 and repos_expected")
	}
	if len(r.Projects) > maxProjects {
		return errors.TooLarge("too many projects")
	}
	if len(r.Latest) > maxLatestPackages {
		return errors.TooLarge("too many latest versions")
	}
	seen := map[int64]bool{}
	for _, project := range r.Projects {
		if err := project.validate(seen); err != nil {
			return err
		}
	}
	for name, version := range r.Latest {
		if name == "" || version == "" || len(name) > maxFieldLength || len(version) > maxFieldLength {
			return errors.Invalid("latest needs a package name and a version")
		}
	}
	return nil
}

func (p ProjectReport) validate(seen map[int64]bool) error {
	if p.GithubRepoID < 1 || strings.TrimSpace(p.Name) == "" || len(p.Name) > maxFieldLength {
		return errors.Invalid("every project needs a github_repo_id and a name")
	}
	if seen[p.GithubRepoID] {
		return errors.Invalid("project " + p.Name + " appears twice")
	}
	seen[p.GithubRepoID] = true
	if len(p.Findings) > maxFindingsPerProject || len(p.Pins) > maxPinsPerProject {
		return errors.TooLarge("project " + p.Name + " reports too much")
	}
	if !p.Covered && (len(p.Findings) > 0 || len(p.Pins) > 0) {
		return errors.Invalid("project " + p.Name + " is not covered but carries findings or pins")
	}
	return p.validateEntries()
}

func (p ProjectReport) validateEntries() error {
	for _, finding := range p.Findings {
		if err := finding.validate(p.Name); err != nil {
			return err
		}
	}
	for _, pin := range p.Pins {
		if pin.Package == "" || pin.Path == "" {
			return errors.Invalid("project " + p.Name + " has a pin without a path or package")
		}
	}
	return nil
}

func (f FindingReport) validate(project string) error {
	if strings.TrimSpace(f.ID) == "" || f.Package == "" || f.Ecosystem == "" {
		return errors.Invalid("project " + project + " has a finding without an id, package or ecosystem")
	}
	if f.EPSS < 0 || f.EPSS > 1 {
		return errors.Invalid("project " + project + " has an epss outside 0 to 1")
	}
	return nil
}
