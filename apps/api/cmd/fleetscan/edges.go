package main

import (
	"cmp"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

var callAuth = map[string]string{
	"URL":             "url",
	"API_URL":         "url",
	"AUDIENCE":        "audience",
	"API_TOKEN":       "static_key",
	"SERVICE_KEY":     "static_key",
	"INTEGRATION_KEY": "static_key",
	"WEBHOOK_SECRET":  "webhook",
}

var dedicatedEdges = map[string][2]string{
	"ANTENNE_URL":      {fleet.EdgeEvent, "Antenne"},
	"ANTENNE_INSTANCE": {fleet.EdgeEvent, "Antenne"},
	"OIDC_ISSUER":      {fleet.EdgeIdentity, "Registre"},
	"JOURNAL_URL":      {fleet.EdgeLogs, "Journal"},
}

var (
	envAssign     = regexp.MustCompile(`(?m)^[ \t]*#?[ \t]*(?:export[ \t]+)?([A-Z][A-Z0-9_]*)=`)
	composeKey    = regexp.MustCompile(`(?m)^[ \t]*-?[ \t]*([A-Z][A-Z0-9_]*)[ \t]*[:=]`)
	composeInterp = regexp.MustCompile(`\$\{([A-Z][A-Z0-9_]*)`)
	commentLine   = regexp.MustCompile(`(?m)^[ \t]*#.*$`)
)

func suitePackage(pkg string) bool {
	return strings.HasPrefix(pkg, "github.com/FacileStudio/") || strings.HasPrefix(pkg, "@facile/")
}

// trackedPins keeps the pins on packages whose drift Sonde follows. The other
// suite pins still count as library edges.
func trackedPins(pins []fleet.PinReport) []fleet.PinReport {
	var out []fleet.PinReport
	for _, pin := range pins {
		if tracked(pin.Package) {
			out = append(out, pin)
		}
	}
	return out
}

// envNames reads the variable names a repository wires: assignments in an
// .env.example, commented examples included, and the keys and interpolations
// of a compose file outside its comments.
func envNames(tree repoTree) []string {
	var names []string
	for name, data := range tree.files {
		base := path.Base(name)
		switch {
		case base == ".env.example":
			names = append(names, submatches(envAssign, data)...)
		case slices.Contains(configNames, base):
			text := commentLine.ReplaceAll(data, nil)
			names = append(names, submatches(composeKey, text)...)
			names = append(names, submatches(composeInterp, text)...)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func submatches(pattern *regexp.Regexp, data []byte) []string {
	var out []string
	for _, match := range pattern.FindAllSubmatch(data, -1) {
		out = append(out, string(match[1]))
	}
	return out
}

// resolveRepo returns the listed repository whose name matches want ignoring
// case, or want itself when none is listed.
func resolveRepo(known []string, want string) string {
	for _, name := range known {
		if strings.EqualFold(name, want) {
			return name
		}
	}
	return want
}

// envEdge turns one variable name into an edge. A call needs the name to be
// `<APP>_<SUFFIX>` for a listed repository, its dashes read as underscores.
func envEdge(name string, known []string) (fleet.EdgeReport, bool) {
	if dedicated, ok := dedicatedEdges[name]; ok {
		return fleet.EdgeReport{To: resolveRepo(known, dedicated[1]), Kind: dedicated[0], Evidence: name}, true
	}
	for _, repo := range known {
		prefix := strings.ToUpper(strings.ReplaceAll(repo, "-", "_")) + "_"
		rest, found := strings.CutPrefix(name, prefix)
		if auth, ok := callAuth[rest]; found && ok {
			return fleet.EdgeReport{To: repo, Kind: fleet.EdgeCall, Auth: auth, Evidence: name}, true
		}
	}
	return fleet.EdgeReport{}, false
}

// deriveEdges reads who the repository self depends on: a library edge per
// suite pin, then an edge per wired variable that names another repository.
// An edge to self is dropped and duplicates collapse.
func deriveEdges(self string, tree repoTree, libs []fleet.PinReport, known []string) []fleet.EdgeReport {
	var edges []fleet.EdgeReport
	for _, pin := range libs {
		edges = append(edges, fleet.EdgeReport{To: resolveRepo(known, path.Base(pin.Package)), Kind: fleet.EdgeLib, Evidence: pin.Spec})
	}
	for _, name := range envNames(tree) {
		if edge, ok := envEdge(name, known); ok {
			edges = append(edges, edge)
		}
	}
	edges = slices.DeleteFunc(edges, func(e fleet.EdgeReport) bool { return strings.EqualFold(e.To, self) })
	slices.SortFunc(edges, func(a, b fleet.EdgeReport) int {
		return cmp.Or(strings.Compare(a.Kind, b.Kind), strings.Compare(a.To, b.To), strings.Compare(a.Evidence, b.Evidence))
	})
	return slices.Compact(edges)
}
