package fleet

import (
	"cmp"
	"slices"
	"strings"
)

var nativePrefixes = []string{"GO-", "RUSTSEC-", "PYSEC-", "OSV-", "MAL-"}

// nameRank orders the names a flaw goes by: a CVE first, then a GitHub
// advisory, then the ecosystem's own database, then anything else. A mirror's
// alias such as a Bitnami BIT- record can be added to a record at any time and
// would otherwise win on spelling alone.
func nameRank(name string) int {
	switch {
	case strings.HasPrefix(name, "CVE-"):
		return 0
	case strings.HasPrefix(name, "GHSA-"):
		return 1
	case slices.ContainsFunc(nativePrefixes, func(prefix string) bool { return strings.HasPrefix(name, prefix) }):
		return 2
	}
	return 3
}

// primaryName picks the highest-ranked name, and the smallest of that rank.
func primaryName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return slices.MinFunc(names, func(a, b string) int {
		if c := cmp.Compare(nameRank(a), nameRank(b)); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
}

// Names lists every identifier the flaw goes by, trimmed, sorted and without
// repeats. Blank aliases are dropped, so an empty string can never become the
// key of every advisory that carried one.
func (f FindingReport) Names() []string {
	return cleanNames(append([]string{f.ID}, f.Aliases...))
}

func cleanNames(raw []string) []string {
	names := make([]string, 0, len(raw))
	for _, name := range raw {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// canonicalName is the name a stored finding is shown under, by the same rule
// Key uses.
func canonicalName(advisory, aliases string) string {
	if name := primaryName(cleanNames(append(strings.Split(aliases, ","), advisory))); name != "" {
		return name
	}
	return advisory
}

// Key identifies a finding across nights. It is built from the flaw's primary
// name, not from the identifier the scanner reported it under, because that
// choice moves between a CVE and its GHSA alias. A finding already stored keeps
// whatever key it was filed under: knownKeys matches any of its names first.
func (f FindingReport) Key() string {
	return f.Ecosystem + "|" + f.Package + "|" + f.Installed + "|" + primaryName(f.Names())
}
