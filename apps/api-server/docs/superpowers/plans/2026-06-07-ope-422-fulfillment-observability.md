# OPE-422 Fulfillment Observability Implementation Plan

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax for tracking. Implemented inline by the author subagent task-by-task with TDD.

**Goal:** Make the Provider Studio + Fulfillment Orchestration paths observable in production via Prometheus metrics, correlation-ID + redaction in worker logs, and audit-log entries for fulfillment state changes — all ADDITIVE and best-effort (never breaks/slows the primary operation).

**Architecture:** A new dependency-free `internal/metrics` package holds a `FulfillmentMetrics` collector built from the SAME hand-rolled atomic-counter + Prometheus-text-exposition pattern as the existing `middleware.MetricsCollector` (this codebase has NO prometheus library dependency — do NOT add one). The collector uses ONLY bounded, low-cardinality enum labels (operation, status, category, event_type, result, state). `middleware.MetricsCollector` gains an optional list of extra `PromCollector` renderers so the existing `/metrics` handler emits the fulfillment metrics too. The fulfillment/validation/registry services and the orchestration worker get a nil-safe `*metrics.FulfillmentMetrics` injected; every record call is wrapped so a metrics error can never break the operation. Fulfillment state-change audit entries are written via the existing `repository.AuditRepository.Log` (tenant audit) inside the operator-action paths.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, sync/atomic, testify. No new dependencies.

---

## File Structure

- Create `internal/metrics/fulfillment_metrics.go` — `FulfillmentMetrics` collector (bounded-label atomics + Prom text rendering + `PromCollector` interface).
- Create `internal/metrics/fulfillment_metrics_test.go` — unit tests for recording + bounded-label assertion.
- Modify `internal/middleware/metrics.go` — add optional `extra []PromCollector` to `MetricsCollector`, render them in `Handler()`.
- Modify `internal/middleware/metrics_test.go` — test extra collector rendering.
- Modify `internal/worker/orchestration_worker.go` — inject metrics, record claimed/processed/failed + queue depth, add correlation-id log fields.
- Modify `internal/repository/orchestration_repository.go` — add `CountPending` for the queue-depth gauge.
- Modify `internal/service/fulfillment_service.go` — inject metrics, record provider attempts (operation+status) + blockers (category).
- Modify `internal/service/fulfillment_unit_service.go` — record unit/step transitions + supplier blockers (category).
- Modify `internal/service/fulfillment_read_service.go` — record stuck/blocked process gauge from OperationsSummary; audit operator actions (resolve blocker / retry step).
- Modify `internal/service/provider_validation_service.go` — record validation runs + failures.
- Modify `internal/service/provider_registry_service.go` — record publication transitions.
- Modify `cmd/server/main.go` — construct `FulfillmentMetrics` early, inject into services/worker, register with `MetricsCollector`.

## Cardinality discipline (CRITICAL — documented in code)

Metric labels are ONLY bounded enums:
- `operation` ∈ {create_shipment, generate_label, download_label, sync_tracking}
- `status` ∈ {pending, succeeded, failed}
- `category` ∈ {integration, supplier, operator, capability, mapping}
- `event_type` ∈ {order.created, fulfillment.step}
- `result` ∈ {claimed, processed, failed} (outbox) / {passed, failed, error, skipped} (validation)
- `state` ∈ provider publication states (research…retired) — bounded
- `bucket` ∈ {ready, processing, stuck, blocked, provider_issue, missing_data}

NEVER tenant_id / order_id / process_id / unit_id / user_id / any UUID as a label. Any value passed as a label is validated against the bounded set and dropped to `"other"` if unknown, so a bug cannot explode cardinality.

---

### Task 1: FulfillmentMetrics collector

**Files:**
- Create: `internal/metrics/fulfillment_metrics.go`
- Test: `internal/metrics/fulfillment_metrics_test.go`

