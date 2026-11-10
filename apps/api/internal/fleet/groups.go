package fleet

import (
	"cmp"
	"slices"
	"strings"
)

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
