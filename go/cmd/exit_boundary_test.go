package cmd

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type fixtureExitError struct {
	code int
}

func (e fixtureExitError) Error() string { return "fixture exit" }
func (e fixtureExitError) ExitCode() int { return e.code }

func TestExitCodeUsesTypedErrors(t *testing.T) {
	wrapped := errors.Join(errors.New("context"), fixtureExitError{code: 255})
	if got := exitCode(wrapped); got != 255 {
		t.Fatalf("exitCode() = %d, want 255", got)
	}
}

func TestOSExitStaysAtRootExecutionBoundary(t *testing.T) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test filename")
	}
	goRoot := filepath.Dir(filepath.Dir(filename))
	allowed := filepath.Join(goRoot, "cmd", "root.go")
	fileset := token.NewFileSet()

	var violations []string
	err := filepath.WalkDir(goRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || path == allowed {
			return nil
		}
		file, err := parser.ParseFile(fileset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Exit" {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}
			position := fileset.Position(call.Pos())
			relative, err := filepath.Rel(goRoot, position.Filename)
			if err != nil {
				relative = position.Filename
			}
			violations = append(violations, relative+":"+strconv.Itoa(position.Line))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("os.Exit is only allowed in cmd/root.go; return errors below it:\n%s", strings.Join(violations, "\n"))
	}
}
