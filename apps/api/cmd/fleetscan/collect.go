package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
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
	kindBuild
)

// lockfiles mirrors douane's inventory.Lockfiles: the ones it parses and the
// ones it only recognises. A file douane cannot read must still reach it, or
// it cannot raise the gap that keeps the repository from reading as clean.
var lockfiles = []string{
	"go.mod", "package-lock.json", "bun.lock", "Cargo.lock",
	"pnpm-lock.yaml", "yarn.lock", "requirements.txt", "poetry.lock", "Pipfile.lock", "uv.lock",
	"Gemfile.lock", "composer.lock", "packages.lock.json", "pubspec.lock", "go.sum",
}

var skippedDirs = []string{"node_modules/", "vendor/", "testdata/", "fixtures/"}

func isDockerfile(base string) bool {
	return base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") || strings.HasSuffix(base, ".Dockerfile")
}

func classify(name string) fileKind {
	if !filepath.IsLocal(filepath.FromSlash(name)) {
		return kindNone
	}
	for _, dir := range skippedDirs {
		if strings.HasPrefix(name, dir) || strings.Contains(name, "/"+dir) {
			return kindNone
		}
	}
	base := path.Base(name)
	switch {
	case slices.Contains(lockfiles, base) || strings.HasSuffix(base, ".csproj"):
		return kindLockfile
	case base == "package.json":
		return kindManifest
	case strings.HasPrefix(name, ".github/workflows/") && (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")):
		return kindWorkflow
	case isDockerfile(base):
		return kindBuild
	}
	return kindNone
}

type fetchedRepo struct {
	repo    repoInfo
	dir     string
	covered bool
	nothing bool
	reason  string
	ci      string
	repoState
}

// repoState is what the runner read from a repository's files: the state it
// reports as is, and the tree that edges and conformance checks derive from.
type repoState struct {
	filetWired  bool
	filetPin    string
	goDirective string
	pins        []fleet.PinReport
	libs        []fleet.PinReport
	tree        repoTree
}

func uncovered(repo repoInfo, reason string) fetchedRepo {
	return fetchedRepo{repo: repo, reason: reason}
}

func nothingToScan(repo repoInfo) fetchedRepo {
	return fetchedRepo{repo: repo, covered: true, nothing: true}
}

type selection struct {
	lockfiles int
	paths     []string
	config    []string
}

func selectFiles(paths []string) selection {
	present := map[string]bool{}
	for _, p := range paths {
		present[p] = true
	}
	var out selection
	for _, p := range paths {
		if isConfig(p) {
			out.config = append(out.config, p)
			continue
		}
		kind := classify(p)
		if kind == kindNone || (path.Base(p) == "go.sum" && present[path.Join(path.Dir(p), "go.mod")]) {
			continue
		}
		if kind == kindLockfile {
			out.lockfiles++
		}
		out.paths = append(out.paths, p)
	}
	slices.Sort(out.paths)
	slices.Sort(out.config)
	return out
}

func (r *runner) collect(ctx context.Context, repo repoInfo) fetchedRepo {
	paths, truncated, err := r.github.paths(ctx, repo)
	var status statusError
	switch {
	case errors.As(err, &status) && status.status == http.StatusConflict:
		return nothingToScan(repo)
	case err != nil:
		return uncovered(repo, "tree failed: "+err.Error())
	case truncated:
		return uncovered(repo, "tree too large to list")
	}
	chosen := selectFiles(paths)
	switch {
	case chosen.lockfiles == 0:
		return nothingToScan(repo)
	case len(chosen.paths) > maxManifestFiles:
		return uncovered(repo, fmt.Sprintf("too many manifests (%d)", len(chosen.paths)))
	}
	files, err := r.fetchAll(ctx, repo, slices.Concat(chosen.paths, chosen.config))
	if err != nil {
		return uncovered(repo, "fetch failed: "+err.Error())
	}
	dir := strconv.FormatInt(repo.ID, 10)
	if err := writeTree(r.root, dir, files); err != nil {
		return uncovered(repo, "write failed: "+err.Error())
	}
	out := readState(repo, files, paths)
	out.dir = dir
	out.ci = r.github.conclusion(ctx, repo)
	return out
}

func (r *runner) fetchAll(ctx context.Context, repo repoInfo, paths []string) (map[string][]byte, error) {
	files := make(map[string][]byte, len(paths))
	for _, p := range paths {
		data, err := r.github.raw(ctx, repo.FullName, repo.DefaultBranch, p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		files[p] = data
	}
	return files, nil
}
