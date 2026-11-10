package fleet

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Level says how soon a group of findings needs a person.
type Level string

const (
	LevelAct      Level = "act"
	LevelSchedule Level = "schedule"
	LevelTrack    Level = "track"
)

// Tier says what a project is for, which decides how much its findings matter.
type Tier string

const (
	TierUnclassified Tier = "unclassified"
	TierDeployed     Tier = "deployed"
	TierLibrary      Tier = "library"
	TierCLI          Tier = "cli"
	TierClient       Tier = "client"
	TierExperiment   Tier = "experiment"
	TierArchive      Tier = "archive"
)

var tiers = []Tier{TierUnclassified, TierDeployed, TierLibrary, TierCLI, TierClient, TierExperiment, TierArchive}

// ValidTier reports whether name is a tier a project can hold.
func ValidTier(name string) bool {
	return slices.Contains(tiers, Tier(name))
}

const exploitLikely = 0.1

func severityRank(severity string) int {
	switch severity {
	case "CRITICAL":
		return 4
	case "HIGH":
		return 3
	case "MEDIUM":
		return 2
	case "LOW":
		return 1
	}
	return 0
}

// OpenFinding is a finding that has not been resolved, with the project it sits in.
type OpenFinding struct {
	ProjectID   int64
	ProjectName string
	Tier        Tier
	Advisory    string
	Severity    string
	Package     string
	Ecosystem   string
	Installed   string
	FixedIn     string
	KEV         bool
	EPSS        float64
	FirstSeen   time.Time
}

// ActionProject is one project an action touches.
type ActionProject struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Tier      Tier   `json:"tier"`
	Installed string `json:"installed"`
}

// Action is one decision: take this package to this version. Several advisories
// in several projects usually collapse into it, which is why the list is short.
type Action struct {
	Level         Level           `json:"level"`
	Ecosystem     string          `json:"ecosystem"`
	Package       string          `json:"package"`
	FixedIn       string          `json:"fixed_in"`
	Findings      int             `json:"findings"`
	WorstSeverity string          `json:"worst_severity"`
	KEV           bool            `json:"kev"`
	MaxEPSS       float64         `json:"max_epss"`
	FirstSeen     time.Time       `json:"first_seen"`
	Advisories    []string        `json:"advisories"`
	Projects      []ActionProject `json:"projects"`
}

func dormant(tier Tier) bool {
	return tier == TierArchive || tier == TierExperiment
}

// classify decides the level of one group. The order of the rules is the
// policy: known exploitation first, then whether anything deployed is touched,
// then plain severity. A group that only touches archived or experimental
// projects never asks for a person, whatever its score.
func classify(action Action) Level {
	live, deployed := false, false
	for _, project := range action.Projects {
		if !dormant(project.Tier) {
			live = true
		}
		if project.Tier == TierDeployed {
			deployed = true
		}
	}
	worst := severityRank(action.WorstSeverity)
	switch {
	case !live:
		return LevelTrack
	case action.KEV || action.MaxEPSS >= exploitLikely:
		return LevelAct
	case deployed && worst >= severityRank("CRITICAL"):
		return LevelAct
	case deployed && worst >= severityRank("HIGH"):
		return LevelSchedule
	case worst >= severityRank("CRITICAL"):
		return LevelSchedule
	}
	return LevelTrack
}

func levelRank(level Level) int {
	switch level {
	case LevelAct:
		return 0
	case LevelSchedule:
		return 1
	}
	return 2
}

// BuildActions groups open findings by the upgrade that clears them and ranks
// the groups. Findings with no fix are grouped apart, because "no fix exists" is
// a different decision from "a fix is waiting".
func BuildActions(findings []OpenFinding) []Action {
	groups := map[string]*group{}
	for _, f := range findings {
		id := f.Ecosystem + "|" + f.Package + "|" + f.FixedIn
		if groups[id] == nil {
			groups[id] = newGroup(f)
		}
		groups[id].add(f)
	}
	out := make([]Action, 0, len(groups))
	for _, g := range groups {
		out = append(out, g.finish())
	}
	slices.SortFunc(out, compareActions)
	return out
}

type group struct {
	action     Action
	advisories map[string]bool
	projects   map[int64]int
	versions   map[int64]map[string]bool
}

func newGroup(f OpenFinding) *group {
	return &group{
		action:     Action{Ecosystem: f.Ecosystem, Package: f.Package, FixedIn: f.FixedIn, FirstSeen: f.FirstSeen},
		advisories: map[string]bool{},
		projects:   map[int64]int{},
		versions:   map[int64]map[string]bool{},
	}
}

func (g *group) add(f OpenFinding) {
	g.action.Findings++
	g.advisories[f.Advisory] = true
	if severityRank(f.Severity) > severityRank(g.action.WorstSeverity) {
		g.action.WorstSeverity = f.Severity
	}
	g.action.KEV = g.action.KEV || f.KEV
	g.action.MaxEPSS = max(g.action.MaxEPSS, f.EPSS)
	if f.FirstSeen.Before(g.action.FirstSeen) {
		g.action.FirstSeen = f.FirstSeen
	}
	if _, seen := g.projects[f.ProjectID]; !seen {
		g.projects[f.ProjectID] = len(g.action.Projects)
		g.versions[f.ProjectID] = map[string]bool{}
		g.action.Projects = append(g.action.Projects, ActionProject{ID: f.ProjectID, Name: f.ProjectName, Tier: f.Tier})
	}
	g.versions[f.ProjectID][f.Installed] = true
}

func (g *group) finish() Action {
	for id, index := range g.projects {
		g.action.Projects[index].Installed = strings.Join(sortedKeys(g.versions[id]), ", ")
	}
	g.action.Advisories = sortedKeys(g.advisories)
	slices.SortFunc(g.action.Projects, func(a, b ActionProject) int { return cmp.Compare(a.Name, b.Name) })
	g.action.Level = classify(g.action)
	return g.action
}

func compareActions(a, b Action) int {
	if c := cmp.Compare(levelRank(a.Level), levelRank(b.Level)); c != 0 {
		return c
	}
	if a.KEV != b.KEV {
		if a.KEV {
			return -1
		}
		return 1
	}
	if c := cmp.Compare(b.MaxEPSS, a.MaxEPSS); c != 0 {
		return c
	}
	if c := cmp.Compare(severityRank(b.WorstSeverity), severityRank(a.WorstSeverity)); c != 0 {
		return c
	}
	if c := cmp.Compare(len(b.Projects), len(a.Projects)); c != 0 {
		return c
	}
	return strings.Compare(a.Ecosystem+"|"+a.Package+"|"+a.FixedIn, b.Ecosystem+"|"+b.Package+"|"+b.FixedIn)
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}
