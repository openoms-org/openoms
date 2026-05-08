package handler

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAllegroTokenRefreshContextDetachesCancellationAndPreservesValues(t *testing.T) {
	type ctxKey struct{}
	key := ctxKey{}
	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), key, "request-id"))
	cancelParent()

	refreshCtx, cancel := allegroTokenRefreshContext(parent)
	defer cancel()

	if parent.Err() == nil {
		t.Fatal("expected parent context to be canceled")
	}
	if err := refreshCtx.Err(); err != nil {
		t.Fatalf("expected token refresh context to ignore parent cancellation, got %v", err)
	}
	select {
	case <-refreshCtx.Done():
		t.Fatal("expected token refresh context to remain active until its own timeout")
	default:
	}
	if got := refreshCtx.Value(key); got != "request-id" {
		t.Fatalf("expected token refresh context to preserve request values, got %v", got)
	}
	deadline, ok := refreshCtx.Deadline()
	if !ok {
		t.Fatal("expected token refresh context to have its own timeout")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > allegroTokenRefreshPersistTimeout {
		t.Fatalf("unexpected token refresh context deadline: %v", remaining)
	}
}

func TestAllegroTokenRefreshCallbackDoesNotUseRequestContext(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	filePath := filepath.Join(filepath.Dir(testFile), "allegro_common.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parse allegro_common.go: %v", err)
	}

	var foundCallback bool
	var requestContextCalls []token.Position
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isAllegroOnTokenRefreshCall(call) {
			return true
		}
		foundCallback = true
		if len(call.Args) == 0 {
			return true
		}
		fn, ok := call.Args[0].(*ast.FuncLit)
		if !ok {
			return true
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if isRequestContextCall(n) {
				requestContextCalls = append(requestContextCalls, fset.Position(n.Pos()))
			}
			return true
		})
		return true
	})

	if !foundCallback {
		t.Fatal("expected buildAllegroClient to register an Allegro token refresh callback")
	}
	if len(requestContextCalls) > 0 {
		t.Fatalf("Allegro token refresh callback must not use request context directly; found r.Context() at %v", requestContextCalls)
	}
}

func isAllegroOnTokenRefreshCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WithOnTokenRefresh" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "allegrosdk"
}

func isRequestContextCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Context" {
		return false
	}
	receiver, ok := sel.X.(*ast.Ident)
	return ok && receiver.Name == "r"
}
