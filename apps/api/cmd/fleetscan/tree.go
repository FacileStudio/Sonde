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

func readState(repo repoInfo, files map[string][]byte) fetchedRepo {
	out := fetchedRepo{repo: repo, covered: true}
	var workflows [][]byte
	directivePath := ""
	for _, name := range sortedPaths(files) {
		switch {
		case classify(name) == kindWorkflow:
			workflows = append(workflows, files[name])
		case path.Base(name) == "package.json":
			out.pins = append(out.pins, packageJSONPins(name, files[name])...)
		case path.Base(name) == "go.mod":
			pins, directive := goModPins(name, files[name])
			out.pins = append(out.pins, pins...)
			if directive != "" && (directivePath == "" || depth(name) < depth(directivePath)) {
				out.goDirective, directivePath = directive, name
			}
		}
	}
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
