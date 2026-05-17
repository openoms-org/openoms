# OPE-297 Public Return Tenant Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure public return creation uses the shared `database.WithTenant` helper instead of a duplicated handler-local tenant transaction helper.

**Architecture:** Keep the public endpoint behavior unchanged and make the smallest safe security fix in the existing handler. The public SECURITY DEFINER lookups for order tenant and return token remain as public lookup paths; the tenant-scoped return insert must go through the canonical `database.WithTenant` boundary.

**Tech Stack:** Go 1.25, chi/v5 handlers, pgx/v5 transactions, OpenOMS `database.WithTenant`, testify.

---

## Files

- Modify: `apps/api-server/internal/handler/public_return_handler.go`
  - Import `internal/database`.
  - Replace `h.withTenant(...)` with `database.WithTenant(...)`.
  - Delete the duplicated `withTenant` method and unused imports.
- Modify: `apps/api-server/internal/handler/public_return_handler_test.go`
  - Add a regression guard that fails if `PublicReturnHandler` reintroduces a local `withTenant` helper or stops calling `database.WithTenant` for create.
- Create: `docs/superpowers/plans/2026-05-17-ope-297-public-return-tenant-context.md`
  - Record the implementation plan and validation checklist for OPE-297.

## Validation

- Targeted RED/GREEN:
  - `cd public/apps/api-server && go test ./internal/handler -run TestPublicReturnHandler_UsesSharedTenantHelper -count=1`
- Package validation:
  - `cd public/apps/api-server && go test ./internal/handler -count=1`
- Diff hygiene:
  - `cd public && git diff --check`
- Before push/PR:
  - `cd public && ./scripts/local-ci.sh`

## Risk And Rollback

- Risk: public return submission is unauthenticated and customer-facing; keep response shapes and status codes unchanged.
- Risk: a purely static regression guard can be brittle, so keep it narrow and tied to the explicit invariant from OPE-297.
- Rollback: revert the PR; no DB, API contract, Helm, or production configuration changes are involved.

## Task 1: Add Regression Guard

**Files:**
- Modify: `apps/api-server/internal/handler/public_return_handler_test.go`

- [ ] **Step 1: Add the failing test**

Add imports:

```go
	_ "embed"
```

Final implementation note: use `go:embed` instead of `os.ReadFile(runtime.Caller(...))` to keep `gosec G304` clean.

```go
//go:embed public_return_handler.go
var publicReturnHandlerSource string
```

Add this test near the other `PublicReturnHandler` tests:

```go
func TestPublicReturnHandler_UsesSharedTenantHelper(t *testing.T) {
	assert.NotContains(t, publicReturnHandlerSource, "func (h *PublicReturnHandler) withTenant")
	assert.Contains(t, publicReturnHandlerSource, "database.WithTenant(r.Context(), h.pool, tenantID")
}
```

- [ ] **Step 2: Run RED**

Run:

```bash
cd public/apps/api-server
go test ./internal/handler -run TestPublicReturnHandler_UsesSharedTenantHelper -count=1
```

Expected: FAIL because `public_return_handler.go` still contains `func (h *PublicReturnHandler) withTenant` and does not call `database.WithTenant(...)`.

## Task 2: Replace Local Tenant Helper

**Files:**
- Modify: `apps/api-server/internal/handler/public_return_handler.go`

- [ ] **Step 1: Use canonical tenant helper**

Add the import:

```go
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
```

Replace:

```go
	err = h.withTenant(r.Context(), tenantID, func(tx pgx.Tx) error {
		return h.returnRepo.Create(r.Context(), tx, ret)
	})
```

with:

```go
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.returnRepo.Create(r.Context(), tx, ret)
	})
```

Delete the `withTenant` method entirely and remove unused imports:

```go
	"context"
	"fmt"
```

- [ ] **Step 2: Run GREEN targeted test**

Run:

```bash
cd public/apps/api-server
go test ./internal/handler -run TestPublicReturnHandler_UsesSharedTenantHelper -count=1
```

Expected: PASS.

## Task 3: Confirm Documentation Scope

**Files:**
- Review: `docs/superpowers/plans/2026-05-17-ope-297-public-return-tenant-context.md`

- [ ] **Step 1: Confirm no tracked contract docs need updates**

This change does not alter API routes, request/response shapes, database schema, Helm, CI/CD, or user-facing behavior. The tracked implementation plan is the only PR documentation update required.

- [ ] **Step 2: PR docs section**

Use:

```md
## Docs updated
- [x] docs/superpowers/plans/2026-05-17-ope-297-public-return-tenant-context.md — implementation and validation plan
- [ ] N/A — no API/DB/user-facing docs changed
```

## Task 4: Self-Review And Validation

**Files:**
- Review all changed files.

- [ ] **Step 1: Run targeted package tests**

Run:

```bash
cd public/apps/api-server
go test ./internal/handler -count=1
```

Expected: PASS.

- [ ] **Step 2: Run diff review**

Run:

```bash
cd public
git diff --check
git diff --stat
git diff
```

Expected: no whitespace errors; diff is limited to OPE-297 plan, handler, handler test, and security context.

- [ ] **Step 3: Run full local CI before push**

Run:

```bash
cd public
./scripts/local-ci.sh
```

Expected: PASS with `/tmp/openoms-local-ci-full-results.txt` showing `STATUS=pass` for the current clean HEAD before push.
