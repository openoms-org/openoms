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
	"unicode"
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

func TestIsSensitiveLogFieldDetectsCamelCaseKeys(t *testing.T) {
	fset := token.NewFileSet()
	value := &ast.BasicLit{Kind: token.STRING, Value: `"redacted"`}
	for _, key := range []string{
		"customerEmail",
		"emailAddress",
		"phoneNumber",
		"billingAddress",
		"refreshToken",
		"apiCredential",
		"nipNumber",
	} {
		if !isSensitiveLogField(key, value, fset) {
			t.Fatalf("expected camelCase log field %q to be sensitive", key)
		}
	}
}

func TestIsLogCallDetectsStandardSlogVariants(t *testing.T) {
	for _, expr := range []string{
		`logger.InfoContext(ctx, "message", "tenant_id", tenantID)`,
		`slog.WarnContext(ctx, "message", "tenant_id", tenantID)`,
		`slog.Log(ctx, slog.LevelInfo, "message", "tenant_id", tenantID)`,
		`slog.LogAttrs(ctx, slog.LevelInfo, "message", slog.String("tenant_id", tenantID))`,
		`slog.Group("customer", "tenant_id", tenantID)`,
	} {
		parsed, err := parser.ParseExpr(expr)
		if err != nil {
			t.Fatalf("parse %s: %v", expr, err)
		}
		call, ok := parsed.(*ast.CallExpr)
		if !ok {
			t.Fatalf("expected %s to parse as call expression", expr)
		}
		if !isLogCall(call) {
			t.Fatalf("expected %s to be detected as a log-related call", expr)
		}
	}
}

func TestIsLogCallIgnoresNonSlogGroup(t *testing.T) {
	parsed, err := parser.ParseExpr(`builder.Group("customer", "tenant_id", tenantID)`)
	if err != nil {
		t.Fatal(err)
	}
	call, ok := parsed.(*ast.CallExpr)
	if !ok {
		t.Fatal("expected expression to parse as call expression")
	}
	if isLogCall(call) {
		t.Fatal("expected non-slog Group call to be ignored")
	}
}

func TestSlogAttrFieldRequiresValue(t *testing.T) {
	parsed, err := parser.ParseExpr(`slog.String("tenant_id")`)
	if err != nil {
		t.Fatal(err)
	}
	if key, _, ok := slogAttrField(parsed); ok {
		t.Fatalf("expected malformed slog attr to be ignored, got key %q", key)
	}
}

func TestStructuredLogScanDetectsSlogAttrCompositeFields(t *testing.T) {
	source := `package service

import (
	"context"
	"log/slog"
)

func logCustomerAttr(ctx context.Context, logger *slog.Logger, email string) {
	logger.LogAttrs(ctx, slog.LevelWarn, "customer import failed", slog.Attr{Key: "customerEmail", Value: slog.StringValue(email)})
}`

	findings, err := collectSensitiveLogFindingsFromSource("service/customer_attr.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0], `field "customerEmail"`) {
		t.Fatalf("expected exactly one slog.Attr customerEmail finding, got %v", findings)
	}
}

func TestStructuredLogScanDetectsBackToBackSensitiveStringKeys(t *testing.T) {
	source := `package service

import "log/slog"

func logImportWarning(logger *slog.Logger, userID string, email string) {
	logger.Warn("customer import failed", "user_id", "email", email)
}`

	findings, err := collectSensitiveLogFindingsFromSource("service/import.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0], `field "email"`) {
		t.Fatalf("expected exactly one email field finding, got %v", findings)
	}
}

func TestStructuredLogScanDetectsSensitiveKeysInSlogGroup(t *testing.T) {
	source := `package service

import "log/slog"

func logCustomerGroup(logger *slog.Logger, email string) {
	logger.Info("customer import failed", slog.Group("customer", "email", email))
}`

	findings, err := collectSensitiveLogFindingsFromSource("service/customer.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0], `field "email"`) {
		t.Fatalf("expected exactly one grouped email field finding, got %v", findings)
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
		findings = append(findings, collectSensitiveLogFindingsInFile(file, fset, internalDir)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) > 0 {
		t.Fatalf("structured logs must not use sensitive/PII field keys; use IDs or safe hashes instead: %s", strings.Join(findings, ", "))
	}
}

func collectSensitiveLogFindingsFromSource(filename string, source string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	return collectSensitiveLogFindingsInFile(file, fset, filepath.Dir(filename)), nil
}

func collectSensitiveLogFindingsInFile(file *ast.File, fset *token.FileSet, rootDir string) []string {
	var findings []string
	addFinding := func(node ast.Node, key string) {
		pos := fset.Position(node.Pos())
		rel, err := filepath.Rel(rootDir, pos.Filename)
		if err != nil {
			rel = pos.Filename
		}
		findings = append(findings, fmt.Sprintf("%s:%d field %q", rel, pos.Line, key))
	}

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isLogCall(call) {
			return true
		}
		start := logAttrStartIndex(call)
		if start >= len(call.Args) {
			return false
		}
		inspectLogFields(call.Args[start:], fset, addFinding)
		return false
	})
	return findings
}

