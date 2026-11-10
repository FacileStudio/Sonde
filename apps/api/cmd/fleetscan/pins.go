package main

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

var trackedByRepo = map[string]string{
	"@facile/muse":                  "FacileStudio/muse",
	"github.com/FacileStudio/tronc": "FacileStudio/tronc",
	"github.com/FacileStudio/porte": "FacileStudio/porte",
	"github.com/FacileStudio/filet": "FacileStudio/filet",
}

func tracked(pkg string) bool {
	_, ok := trackedByRepo[pkg]
	return ok
}

// packageJSONPins reads tracked dependencies from a package.json. Resolved is
// the git ref after the '#' with any `semver:` prefix removed: a tag, a commit,
// a branch name or a range, compared literally against the latest tag, so
// anything but the exact tag reads as drift. An empty Resolved means unpinned.
func packageJSONPins(path string, data []byte) []fleet.PinReport {
	var manifest struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	seen := map[string]bool{}
	var pins []fleet.PinReport
	for _, group := range []map[string]string{manifest.Dependencies, manifest.DevDependencies, manifest.PeerDependencies, manifest.OptionalDependencies} {
		for name, spec := range group {
			if tracked(name) && !seen[name] {
				seen[name] = true
				pins = append(pins, fleet.PinReport{Path: path, Package: name, Spec: spec, Resolved: refOf(spec)})
			}
		}
	}
	slices.SortFunc(pins, func(a, b fleet.PinReport) int { return strings.Compare(a.Package, b.Package) })
	return pins
}

func refOf(spec string) string {
	_, ref, found := strings.Cut(spec, "#")
	if !found {
		return ""
	}
	return strings.TrimPrefix(ref, "semver:")
}

var (
	replaceBlock = regexp.MustCompile(`(?ms)^replace\s*\(.*?^\)`)
	replaceLine  = regexp.MustCompile(`(?m)^replace\s.*$`)
	requireLine  = regexp.MustCompile(`(?m)^\s*(?:require\s+)?(github\.com/FacileStudio/[A-Za-z0-9_.-]+)(?:/v[0-9]+)?\s+(v[^\s/]+)`)
	goLine       = regexp.MustCompile(`(?m)^go\s+(\S+)`)
)

// goModPins reads tracked modules from a go.mod. A /vN suffix belongs to the
// same tracked repository; replace directives are ignored, since their right
// side names a local path or a fork rather than a release.
func goModPins(path string, data []byte) ([]fleet.PinReport, string) {
	text := replaceLine.ReplaceAllString(replaceBlock.ReplaceAllString(string(data), ""), "")
	var pins []fleet.PinReport
	for _, match := range requireLine.FindAllStringSubmatch(text, -1) {
		if tracked(match[1]) {
			pins = append(pins, fleet.PinReport{Path: path, Package: match[1], Spec: match[2], Resolved: match[2]})
		}
	}
	directive := ""
	if match := goLine.FindStringSubmatch(text); match != nil {
		directive = match[1]
	}
	return pins, directive
}

var (
	filetUse = regexp.MustCompile(`FacileStudio/filet@([^\s'"]+)`)
	filetRun = regexp.MustCompile(`(?m)^\s*(?:-\s*)?run:.*\bfilet\s+(?:check|test)\b`)
)

// filetState reports whether a workflow runs Filet, through its action or a
// `filet check` / `filet test` step, and the action version it pins.
func filetState(workflows [][]byte) (bool, string) {
	wired, pin := false, ""
	for _, workflow := range workflows {
		match := filetUse.FindSubmatch(workflow)
		if match != nil || filetRun.Match(workflow) {
			wired = true
		}
		if match != nil && pin == "" {
			pin = string(match[1])
		}
	}
	return wired, pin
}

type semver [3]int

func parseSemver(tag string) (semver, bool) {
	trimmed := strings.TrimPrefix(tag, "v")
	if strings.ContainsAny(trimmed, "-+") {
		return semver{}, false
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var out semver
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return semver{}, false
		}
		out[i] = n
	}
	return out, true
}

func highestTag(tags []string) string {
	best, bestTag := semver{}, ""
	for _, tag := range tags {
		parsed, ok := parseSemver(tag)
		if ok && (bestTag == "" || slices.Compare(parsed[:], best[:]) > 0) {
			best, bestTag = parsed, tag
		}
	}
	return bestTag
}
