package fleet

import (
	"slices"
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
