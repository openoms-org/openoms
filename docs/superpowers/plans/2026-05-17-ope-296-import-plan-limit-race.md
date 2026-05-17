# OPE-296 Import Plan Limit Race Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the monthly order-limit bypass in CSV order imports and fail closed when the limit count cannot be read.

**Architecture:** Move CSV import limit enforcement from the HTTP handler into the same tenant-scoped transaction that performs inserts. Serialize monthly order-limit checks per tenant with a transaction-scoped tenant row lock so concurrent imports cannot both pass the same count. Reuse the same lock/check helper from order creation paths where practical so the existing "atomic limit check" comment becomes true under concurrency. During CSV import, the helper is called before each insert with `pendingCreatesBeforeCurrent=0`, because `CountThisMonth` runs in the same transaction and already sees rows inserted earlier in the import batch.

**Tech Stack:** Go 1.25, pgx/v5, chi handlers, OpenOMS `database.WithTenant`, existing `service.ErrOrderLimitExceeded`.

---

### Task 1: Add Regression Tests For Limit Failure Semantics

**Files:**
- Modify: `apps/api-server/internal/service/import_service_test.go`
- Modify: `apps/api-server/internal/handler/crud_handler_c_test.go`

- [x] **Step 1: Branch and Linear**

Move `OPE-296` and `OPE-316` to In Progress and create branch:

```bash
cd /Users/rafs/praca/openoms-dev/public
git checkout -b fix/OPE-296-OPE-316-import-limit-race
```

- [x] **Step 2: Add service-level tests for the monthly limit helper**

Add tests that exercise the new helper without a real database by using a fake `OrderRepo` and a fake transaction executor:

```go
func TestImportService_EnforceMonthlyOrderLimit_RejectsWhenCurrentPlusImportedHitsLimit(t *testing.T) {
    svc := &ImportService{orderRepo: importLimitOrderRepo{count: 9}}
    err := svc.enforceMonthlyOrderLimit(context.Background(), importLimitTx{}, uuid.New(), 10, 1)
    require.ErrorIs(t, err, ErrOrderLimitExceeded)
}

func TestImportService_EnforceMonthlyOrderLimit_PropagatesCountError(t *testing.T) {
    countErr := errors.New("database timeout")
    svc := &ImportService{orderRepo: importLimitOrderRepo{countErr: countErr}}
    err := svc.enforceMonthlyOrderLimit(context.Background(), importLimitTx{}, uuid.New(), 10, 0)
    require.Error(t, err)
    require.ErrorIs(t, err, countErr)
}
```

- [x] **Step 3: Add handler regression for passing the plan limit into the service**

Refactor the handler to use a small local interface, then add a test proving the handler passes `MaxOrdersMonthly` to the import service and no longer does an out-of-transaction pre-check.

