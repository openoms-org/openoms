package observability_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var sensitiveLogFields = map[string]struct{}{
	"address":       {},
	"authorization": {},
	"cookie":        {},
	"credential":    {},
	"credentials":   {},
	"email":         {},
	"nip":           {},
	"password":      {},
	"phone":         {},
	"secret":        {},
	"token":         {},
}

func TestIsSensitiveLogFieldDetectsCompoundKeys(t *testing.T) {
	fset := token.NewFileSet()
	value := &ast.BasicLit{Kind: token.STRING, Value: `"redacted"`}
	for _, key := range []string{
		"customer_email",
		"buyer-phone",
		"billing.address",
		"api.credentials",
		"refresh_token",
	} {
		if !isSensitiveLogField(key, value, fset) {
			t.Fatalf("expected compound log field %q to be sensitive", key)
		}
	}
}

func TestStructuredLogsDoNotUsePIIFieldKeys(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	internalDir := filepath.Clean(filepath.Join(filepath.Dir(testFile), ".."))
	fset := token.NewFileSet()
	var findings []string
	addFinding := func(node ast.Node, key string) {
		pos := fset.Position(node.Pos())
		rel, err := filepath.Rel(internalDir, pos.Filename)
		if err != nil {
			rel = pos.Filename
		}
		findings = append(findings, fmt.Sprintf("%s:%d field %q", rel, pos.Line, key))
	}

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

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isLogCall(call) {
				return true
			}
			for i := 1; i < len(call.Args); i++ {
				if key, value, ok := slogAttrField(call.Args[i]); ok {
					if isSensitiveLogField(key, value, fset) {
						addFinding(call.Args[i], key)
					}
					continue
				}
				if i+1 >= len(call.Args) {
					continue
				}
				key, ok := stringLiteral(call.Args[i])
				if !ok {
					continue
				}
				if isSensitiveLogField(key, call.Args[i+1], fset) {
					addFinding(call.Args[i], key)
				}
				i++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) > 0 {
		t.Fatalf("structured logs must not use sensitive/PII field keys; use IDs or safe hashes instead: %s", strings.Join(findings, ", "))
	}
}

func isSensitiveLogField(key string, value ast.Expr, fset *token.FileSet) bool {
	normalizedKey := strings.ToLower(key)
	if _, sensitive := sensitiveLogFields[normalizedKey]; sensitive {
		return true
	}
	parts := strings.FieldsFunc(normalizedKey, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for _, part := range parts {
		if _, sensitive := sensitiveLogFields[part]; sensitive {
			return true
		}
	}
	if normalizedKey == "to" {
		return looksLikePIIValue(value, fset)
	}
	return false
}

func looksLikePIIValue(value ast.Expr, fset *token.FileSet) bool {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, value); err != nil {
		return false
	}
	expr := strings.ToLower(buf.String())
	return strings.Contains(expr, "email") ||
		strings.Contains(expr, "phone") ||
		strings.Contains(expr, "address") ||
		strings.Contains(expr, "nip")
}

func isLogCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch selector.Sel.Name {
	case "Debug", "Info", "Warn", "Error":
		return true
	default:
		return false
	}
}

func slogAttrField(expr ast.Expr) (string, ast.Expr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return "", nil, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil, false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok || ident.Name != "slog" {
		return "", nil, false
	}

	switch selector.Sel.Name {
	case "Any", "Bool", "Duration", "Float64", "Group", "Int", "Int64", "String", "Time", "Uint64":
	default:
		return "", nil, false
	}

	key, ok := stringLiteral(call.Args[0])
	if !ok {
		return "", nil, false
	}
	value := call.Args[0]
	if len(call.Args) > 1 {
		value = call.Args[1]
	}
	return key, value, true
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}
