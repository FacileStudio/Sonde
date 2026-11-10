package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const maxManifestFiles = 80

type fileKind int

const (
	kindNone fileKind = iota
	kindLockfile
	kindManifest
	kindWorkflow
)

var lockfiles = []string{"go.mod", "bun.lock", "package-lock.json", "Cargo.lock"}

var skippedDirs = []string{"node_modules/", "vendor/", "testdata/", "fixtures/"}

func classify(path string) fileKind {
	for _, dir := range skippedDirs {
		if strings.HasPrefix(path, dir) || strings.Contains(path, "/"+dir) {
			return kindNone
		}
	}
	base := filepath.Base(path)
	switch {
	case slices.Contains(lockfiles, base):
		return kindLockfile
	case base == "package.json":
		return kindManifest
	case strings.HasPrefix(path, ".github/workflows/") && (strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")):
		return kindWorkflow
	}
	return kindNone
}

type fetchedRepo struct {
	repo        repoInfo
	covered     bool
	nothing     bool
	reason      string
	ci          string
	filetWired  bool
	filetPin    string
	goDirective string
	pins        []fleet.PinReport
}

func uncovered(repo repoInfo, reason string) fetchedRepo {
	return fetchedRepo{repo: repo, reason: reason}
}

func nothingToScan(repo repoInfo, reason string) fetchedRepo {
	return fetchedRepo{repo: repo, reason: reason, nothing: true}
}

type selection struct {
	lockfiles int
	paths     []string
}

func selectFiles(paths []string) selection {
	var out selection
	for _, path := range paths {
		switch classify(path) {
		case kindNone:
			continue
		case kindLockfile:
			out.lockfiles++
		}
		out.paths = append(out.paths, path)
	}
	slices.Sort(out.paths)
	return out
}

func (r *runner) collect(ctx context.Context, repo repoInfo) fetchedRepo {
	paths, truncated, err := r.github.paths(ctx, repo)
	var status statusError
	switch {
	case errors.As(err, &status) && status.status == http.StatusConflict:
		return nothingToScan(repo, "empty repository")
	case err != nil:
		return uncovered(repo, "tree failed: "+err.Error())
	case truncated:
		return uncovered(repo, "tree too large to list")
	}
	chosen := selectFiles(paths)
	switch {
	case chosen.lockfiles == 0:
		return nothingToScan(repo, "no lockfile")
	case len(chosen.paths) > maxManifestFiles:
		return uncovered(repo, fmt.Sprintf("too many manifests (%d)", len(chosen.paths)))
	}
	files, err := r.fetchAll(ctx, repo, chosen.paths)
	if err != nil {
		return uncovered(repo, "fetch failed: "+err.Error())
	}
	if err := writeTree(r.root, repo.Name, files); err != nil {
		return uncovered(repo, "write failed: "+err.Error())
	}
	out := readState(repo, files)
	out.ci = r.github.conclusion(ctx, repo)
	return out
}

func (r *runner) fetchAll(ctx context.Context, repo repoInfo, paths []string) (map[string][]byte, error) {
	files := make(map[string][]byte, len(paths))
	for _, path := range paths {
		data, err := r.github.raw(ctx, repo.FullName, repo.DefaultBranch, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		files[path] = data
	}
	return files, nil
}

func writeTree(root, name string, files map[string][]byte) error {
	if err := os.MkdirAll(filepath.Join(root, name, ".git"), 0o755); err != nil {
		return err
	}
	for path, data := range files {
		target := filepath.Join(root, name, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func readState(repo repoInfo, files map[string][]byte) fetchedRepo {
	out := fetchedRepo{repo: repo, covered: true}
	var workflows [][]byte
	directivePath := ""
	for _, path := range sortedPaths(files) {
		switch {
		case classify(path) == kindWorkflow:
			workflows = append(workflows, files[path])
		case filepath.Base(path) == "package.json":
			out.pins = append(out.pins, packageJSONPins(path, files[path])...)
		case filepath.Base(path) == "go.mod":
			pins, directive := goModPins(path, files[path])
			out.pins = append(out.pins, pins...)
			if directive != "" && (directivePath == "" || depth(path) < depth(directivePath)) {
				out.goDirective, directivePath = directive, path
			}
		}
	}
	out.filetWired, out.filetPin = filetState(workflows)
	return out
}

func sortedPaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}

func depth(path string) int { return strings.Count(path, "/") }
