package service

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

func TestWebhookDispatchCallSitesUseSafeGo(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	serviceDir := filepath.Dir(testFile)
	internalDir := filepath.Clean(filepath.Join(serviceDir, ".."))

	files, err := productionWebhookDispatchFiles(serviceDir, internalDir)
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	var syncDispatches []string
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		var stack []ast.Node
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, n)

			call, ok := n.(*ast.CallExpr)
			if !ok || !isWebhookDispatchCall(call) || isInsideSafeGo(stack) {
				return true
			}

			pos := fset.Position(call.Pos())
			rel, err := filepath.Rel(internalDir, pos.Filename)
			if err != nil {
				rel = pos.Filename
			}
			syncDispatches = append(syncDispatches, fmt.Sprintf("%s:%d", rel, pos.Line))
			return true
		})
	}

	if len(syncDispatches) > 0 {
		t.Fatalf("webhook dispatch call-sites must use asyncutil.SafeGo; synchronous calls found at: %s", strings.Join(syncDispatches, ", "))
	}
}

func productionWebhookDispatchFiles(serviceDir, internalDir string) ([]string, error) {
	files := []string{filepath.Join(internalDir, "handler", "order_handler.go")}
	err := filepath.WalkDir(serviceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func isWebhookDispatchCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Dispatch" {
		return false
	}

	switch receiver := selector.X.(type) {
	case *ast.Ident:
		return receiver.Name == "wd"
	case *ast.SelectorExpr:
		return receiver.Sel.Name == "webhookDispatch"
	default:
		return false
	}
}

func isInsideSafeGo(stack []ast.Node) bool {
	for _, node := range stack[:len(stack)-1] {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			continue
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "SafeGo" {
			continue
		}
		receiver, ok := selector.X.(*ast.Ident)
		if ok && receiver.Name == "asyncutil" {
			return true
		}
	}
	return false
}