- [x] **Step 1: Write failing tests** asserting: counters increment; unknown label coerced to `other`; gauge set; `Render` output contains the metric names + `# TYPE` lines; an assertion that no rendered line contains an id-like label key (`tenant_id`,`order_id`,`process_id`,`unit_id`,`user_id`).
- [x] **Step 2:** Run `go test ./internal/metrics/...` → FAIL (package missing).
- [x] **Step 3:** Implement collector: `PromCollector` interface (`Render(*strings.Builder)`); `FulfillmentMetrics` with atomic counter maps keyed by joined bounded labels + atomic gauges; constructor `NewFulfillmentMetrics`; record methods (`RecordProviderAttempt(op,status)`, `RecordBlocker(category)`, `RecordOutboxEvent(result)`, `SetOutboxQueueDepth(n)`, `SetStuckProcesses(n)`, `SetBlockedProcesses(n)`, `RecordValidationRun(verdict)`, `RecordValidationFailure()`, `RecordPublicationTransition(toState)`, `RecordUnitTransition(status)`, `RecordStepTransition(status)`); bounded-set coercion helper; `Render`. All record methods are nil-receiver-safe.
- [x] **Step 4:** Run `go test ./internal/metrics/...` → PASS.
- [x] **Step 5:** Commit.

### Task 2: MetricsCollector extra renderers

**Files:**
- Modify: `internal/middleware/metrics.go`
- Test: `internal/middleware/metrics_test.go`

- [x] **Step 1:** Test that `MetricsCollector` with a registered extra collector renders the extra collector's output in `Handler()`.
- [x] **Step 2:** Run → FAIL.
- [x] **Step 3:** Add `extra []PromCollector` field + `Register(PromCollector)` (or constructor variadic) where `PromCollector` is a tiny local interface `{ Render(*strings.Builder) }` (structurally satisfied by metrics.FulfillmentMetrics — no import of metrics pkg → no cycle); append their output at the end of `Handler()`.
- [x] **Step 4:** Run → PASS.
- [x] **Step 5:** Commit.

### Task 3: Orchestration repo CountPending

**Files:**
- Modify: `internal/repository/orchestration_repository.go`

- [x] **Step 1:** Add `CountPending(ctx, q) (int, error)` → `SELECT COUNT(*) FROM orchestration_outbox WHERE status='pending'` (runs on privileged pool, cross-tenant — same as ClaimDue). (No new test file: covered by existing repo test conventions / build + vet; queue-depth wiring tested at worker level.)
- [x] **Step 2:** `go build ./...` → OK.
- [x] **Step 3:** Commit.

### Task 4: Orchestration worker metrics + correlation logging

**Files:**
- Modify: `internal/worker/orchestration_worker.go`
- Test: `internal/worker/orchestration_worker_test.go` (create if absent — pure metrics behavior on the record calls)

- [x] **Step 1:** Tests: a fake dispatcher + a metrics collector assert that processing increments `processed` on success and `failed` on permanent failure; correlation-id log field present (capture slog). Use nil-safe metrics in constructor so existing callers unaffected; add `WithMetrics(*FulfillmentMetrics)` style setter OR new constructor param. Prefer a setter to avoid touching the existing 21-worker registration signature widely — but the worker is constructed once in main, so adding a constructor param is acceptable; use a setter to keep the diff minimal and nil-safe.
- [x] **Step 2:** Run → FAIL.
- [x] **Step 3:** Implement: add `metrics *metrics.FulfillmentMetrics` field + `WithMetrics` setter; in `Run`, after `ClaimDue`, record `claimed` per event and set queue-depth gauge via `repo.CountPending`; in `processEvent` record `processed`/`failed`; add `correlation_id` (event idempotency key) + `event_type` + `process_id` as structured log fields (process_id is a LOG field, allowed — NOT a metric label). All metrics calls nil-safe + best-effort (no error propagation).
- [x] **Step 4:** Run → PASS.
- [x] **Step 5:** Commit.

### Task 5: FulfillmentService metrics (attempts + blockers)

**Files:**
- Modify: `internal/service/fulfillment_service.go`
- Modify: `internal/service/fulfillment_unit_service.go`
- Test: `internal/service/fulfillment_metrics_test.go` (new — pure: call a tiny exported recording shim with a collector)

