package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

func writeTree(root, dir string, files map[string][]byte) error {
	if err := os.MkdirAll(filepath.Join(root, dir, ".git"), 0o755); err != nil {
		return err
	}
	for name, data := range files {
		if isConfig(name) {
			continue
		}
		local := filepath.FromSlash(name)
		if !filepath.IsLocal(local) {
			return fmt.Errorf("refusing to write %q outside the repository", name)
		}
		target := filepath.Join(root, dir, local)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func readState(repo repoInfo, files map[string][]byte, paths []string) fetchedRepo {
	out := fetchedRepo{repo: repo, covered: true}
	out.tree = newRepoTree(paths)
	var workflows [][]byte
	directivePath := ""
	for _, name := range sortedPaths(files) {
		switch {
		case isConfig(name):
			out.tree.files[name] = files[name]
		case classify(name) == kindWorkflow:
			workflows = append(workflows, files[name])
			out.tree.files[name] = files[name]
		case path.Base(name) == "package.json":
			out.libs = append(out.libs, packageJSONPins(name, files[name])...)
		case path.Base(name) == "go.mod":
			pins, directive := goModPins(name, files[name])
			out.libs = append(out.libs, pins...)
			if directive != "" && (directivePath == "" || depth(name) < depth(directivePath)) {
				out.goDirective, directivePath = directive, name
			}
		}
	}
	out.pins = trackedPins(out.libs)
	out.filetWired, out.filetPin = filetState(workflows)
	return out
}

func sortedPaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return paths
}

func depth(name string) int { return strings.Count(name, "/") }

var configNames = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml", ".env.example"}

var configPaths = []string{"filet.yml", "lefthook.yml", "lefthook.yaml", "scripts/check.sh"}

// isConfig names the files read only for edges and conformance. They are
// fetched beside the lockfiles but never handed to douane, and they do not
// count towards the manifest limit.
func isConfig(name string) bool {
	if slices.Contains(configPaths, name) {
		return true
	}
	for _, dir := range skippedDirs {
		if strings.HasPrefix(name, dir) || strings.Contains(name, "/"+dir) {
			return false
		}
	}
	return slices.Contains(configNames, path.Base(name))
}

// repoTree is the part of a repository that edges and checks read: every path
// outside node_modules and vendor, and the bytes of the config and workflow
// files.
type repoTree struct {
	paths []string
	files map[string][]byte
}

func newRepoTree(paths []string) repoTree {
	tree := repoTree{files: map[string][]byte{}}
	for _, p := range paths {
		if !strings.HasPrefix(p, "node_modules/") && !strings.Contains(p, "/node_modules/") &&
			!strings.HasPrefix(p, "vendor/") && !strings.Contains(p, "/vendor/") {
			tree.paths = append(tree.paths, p)
		}
	}
	return tree
}
