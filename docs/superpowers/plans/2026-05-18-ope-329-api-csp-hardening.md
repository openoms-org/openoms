# OPE-329 API CSP Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden API server Content Security Policy so generic API responses, OpenAPI spec, and Swagger UI all have an explicit CSP beyond `frame-ancestors 'none'`.

**Architecture:** Keep the global middleware strict and generic for JSON/API responses. Let documentation handlers set endpoint-specific CSP when they need controlled external Swagger UI assets, so inline/CDN allowances do not leak into normal API responses.

**Tech Stack:** Go 1.25, chi middleware, `httptest`, `testify/assert`, `testify/require`.

---

## Files And Responsibilities

- Modify `apps/api-server/internal/middleware/security_headers.go`: replace minimal CSP with a named strict policy constant used for all responses by default.
- Modify `apps/api-server/internal/middleware/security_headers_test.go`: assert the global CSP contains restrictive default directives and no broad asset allowances.
- Modify `apps/api-server/internal/middleware/security_test.go`: update aggregate security header assertion to match the stricter CSP.
- Modify `apps/api-server/internal/handler/docs_handler.go`: set explicit CSP for `/v1/openapi.yaml` and `/v1/docs`; allow only the Swagger CDN and required inline script/style on the docs page.
- Create `apps/api-server/internal/handler/docs_handler_test.go`: cover CSP headers for OpenAPI spec and Swagger UI.
- Modify `public/.claude/context/SECURITY_POSTURE.md`: record the API CSP hardening decision.

## Task 1: Encode Current CSP Gap In Middleware Tests

**Files:**
- Modify: `apps/api-server/internal/middleware/security_headers_test.go`
- Modify: `apps/api-server/internal/middleware/security_test.go`

- [ ] **Step 1: Replace the exact minimal CSP assertion**

In `apps/api-server/internal/middleware/security_headers_test.go`, replace `TestSecurityHeaders_SetsCSPFrameAncestorsNone` with a test that requires these directives:

```go
func TestSecurityHeaders_SetsStrictDefaultCSP(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "base-uri 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "form-action 'none'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.NotContains(t, csp, "script-src")
	assert.NotContains(t, csp, "style-src")
}
```

- [ ] **Step 2: Update the aggregate middleware security test**

In `apps/api-server/internal/middleware/security_test.go`, replace the exact CSP equality with directive checks:

```go
csp := rr.Header().Get("Content-Security-Policy")
assert.Contains(t, csp, "default-src 'none'")
assert.Contains(t, csp, "frame-ancestors 'none'")
assert.Contains(t, csp, "object-src 'none'")
```

- [ ] **Step 3: Run focused middleware tests and confirm they fail before implementation**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/middleware -run 'TestSecurityHeaders_SetsStrictDefaultCSP|TestSecurity_Headers_AllPresent' -count=1
```

Expected before implementation: FAIL because the current middleware only sets `frame-ancestors 'none'`.

## Task 2: Implement Strict Default API CSP

**Files:**
- Modify: `apps/api-server/internal/middleware/security_headers.go`

- [ ] **Step 1: Add a named default CSP constant**

Add this constant near the top of `security_headers.go`:

```go
const defaultContentSecurityPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'"
```

- [ ] **Step 2: Use the named policy in middleware**

Replace:

```go
h.Set("Content-Security-Policy", "frame-ancestors 'none'")
```

with:

```go
h.Set("Content-Security-Policy", defaultContentSecurityPolicy)
```

- [ ] **Step 3: Run focused middleware tests and confirm they pass**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/middleware -run 'TestSecurityHeaders_SetsStrictDefaultCSP|TestSecurity_Headers_AllPresent' -count=1
```

Expected after implementation: PASS.

## Task 3: Add Documentation Endpoint CSP

**Files:**
- Modify: `apps/api-server/internal/handler/docs_handler.go`
- Create: `apps/api-server/internal/handler/docs_handler_test.go`

- [ ] **Step 1: Add handler-level CSP constants**

Add constants above `swaggerHTML`:

```go
const openAPISpecContentSecurityPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'"

const swaggerUIContentSecurityPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'; img-src 'self' data:; font-src https://unpkg.com data:; style-src https://unpkg.com 'unsafe-inline'; script-src https://unpkg.com 'unsafe-inline'; connect-src 'self'"
```

- [ ] **Step 2: Set CSP in both documentation handlers**

In `ServeSpec`, set:

```go
w.Header().Set("Content-Security-Policy", openAPISpecContentSecurityPolicy)
```

In `ServeSwaggerUI`, set:

```go
w.Header().Set("Content-Security-Policy", swaggerUIContentSecurityPolicy)
```

- [ ] **Step 3: Create handler tests**

Create `apps/api-server/internal/handler/docs_handler_test.go`:

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsHandler_ServeSpecSetsStrictCSP(t *testing.T) {
	handler := NewDocsHandler([]byte("openapi: 3.1.0"))
	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.yaml", nil)
	rr := httptest.NewRecorder()

	handler.ServeSpec(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.NotContains(t, csp, "script-src")
}

func TestDocsHandler_ServeSwaggerUIAllowsOnlyRequiredDocsAssets(t *testing.T) {
	handler := NewDocsHandler([]byte("openapi: 3.1.0"))
	req := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	rr := httptest.NewRecorder()

	handler.ServeSwaggerUI(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "object-src 'none'")
assert.Contains(t, csp, "script-src https://unpkg.com 'unsafe-inline'")
assert.Contains(t, csp, "style-src https://unpkg.com 'unsafe-inline'")
assert.Contains(t, csp, "connect-src 'self'")
assert.NotContains(t, csp, "default-src https:")
assert.NotContains(t, csp, "*")
}
```

- [ ] **Step 4: Run focused handler tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/handler -run 'TestDocsHandler_ServeSpecSetsStrictCSP|TestDocsHandler_ServeSwaggerUIAllowsOnlyRequiredDocsAssets' -count=1
```

Expected: PASS.

## Task 4: Documentation And Validation

**Files:**
- Modify: `public/.claude/context/SECURITY_POSTURE.md`

- [ ] **Step 1: Add a short factual security posture note**

Add a note under the HTTP/API security section:

```markdown
- API server CSP is strict by default (`default-src 'none'`, no script/style sources); Swagger UI overrides CSP only for the documentation endpoint and allows the pinned Swagger UI CDN plus required inline bootstrap.
```

- [ ] **Step 2: Format and run targeted tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
gofmt -w internal/middleware/security_headers.go internal/middleware/security_headers_test.go internal/middleware/security_test.go internal/handler/docs_handler.go internal/handler/docs_handler_test.go
go test ./internal/middleware ./internal/handler -count=1
```

Expected: PASS.

- [ ] **Step 3: Run required pre-push validation**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: PASS before push/PR.

## Risk And Rollback

- Main risk: over-tightening CSP could break Swagger UI. Mitigation: keep the global API policy strict and use a docs-only override that allows only `https://unpkg.com` and required inline bootstrap for current Swagger HTML.
- Rollback: revert the PR. This restores the previous minimal CSP and existing documentation behavior.
- No database, Helm, production config, or public API contract changes are expected.
