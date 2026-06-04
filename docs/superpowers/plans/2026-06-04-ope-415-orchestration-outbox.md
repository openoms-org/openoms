# OPE-415 Orchestration Outbox, Attempts, Idempotent Worker & Retry

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** A transactional outbox + attempts model and an idempotent background worker that durably executes fulfillment side effects with retry/backoff and permanent-failure→blocker, on top of the OPE-414 model. The side-effect *handlers* themselves are pluggable and land in OPE-416+ — this task delivers the durable queue + worker framework.

**Architecture:** `orchestration_outbox` + `orchestration_attempts` (tenant-scoped, RLS). Side effects are enqueued **in the same transaction** as the fulfillment state change (`EnqueueEvent(tx)` within `database.WithTenant`). A background `OrchestrationWorker` claims due rows **across tenants** via the privileged `workerPool` using `FOR UPDATE SKIP LOCKED`, runs them through a pluggable `OrchestrationDispatcher` (per-`event_type` handler registry), records an attempt, and on failure either retries with backoff or — when permanent or attempts are exhausted — marks the row failed and opens a `fulfillment_blocker`. Idempotency: enqueue dedupes on `(tenant_id, idempotency_key)`.

**Tech Stack:** Go 1.25, pgx/v5, PostgreSQL RLS, the existing worker `Manager`/`Worker` interface + distributed lock, testify.

---

## Key decisions

1. **Cross-tenant claim via `workerPool`.** The worker claims globally (all tenants) with `SKIP LOCKED` on the privileged pool that bypasses RLS (same pool the webhook resolver uses). Enqueue stays RLS-scoped (within `WithTenant`).
2. **Pluggable dispatcher, handlers deferred.** `OrchestrationDispatcher` is a `map[eventType]Handler`. OPE-415 ships it empty (+ a test handler); OPE-416/417/418 register real handlers. An unregistered `event_type` is a **permanent** failure → blocker (`integration_capability_missing`).
3. **Retry policy.** Per-row `attempts`/`max_attempts` (default 8) + exponential backoff (`next_attempt_at = now + base*2^attempts`, capped). Handlers signal permanence via a `PermanentError` wrapper; permanent or exhausted → `failed` + blocker, else → `pending` re-queued.
4. **Foundation:** the worker is registered (gated by config) but does no real work until handlers exist; safe to ship.

## Data model (migration 000037) — TENANT-SCOPED, RLS

```sql
CREATE TABLE public.orchestration_outbox (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id uuid NOT NULL REFERENCES public.fulfillment_processes(id) ON DELETE CASCADE,
    unit_id uuid REFERENCES public.fulfillment_units(id) ON DELETE SET NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','claimed','succeeded','failed')),
    idempotency_key text NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 8,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    claimed_at timestamptz,
    last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX idx_orchestration_outbox_due ON public.orchestration_outbox (status, next_attempt_at); -- migrate:index-lock-ok

CREATE TABLE public.orchestration_attempts (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    outbox_id uuid NOT NULL REFERENCES public.orchestration_outbox(id) ON DELETE CASCADE,
    attempt_number integer NOT NULL,
    status text NOT NULL CHECK (status IN ('running','succeeded','failed')),
    error text NOT NULL DEFAULT '',
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);
CREATE INDEX idx_orchestration_attempts_outbox ON public.orchestration_attempts (tenant_id, outbox_id); -- migrate:index-lock-ok
```
Both: `ENABLE`+`FORCE ROW LEVEL SECURITY` + `tenant_isolation` (current_setting). Dual-role grants.

## Implementation Tasks (TDD)

### Task 1: Migration + model
- [ ] `migrations/000037_orchestration_outbox.up.sql`/`.down.sql`.
- [ ] `internal/model/orchestration.go`: `OrchestrationOutboxEvent`, `OrchestrationAttempt` structs; status enums + `IsValidOutboxStatus/AttemptStatus`; `NextOutboxBackoff(attempts int) time.Duration` (base 30s × 2^attempts, capped 1h); `PermanentError` type + `IsPermanent(err) bool`.
- [ ] `model/orchestration_test.go`: enum + backoff (monotonic, capped) + PermanentError matrices.