- [x] **Step 1:** Tests asserting RecordProviderAttempt increments `(operation,status)`; CreateCarrierBlocker/CreateSupplierBlocker increment `(category)`; RecordUnitTransition/RecordStep increment transition counters. Because these methods need a DB pool, test the metric emission through the nil-safe collector hooks directly (extract emission into best-effort helper methods on the service that take already-known bounded values, callable without a DB).
- [x] **Step 2:** Run → FAIL.
- [x] **Step 3:** Add `metrics *metrics.FulfillmentMetrics` field + `WithMetrics` setter on `FulfillmentService`; call nil-safe metric records at the success points of RecordProviderAttempt, CreateCarrierBlocker, CreateSupplierBlocker, RecordUnitTransition, RecordStep. Category derived via `model.BlockerCategory(code)`.
- [x] **Step 4:** Run → PASS.
- [x] **Step 5:** Commit.

### Task 6: FulfillmentReadService — stuck/blocked gauge + operator-action audit

**Files:**
- Modify: `internal/service/fulfillment_read_service.go`
- Test: `internal/service/fulfillment_read_metrics_test.go` (new — pure gauge math via OperationsSummaryResult → gauge values)

- [x] **Step 1:** Test a pure helper `processGaugesFromSummary(OperationsSummaryResult) (stuck, blocked int)` returns the stuck + blocked bucket counts; and that ResolveBlocker/RetryStep write a tenant audit entry (assert the audit-entry construction via an injected fake AuditRepository-like — use the real `*repository.AuditRepository` with a nil-safe guard so absence of audit repo = no-op).
- [x] **Step 2:** Run → FAIL.
- [x] **Step 3:** Add nil-safe `metrics` + `audit *repository.AuditRepository` to the read service via setters; after `OperationsSummary` set stuck/blocked gauges; in ResolveBlocker and RetryStep, write a best-effort audit entry (`fulfillment.blocker.resolved`, `fulfillment.step.retried`) inside the same tenant tx, log-and-continue on error (must not fail the action). Operator user id taken from a new optional `actorID uuid.UUID` param OR from context — inspect how the handler passes actor; if not available, audit with `UserID: uuid.Nil` (nilIfUUIDEmpty handles it) and entity ids.
- [x] **Step 4:** Run → PASS.
- [x] **Step 5:** Commit.

### Task 7: Provider validation + publication transition metrics

**Files:**
- Modify: `internal/service/provider_validation_service.go`
- Modify: `internal/service/provider_registry_service.go`
- Test: covered via metrics-collector unit tests (Task 1) + build/vet; add a small pure test if a helper is extracted.

- [x] **Step 1:** Add nil-safe `metrics` setter to both services; in `CompleteRun` record `RecordValidationRun(verdict)` and `RecordValidationFailure()` per failed/errored result; in `Transition` record `RecordPublicationTransition(toState)` after successful commit. Best-effort, nil-safe.
- [x] **Step 2:** `go build ./... && go vet ./...` → OK.
- [x] **Step 3:** Commit.

### Task 8: Wire in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [x] **Step 1:** Construct `fulfillmentMetrics := metrics.NewFulfillmentMetrics()` before the fulfillment services (move/insert near line 320). Inject via the new setters into fulfillmentService, fulfillmentReadService, providerRegistryService, providerValidationService, and the orchestration worker (line 1005). Register it with `metricsCollector` (`metricsCollector.Register(fulfillmentMetrics)`).
- [x] **Step 2:** `go build ./...` → OK.
- [x] **Step 3:** Commit.

### Task 9: Full validation + final commit

- [x] `gofmt -w -s .` then `gofmt -l .` clean.
- [x] `go build ./...`, `go vet ./...`, `go test ./...` all pass.
- [x] `golangci-lint run` → 0 issues.
- [x] Final squash/clean commit with the required subject.

## Deferred (enterprise repo / separate tasks)

- Grafana dashboards, alert rules — enterprise monitoring stack.
- Operational runbooks — enterprise repo.
- SLO-definition narrative doc, security-review writeup — separate docs.
- Any metric whose source signal isn't emitted yet by current code.
