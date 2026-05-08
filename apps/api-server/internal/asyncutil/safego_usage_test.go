package asyncutil

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionGoroutinesUseSafeGo(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	internalDir := filepath.Clean(filepath.Join(filepath.Dir(testFile), ".."))
	fset := token.NewFileSet()
	var rawGoroutines []string

	err := filepath.WalkDir(internalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Dir(path) == filepath.Join(internalDir, "asyncutil") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if goStmt, ok := n.(*ast.GoStmt); ok {
				pos := fset.Position(goStmt.Go)
				rel, err := filepath.Rel(internalDir, pos.Filename)
				if err != nil {
					rel = pos.Filename
				}
				rawGoroutines = append(rawGoroutines, fmt.Sprintf("%s:%d", rel, pos.Line))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rawGoroutines) > 0 {
		t.Fatalf("production goroutines must use asyncutil.SafeGo; raw go statements found at: %s", strings.Join(rawGoroutines, ", "))
	}
}