func inspectLogFields(args []ast.Expr, fset *token.FileSet, addFinding func(ast.Node, string)) {
	for i := 0; i < len(args); i++ {
		if key, value, ok := slogAttrCompositeField(args[i]); ok {
			if isSensitiveLogField(key, value, fset) {
				addFinding(args[i], key)
			}
			continue
		}
		if key, groupArgs, ok := slogGroupField(args[i]); ok {
			if isSensitiveLogField(key, args[i], fset) {
				addFinding(args[i], key)
			}
			inspectLogFields(groupArgs, fset, addFinding)
			continue
		}
		if key, value, ok := slogAttrField(args[i]); ok {
			if isSensitiveLogField(key, value, fset) {
				addFinding(args[i], key)
			}
			continue
		}
		if i+1 >= len(args) {
			continue
		}
		key, ok := stringLiteral(args[i])
		if !ok {
			continue
		}
		value := args[i+1]
		if isSensitiveLogField(key, value, fset) {
			addFinding(args[i], key)
		}
		if nextKey, nextIsKey := stringLiteral(value); nextIsKey && i+2 < len(args) {
			_, nextValueIsString := stringLiteral(args[i+2])
			if !nextValueIsString && isSensitiveLogField(nextKey, args[i+2], fset) {
				addFinding(value, nextKey)
			}
		}
		i++
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
	for _, token := range sensitiveLogFieldTokens(key) {
		if _, sensitive := sensitiveLogFields[token]; sensitive {
			return true
		}
	}
	for sensitive := range sensitiveLogFields {
		if len(sensitive) >= 4 && strings.Contains(normalizedKey, sensitive) {
			return true
		}
	}
	if normalizedKey == "to" {
		return looksLikePIIValue(value, fset)
	}
	return false
}

func sensitiveLogFieldTokens(key string) []string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(string(current)))
		current = current[:0]
	}

	runes := []rune(key)
	for i, r := range runes {
		if r == '_' || r == '-' || r == '.' || unicode.IsSpace(r) {
			flush()
			continue
		}
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
			flush()
		}
		current = append(current, r)
	}
	flush()
	return tokens
}

func looksLikePIIValue(value ast.Expr, fset *token.FileSet) bool {
	if value == nil {
		return false
	}
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
	return logAttrStartIndex(call) >= 0
}

func logAttrStartIndex(call *ast.CallExpr) int {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return -1
	}
	switch selector.Sel.Name {
	case "Debug", "Info", "Warn", "Error":
		return 1
	case "DebugContext", "InfoContext", "WarnContext", "ErrorContext":
		return 2
	case "Log", "LogAttrs":
		return 3
	case "Group":
		if ident, ok := selector.X.(*ast.Ident); !ok || ident.Name != "slog" {
			return -1
		}
		return 1
	default:
		return -1
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
	case "Any", "Bool", "Duration", "Float64", "Int", "Int64", "String", "Time", "Uint64":
	default:
		return "", nil, false
	}

	if len(call.Args) < 2 {
		return "", nil, false
	}
	key, ok := stringLiteral(call.Args[0])
	if !ok {
		return "", nil, false
	}
	return key, call.Args[1], true
}

func slogAttrCompositeField(expr ast.Expr) (string, ast.Expr, bool) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return "", nil, false
	}
	selector, ok := composite.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Attr" {
		return "", nil, false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok || ident.Name != "slog" {
		return "", nil, false
	}

	var key string
	var value ast.Expr
	for _, element := range composite.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		field, ok := keyValue.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch field.Name {
		case "Key":
			if literal, ok := stringLiteral(keyValue.Value); ok {
				key = literal
			}
		case "Value":
			value = keyValue.Value
		}
	}
	if key == "" {
		return "", nil, false
	}
	return key, value, true
}

func slogGroupField(expr ast.Expr) (string, []ast.Expr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return "", nil, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Group" {
		return "", nil, false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok || ident.Name != "slog" {
		return "", nil, false
	}
	key, ok := stringLiteral(call.Args[0])
	if !ok {
		return "", nil, false
	}
	return key, call.Args[1:], true
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
