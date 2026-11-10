package env

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const apiRoot = "../.."

var variableName = regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`)

// readsEnv reports whether call reads the environment: os.Getenv, os.LookupEnv,
// or one of the *OrDefault helpers that wrap them.
func readsEnv(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name == "Getenv" || fun.Sel.Name == "LookupEnv"
	case *ast.Ident:
		return strings.HasSuffix(fun.Name, "OrDefault")
	}
	return false
}

// stringConsts maps every string constant declared in files to its value.
func stringConsts(files []*ast.File) map[string]string {
	consts := map[string]string{}
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if spec, ok := node.(*ast.ValueSpec); ok {
				addStringConsts(spec, consts)
			}
			return true
		})
	}
	return consts
}

func addStringConsts(spec *ast.ValueSpec, consts map[string]string) {
	for i, name := range spec.Names {
		if i >= len(spec.Values) {
			return
		}
		if lit, ok := spec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			consts[name.Name], _ = strconv.Unquote(lit.Value)
		}
	}
}

// argumentName resolves the first argument of an environment read to a
// variable name, through a string literal or a constant of the same package.
func argumentName(call *ast.CallExpr, consts map[string]string) string {
	if len(call.Args) == 0 {
		return ""
	}
	switch arg := call.Args[0].(type) {
	case *ast.BasicLit:
		value, _ := strconv.Unquote(arg.Value)
		return value
	case *ast.Ident:
		return consts[arg.Name]
	}
	return ""
}

// packageFiles groups the server's non-test Go files by directory, leaving out
// cmd/, whose programs run outside the container.
func packageFiles(t *testing.T) map[string][]*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	packages := map[string][]*ast.File{}
	err := filepath.WalkDir(apiRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "cmd" {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], file)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", apiRoot, err)
	}
	return packages
}

func collectReads(file *ast.File, consts map[string]string, names map[string]bool) {
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !readsEnv(call) {
			return true
		}
		if name := argumentName(call, consts); variableName.MatchString(name) {
			names[name] = true
		}
		return true
	})
}

// serverVariables lists every environment variable the server reads by name.
func serverVariables(t *testing.T) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, files := range packageFiles(t) {
		consts := stringConsts(files)
		for _, file := range files {
			collectReads(file, consts, names)
		}
	}
	return names
}

// libraryVariables are read inside a dependency, where the source scan cannot
// see them: antenne.ConfigFromEnv reads the Antenne connection.
var libraryVariables = []string{"ANTENNE_URL", "ANTENNE_INSTANCE", "ANTENNE_APP", "ANTENNE_SECRET", "ANTENNE_INSTANCE_ID"}

func TestComposeForwardsEveryVariableTheServerReads(t *testing.T) {
	compose, err := os.ReadFile("../../../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	names := serverVariables(t)
	for _, name := range libraryVariables {
		names[name] = true
	}
	if len(names) < 10 || !names["PORTE_TOKEN_KEY"] {
		t.Fatalf("found %d variables, PORTE_TOKEN_KEY among them: %v; the scan no longer sees how the server reads them", len(names), names["PORTE_TOKEN_KEY"])
	}
	for name := range names {
		forwarded := regexp.MustCompile(`(?m)^\s+` + name + `:`)
		if !forwarded.Match(compose) {
			t.Errorf("the server reads %s but docker-compose.yml does not forward it, so production never sees it", name)
		}
	}
}