### Task 2: Repository
- [ ] `repository/orchestration_repository.go`:
  - `EnqueueEvent(ctx, tx, e) (created bool, *OrchestrationOutboxEvent, error)` — INSERT … ON CONFLICT (tenant_id, idempotency_key) DO NOTHING RETURNING; created=false when duplicate.
  - `ClaimDue(ctx, q Querier, limit int) ([]OrchestrationOutboxEvent, error)` — UPDATE … WHERE id IN (SELECT … WHERE status='pending' AND next_attempt_at<=now() ORDER BY next_attempt_at LIMIT $1 FOR UPDATE SKIP LOCKED) SET status='claimed', claimed_at=now() RETURNING …. (worker pool / cross-tenant)
  - `MarkSucceeded(ctx, q, id)`, `MarkFailedRetry(ctx, q, id, errMsg, nextAt)`, `MarkFailedPermanent(ctx, q, id, errMsg)`.
  - `StartAttempt(ctx, q, e) (*OrchestrationAttempt, error)`, `FinishAttempt(ctx, q, id, status, errMsg)`.
  - `GetEvent(ctx, q, id)`, `ListByProcess(ctx, tx, processID)`.

### Task 3: Dispatcher + worker
- [ ] `service/orchestration_dispatcher.go`: `OrchestrationHandler interface { Handle(ctx, OrchestrationOutboxEvent) error }`; `OrchestrationDispatcher` with `Register(eventType, handler)` + `Dispatch(ctx, e) error` (unregistered → `PermanentError`).
- [ ] `worker/orchestration_worker.go`: `OrchestrationWorker{workerPool, repo, dispatcher, fulfillment *repository.FulfillmentRepository, batchSize}`. `Name()="orchestration"`, `Interval()` from config. `Run(ctx)`: claim batch; per event → StartAttempt → Dispatch → FinishAttempt; success→MarkSucceeded; on error: if `IsPermanent` or `attempts+1>=max_attempts` → MarkFailedPermanent + open a `fulfillment_blocker` (via `database.WithTenant(workerPool, event.TenantID)` using the FulfillmentRepository) ; else → MarkFailedRetry(next=now+NextOutboxBackoff). Recover panics (Sentry), bounded batch.
- [ ] Tests: worker unit with a fake dispatcher (success / retryable / permanent) + a fake/real repo — assert MarkSucceeded vs retry (next_attempt advanced) vs permanent+blocker.

### Task 4: Config + manager wiring
- [ ] `config.go`: `OrchestrationWorkerEnabled bool env:"ORCHESTRATION_WORKER_ENABLED" envDefault:"false"` + interval default.
- [ ] `cmd/server/main.go`: construct repo + dispatcher (+register no real handlers yet) + worker; `if cfg.WorkersEnabled && cfg.OrchestrationWorkerEnabled { workerMgr.Register(orchestrationWorker) }`. Wire the OPE-414 `FulfillmentRepository` here too (it gets its first bootstrap use).

### Task 5: Integration test (live DB)
- [ ] `tests/integration/orchestration_test.go`: enqueue (within WithTenant) 2 events for tenant A; idempotent re-enqueue (created=false). `ClaimDue(workerPool, 10)` claims both; a second immediate `ClaimDue` claims none (no double-claim). MarkSucceeded one; MarkFailedRetry the other (assert status pending + next_attempt_at future + an attempt row). Permanent path → status failed. RLS: reads under tenant B see nothing.

### Task 6: Validate + docs
- [ ] go test ./..., vet, gofmt, golangci-lint (full); integration green; migrate down/up.
- [ ] `docs/system-documentation.md` — orchestration outbox/worker section; worker count note.

## Risks
- **Double-claim:** prevented by `FOR UPDATE SKIP LOCKED`; covered by the concurrent-claim integration assertion.
- **Cross-tenant worker:** claim uses the privileged pool (bypasses RLS) intentionally; blocker creation re-enters the row's tenant via `WithTenant`. No tenant data crosses.
- **Lost side effects:** outbox row persists until `succeeded`; worker restart re-claims `pending`/stale `claimed` (a `claimed` reclaim guard via `next_attempt_at` + a claim-timeout sweep can be a follow-up; for OPE-415, claimed→succeeded/failed within the same Run).
- **No real handlers yet:** safe — worker is config-gated off by default; unregistered events become visible blockers, not silent drops.

## Self-Review
Covers OPE-415: outbox + attempts tables (RLS), idempotent enqueue, `SKIP LOCKED` claim, retry/backoff, permanent-failure→blocker, pluggable dispatcher, worker registration. Real per-event handlers + order-creation routing are OPE-416+.
