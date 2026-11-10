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
	repo, ok := trackedByRepo[pkg]
	return ok && repo != ""
}

func packageJSONPins(path string, data []byte) []fleet.PinReport {
	var manifest struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	var pins []fleet.PinReport
	for _, group := range []map[string]string{manifest.Dependencies, manifest.DevDependencies} {
		for name, spec := range group {
			if tracked(name) {
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
	return ref
}

var (
	requireLine = regexp.MustCompile(`(?m)^\s*(?:require\s+)?(github\.com/FacileStudio/[A-Za-z0-9_.-]+)\s+(v[^\s/]+)`)
	goLine      = regexp.MustCompile(`(?m)^go\s+(\S+)`)
)

func goModPins(path string, data []byte) ([]fleet.PinReport, string) {
	var pins []fleet.PinReport
	for _, match := range requireLine.FindAllStringSubmatch(string(data), -1) {
		if tracked(match[1]) {
			pins = append(pins, fleet.PinReport{Path: path, Package: match[1], Spec: match[2], Resolved: match[2]})
		}
	}
	directive := ""
	if match := goLine.FindStringSubmatch(string(data)); match != nil {
		directive = match[1]
	}
	return pins, directive
}

var filetUse = regexp.MustCompile(`FacileStudio/filet@([^\s'"]+)`)

func filetState(workflows [][]byte) (bool, string) {
	wired, pin := false, ""
	for _, workflow := range workflows {
		if strings.Contains(strings.ToLower(string(workflow)), "filet") {
			wired = true
		}
		if match := filetUse.FindSubmatch(workflow); match != nil && pin == "" {
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