- [x] **Step 4: Run RED tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service -run 'TestImportService_EnforceMonthlyOrderLimit' -count=1
go test ./internal/handler -run 'TestImportHandler_Import_' -count=1
```

Expected: service helper tests fail because the helper does not exist; handler test fails because the handler still calls `CountOrdersThisMonth` before import and the import method has no limit parameter.

### Task 2: Implement Transaction-Scoped Import Limit Enforcement

**Files:**
- Modify: `apps/api-server/internal/service/import_service.go`
- Modify: `apps/api-server/internal/handler/import_handler.go`

- [x] **Step 1: Add an import service option**

Add:

```go
type ImportOrdersOptions struct {
    MaxOrdersMonthly int
}
```

Update `ImportOrders` to accept `opts ImportOrdersOptions`.

- [x] **Step 2: Add the tenant lock and count helper**

Inside `ImportService`, add an unexported helper that:

1. Returns immediately when `maxOrdersMonthly <= 0`.
2. Locks the tenant row inside the current transaction:

```sql
SELECT 1 FROM tenants WHERE id = $1 FOR UPDATE
```

3. Calls `orderRepo.CountThisMonth(ctx, tx)`.
4. Returns `ErrOrderLimitExceeded` when `currentCount + pendingCreatesBeforeCurrent >= maxOrdersMonthly`; `pendingCreatesBeforeCurrent` excludes the create currently being attempted, so a caller can still create the order that reaches the exact limit.
5. Wraps count/lock errors with operation context, preserving `errors.Is`.

- [x] **Step 3: Enforce before each successful row insert**

In the CSV import transaction, call the helper before processing each row that may create an order. Pass `pendingCreatesBeforeCurrent=0` for CSV imports because the count query runs in the same transaction and sees rows already inserted by earlier CSV rows. This rolls back the whole import when the import would exceed the monthly limit, rather than partially importing rows past the plan boundary.

- [x] **Step 4: Remove handler pre-check**

In `ImportHandler.Import`, remove the separate `CountOrdersThisMonth` call. Read `PlanLimitsFromContext`, build `service.ImportOrdersOptions`, pass it to `ImportOrders`, and map `service.ErrOrderLimitExceeded` to the existing 403 response.

### Task 3: Reuse The Lock For Manual Order Creation

**Files:**
- Modify: `apps/api-server/internal/service/order_service.go`
- Modify: `apps/api-server/internal/handler/order_handler.go`

- [x] **Step 1: Add a shared helper in `order_service.go`**

Extract a shared helper in `service` that `OrderService`, `ImportService`, and handler order-duplication code can call:

```go
func EnforceMonthlyOrderLimit(ctx context.Context, tx pgx.Tx, orderCounter MonthlyOrderCounter, tenantID uuid.UUID, maxOrdersMonthly, pendingCreatesBeforeCurrent int) error
```

Use the same tenant row lock and count semantics.

- [x] **Step 2: Replace manual count checks**

Use the helper in:

- `OrderService.Create`
- `OrderHandler.DuplicateOrder`
- `ImportService.ImportOrders`

This keeps all order-creating paths that already enforce plan limits consistent.

### Task 4: Validation

**Files:**
- No additional production files unless gofmt requires formatting.

- [x] **Step 1: Run targeted tests**

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service -run 'TestImportService_EnforceMonthlyOrderLimit|TestOrderService' -count=1
go test ./internal/handler -run 'TestImportHandler_Import|TestOrderHandler' -count=1
```

- [x] **Step 2: Run broader API tests**

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service ./internal/handler -count=1
```

- [x] **Step 3: Run repo checks before commit**

```bash
cd /Users/rafs/praca/openoms-dev/public
gofmt -w apps/api-server/internal/service apps/api-server/internal/handler
git diff --check
```

Expected: formatting and whitespace checks pass before commit.

### Task 5: PR And Deployment

**Files:**
- PR body only.

- [ ] **Step 1: Commit**

Commit title:

```text
OPE-296/OPE-316: enforce import order limits atomically
```

- [ ] **Step 2: Run full local CI on clean HEAD**

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: full local CI passes on the clean committed `HEAD` before push.

- [ ] **Step 3: Push and PR**

Use the already-created branch `fix/OPE-296-OPE-316-import-limit-race`.

PR title:

```text
OPE-296/OPE-316: enforce import order limits atomically
```

PR body must include:

```md
## Docs updated
- [ ] N/A — no public docs changed; behavior remains the same except fail-closed enforcement
```

- [ ] **Step 4: Merge gate**

Before merge:

- all GitHub checks green,
- CodeRabbit comments and review threads read,
- actionable review comments fixed or tracked,
- no AI/GPT/Codex attribution anywhere.

### Risk And Rollback

- Risk: stricter enforcement may reject imports that previously partially succeeded past the plan boundary. This is intended: exceeding the monthly order limit should fail closed.
- Risk: tenant row lock serializes order-creation paths for the same tenant while checking the monthly limit. This is low blast radius because it only wraps order creation/import limit checks and is scoped to one tenant row.
- Rollback: revert the PR; no migration or data rewrite is involved.
- Production deploy: standard public release -> enterprise deploy path only.
