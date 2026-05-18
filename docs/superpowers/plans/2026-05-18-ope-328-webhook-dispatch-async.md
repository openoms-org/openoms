# OPE-328 Webhook Dispatch Async Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure outgoing webhook delivery never blocks service request paths by requiring every domain call-site to execute `WebhookDispatchService.Dispatch` inside `asyncutil.SafeGo`.

**Architecture:** Keep the current `WebhookDispatchService.Dispatch(ctx, tenantID, eventType, payload)` contract and its retry/delivery semantics unchanged. Add a regression test that scans production service/handler code for direct dispatch calls outside `asyncutil.SafeGo`, then fix the remaining synchronous call-site.

**Tech Stack:** Go 1.25, `go/ast`, `go/parser`, `asyncutil.SafeGo`, OpenOMS API service layer.

---

## Scope

This PR is limited to `public/apps/api-server/internal` and does not touch API contracts, database schema, Helm values, or enterprise deployment configuration.

## Files

- Create: `apps/api-server/internal/service/webhook_dispatch_async_usage_test.go`
- Modify: `apps/api-server/internal/service/shipment_service.go`

## Risk And Rollback

- Risk: webhook side effects stay best-effort async, so delivery errors are logged by webhook delivery code instead of being returned to the caller. This matches the existing service-layer pattern for other webhook events.
- Risk: using `context.Background()` detaches webhook delivery from request cancellation. This is already the established OpenOMS pattern for post-commit async side effects such as emails, SMS, invoice hooks, and existing webhook dispatches.
- Rollback: revert the PR; no migration or persistent data changes are involved.

## Task 1: Add Regression Test For Synchronous Webhook Dispatch

**Files:**
- Create: `apps/api-server/internal/service/webhook_dispatch_async_usage_test.go`

- [x] **Step 1: Write the failing test**

Create a test that parses production Go files under `internal/service` plus `internal/handler/order_handler.go`, locates calls to `Dispatch`, and fails when a `WebhookDispatchService` call is not enclosed by an `asyncutil.SafeGo(...)` call.

Core shape:

```go
func TestWebhookDispatchCallSitesUseSafeGo(t *testing.T) {
    // Walk service and selected handler files.
    // Parse each file with go/parser.
    // Find CallExpr selectors named Dispatch.
    // Keep only dispatch calls whose receiver is webhookDispatch, wd, or WebhookDispatch().
    // Walk parent stack upward and require an enclosing asyncutil.SafeGo call.
    // Fail with file:line for every synchronous dispatch.
}
```

- [x] **Step 2: Verify red**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service -run TestWebhookDispatchCallSitesUseSafeGo -count=1
```

Observed: FAIL listed `service/shipment_service.go:690`, because that call used the request context synchronously.

## Task 2: Wrap Remaining Synchronous Shipment Status Dispatch

**Files:**
- Modify: `apps/api-server/internal/service/shipment_service.go`

- [x] **Step 1: Implement minimal fix**

Change the post-commit webhook side effect in `UpdateShipmentStatus` from:

```go
if s.webhookDispatch != nil {
    s.webhookDispatch.Dispatch(ctx, tenantID, "shipment.status_changed", eventData)
}
```

to:

```go
if s.webhookDispatch != nil {
    asyncutil.SafeGo(func() {
        s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.status_changed", eventData)
    })
}
```

- [x] **Step 2: Verify green**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service -run TestWebhookDispatchCallSitesUseSafeGo -count=1
```

Observed: PASS.

## Task 3: Validate Broader Backend Safety

**Files:**
- No additional code changes expected.

- [x] **Step 1: Run service tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service -count=1
```

Observed: PASS.

- [x] **Step 2: Run diff hygiene check**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
```

Observed: no output.

- [x] **Step 3: Run full pre-push validation before pushing**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Observed: PASS, `All 10 checks passed`, 149s total.

## Task 4: Publish PR

**Files:**
- Git metadata only.

- [ ] **Step 1: Commit**

```bash
cd /Users/rafs/praca/openoms-dev/public
git add apps/api-server/internal/service/webhook_dispatch_async_usage_test.go apps/api-server/internal/service/shipment_service.go docs/superpowers/plans/2026-05-18-ope-328-webhook-dispatch-async.md
git commit -m "OPE-328: make webhook dispatch call sites async"
```

- [ ] **Step 2: Push and create PR**

```bash
git push -u origin fix/OPE-328-webhook-dispatch-async
gh pr create --title "OPE-328: make webhook dispatch call sites async" --body-file /tmp/ope-328-pr.md
```

PR body must include:

```md
## Summary
- add a regression test requiring webhook dispatch call-sites to run inside asyncutil.SafeGo
- make shipment status webhook dispatch async after the database commit

## Test plan
- go test ./internal/service -run TestWebhookDispatchCallSitesUseSafeGo -count=1
- go test ./internal/service -count=1
- git diff --check
- ./scripts/local-ci.sh

## Docs updated
- [ ] N/A — no API, DB, workflow, or system documentation changes needed
```

- [ ] **Step 3: Review gate**

Read GitHub checks and CodeRabbit comments. Fix blockers before merge. If a non-blocking follow-up appears, create or update a Linear issue before merging.

## Self-Review

- Spec coverage: OPE-328 requires all webhook dispatches to be async; Task 1 prevents regression and Task 2 fixes the remaining synchronous call-site.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: uses existing `asyncutil.SafeGo`, `context.Background()`, `tenantID`, event name, and payload patterns already present in nearby service code.
