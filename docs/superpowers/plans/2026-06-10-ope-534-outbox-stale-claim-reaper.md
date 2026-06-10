# Orchestration Outbox Stale-Claim Reaper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the outbox's at-least-once guarantee across worker crashes: rows stuck in `status='claimed'` (process-level crash between `ClaimDue` and `Mark*`) are detected by `claimed_at` age and requeued — or failed with a blocker when attempts are exhausted.

**Architecture:** A reap pass at the start of every `OrchestrationWorker.Run` tick, BEFORE `ClaimDue`. A new repository method lists stale claimed rows (claimed longer than a visibility timeout); the worker requeues each via the existing `MarkFailedRetry` (which already increments `attempts`, resets `status='pending'`, clears `claimed_at`) or, when the interrupted attempt exhausts `max_attempts`, marks it permanently failed and opens the standard fulfillment blocker. Dangling `running` rows in `orchestration_attempts` for the reaped event are closed as failed. No migration: `claimed_at timestamptz` already exists (migration 000037), and `MarkFailedRetry`/`MarkFailedPermanent` already maintain the counters.

**Tech Stack:** Go 1.25, pgx/v5 on the privileged worker pool (cross-tenant — same justification as `ClaimDue`), existing `obsmetrics` counters, integration tests on the disposable-Postgres harness (`-tags integration`).

**Context for the engineer (read these first, in full):**
- `apps/api-server/internal/worker/orchestration_worker.go` — the worker you are extending. Note `nextRetryAt` (OPE-522), `openBlocker`, `blockerCodeForEvent`, the nil-safe `metrics` handle, and that `parkAwaitingCallback` re-queues parked events to `pending` (so a legitimately parked external-workflow event is NEVER left `claimed` — `claimed` is strictly a transient state lasting seconds, except after a crash).
- `apps/api-server/internal/repository/orchestration_repository.go` — `ClaimDue` (the claim you are reaping), `MarkFailedRetry` (increments attempts + clears claim — the reap primitive), `MarkFailedPermanent`, `StartAttempt`/`FinishAttempt`, the `orchestrationOutboxColumns` constant and `scanOutbox`.
- `apps/api-server/migrations/000037_orchestration_outbox.up.sql` — schema: `status CHECK IN ('pending','claimed','succeeded','failed')`, `claimed_at timestamptz`, `attempts`/`max_attempts`.
- `apps/api-server/tests/integration/orchestration_test.go` — the established worker-test pattern: `superPool` as the privileged worker pool, `appPool` + `database.WithTenant` for tenant-scoped seeding, `seedTenant`/`seedFulfillmentOrder` helpers.
- `apps/api-server/internal/obsmetrics/fulfillment_metrics.go` — `allowedOutboxResults = set("claimed", "processed", "failed")`; you will add `"reaped"`.

**Design decisions (settled — do not relitigate):**
- Visibility timeout: package constant `staleClaimTimeout = 10 * time.Minute`. Far above any legitimate synchronous dispatch (outbound HTTP calls are client-timeout-bounded to seconds), far below operator-noticeable starvation. Not configurable (YAGNI).
- The reaped interruption COUNTS as an attempt: `MarkFailedRetry` already does `attempts = attempts + 1`, so repeated crash-loops converge to `max_attempts` → permanent failure + blocker instead of looping forever.
- Exhaustion check mirrors the worker's existing rule (`attemptNumber >= e.MaxAttempts` where `attemptNumber = e.Attempts + 1`).
- Reap errors are best-effort: log and continue to the normal claim — a broken reap must never stop outbox draining.
- Per-row processing (not one bulk UPDATE): exhausted rows need the blocker side effect, and the batch is small (`reapBatchLimit = 50`).

---

### Task 1: Repository — `ListStaleClaimed` + `FailRunningAttempts`

**Files:**
- Modify: `apps/api-server/internal/repository/orchestration_repository.go` (append after `ClaimDue`, around line 100)
- Test: `apps/api-server/tests/integration/orchestration_reaper_test.go` (new; integration-tagged — these methods are SQL, unit tests cannot cover them)

- [ ] **Step 1: Write the failing integration test for both methods**

Create `apps/api-server/tests/integration/orchestration_reaper_test.go`:

```go
//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"

	"github.com/jackc/pgx/v5"
)

// seedClaimedEvent enqueues an outbox event for a fresh order/process and force-claims
// it with the given claimed_at age via the superuser pool (simulating a worker that
// claimed the row and then crashed before marking it).
func seedClaimedEvent(t *testing.T, ctx context.Context, tenantID uuid.UUID, age time.Duration, attempts, maxAttempts int) model.OrchestrationOutboxEvent {
	t.Helper()
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantID, "Reaper Customer")
	repo := repository.NewOrchestrationRepository()
	var ev *model.OrchestrationOutboxEvent
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, created, err := repo.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
			TenantID:       tenantID,
			ProcessID:      processID,
			EventType:      "reaper.test",
			IdempotencyKey: "reaper.test:" + orderID.String(),
			Payload:        map[string]any{"order_id": orderID.String()},
		})
		ev = created
		return err
	}))
	require.NotNil(t, ev)
	_, err := superPool.Exec(ctx,
		`UPDATE orchestration_outbox
		    SET status='claimed', claimed_at = now() - make_interval(secs => $2),
		        attempts = $3, max_attempts = $4, updated_at = now()
		  WHERE id = $1`,
		ev.ID, age.Seconds(), attempts, maxAttempts)
	require.NoError(t, err)
	ev.Attempts, ev.MaxAttempts = attempts, maxAttempts
	return *ev
}

func TestOrchestrationReaper_RepoMethods(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	repo := repository.NewOrchestrationRepository()

	stale := seedClaimedEvent(t, ctx, tenantA, 30*time.Minute, 0, 5)
	fresh := seedClaimedEvent(t, ctx, tenantA, 1*time.Minute, 0, 5)

	// ListStaleClaimed returns only the stale row.
	got, err := repo.ListStaleClaimed(ctx, superPool, 10*time.Minute, 50)
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for i := range got {
		ids[got[i].ID] = true
	}
	assert.True(t, ids[stale.ID], "stale claimed row must be listed")
	assert.False(t, ids[fresh.ID], "fresh claimed row must NOT be listed")

	// FailRunningAttempts closes a dangling running attempt for the stale event.
	att, err := repo.StartAttempt(ctx, superPool, tenantA, stale.ID, stale.Attempts+1)
	require.NoError(t, err)
	n, err := repo.FailRunningAttempts(ctx, superPool, stale.ID, "reaped: worker crashed mid-attempt")
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	var status, errMsg string
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status, error FROM orchestration_attempts WHERE id = $1`, att.ID).Scan(&status, &errMsg))
	assert.Equal(t, model.AttemptStatusFailed, status)
	assert.Contains(t, errMsg, "reaped")
}
```

Note: add `"github.com/google/uuid"` to the imports.

- [ ] **Step 2: Run the test to verify it fails to compile**

Run:
```bash
cd apps/api-server
docker exec openoms-postgres-1 psql -U openoms -c "DROP DATABASE IF EXISTS openoms_reaper" && docker exec openoms-postgres-1 psql -U openoms -c "CREATE DATABASE openoms_reaper"
DATABASE_URL="postgres://openoms:openoms-dev-password@localhost:5433/openoms_reaper?sslmode=disable" NO_COLOR=1 go test -tags integration -run TestOrchestrationReaper_RepoMethods ./tests/integration/ -count=1
```
Expected: compile error — `repo.ListStaleClaimed undefined` / `repo.FailRunningAttempts undefined`.

- [ ] **Step 3: Implement both repository methods**

Append to `apps/api-server/internal/repository/orchestration_repository.go` directly after `ClaimDue`:

```go
// ListStaleClaimed returns claimed outbox rows whose claim is older than olderThan —
// evidence of a worker that crashed between ClaimDue and Mark* (OPE-534). Cross-tenant
// by design (same privileged-pool justification as ClaimDue): the reaper must see every
// tenant's stranded rows. FOR UPDATE SKIP LOCKED so concurrent worker instances never
// reap the same row twice, and a row being actively processed (its claimer still holds
// no lock — claims are NOT held in a tx) is protected by the age threshold instead.
func (r *OrchestrationRepository) ListStaleClaimed(ctx context.Context, q Querier, olderThan time.Duration, limit int) ([]model.OrchestrationOutboxEvent, error) {
	rows, err := q.Query(ctx,
		`SELECT `+orchestrationOutboxColumns+`
		   FROM orchestration_outbox
		  WHERE status = 'claimed' AND claimed_at < now() - make_interval(secs => $1)
		  ORDER BY claimed_at
		  LIMIT $2
		    FOR UPDATE SKIP LOCKED`,
		olderThan.Seconds(), limit)
	if err != nil {
		return nil, fmt.Errorf("list stale claimed outbox events: %w", err)
	}
	defer rows.Close()
	result := []model.OrchestrationOutboxEvent{}
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stale outbox event: %w", err)
		}
		result = append(result, *e)
	}
	return result, rows.Err()
}

// FailRunningAttempts closes every dangling 'running' attempt row for an outbox event
// as failed (OPE-534). A worker crash mid-attempt leaves the attempt 'running' forever;
// the reaper closes it so the attempt timeline stays truthful. Returns the number of
// rows closed.
func (r *OrchestrationRepository) FailRunningAttempts(ctx context.Context, q Querier, outboxID uuid.UUID, errMsg string) (int, error) {
	tag, err := q.Exec(ctx,
		`UPDATE orchestration_attempts
		    SET status = 'failed', error = $2, finished_at = now()
		  WHERE outbox_id = $1 AND status = 'running'`,
		outboxID, errMsg)
	if err != nil {
		return 0, fmt.Errorf("fail running outbox attempts: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
```

Note: `time` and `uuid` are already imported in this file; `tag.RowsAffected()` returns int64 — the cast to `int` is bounded by the batch size.

- [ ] **Step 4: Run the test to verify it passes**

Same command as Step 2. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/orchestration_repository.go tests/integration/orchestration_reaper_test.go
git commit -m "OPE-534: repo methods to list stale claimed events and close dangling attempts"
```

---

### Task 2: Worker — reap pass at the start of every tick

**Files:**
- Modify: `apps/api-server/internal/worker/orchestration_worker.go` (constants near line 24; `Run` near line 107; new method after `Run`)
- Modify: `apps/api-server/internal/obsmetrics/fulfillment_metrics.go` (`allowedOutboxResults`)
- Test: `apps/api-server/internal/obsmetrics/fulfillment_metrics_test.go` (extend the allow-list test)

- [ ] **Step 1: Extend the bounded outbox-result allow-list**

In `apps/api-server/internal/obsmetrics/fulfillment_metrics.go` change:

```go
	allowedOutboxResults      = set("claimed", "processed", "failed")
```
to:
```go
	allowedOutboxResults      = set("claimed", "processed", "failed", "reaped")
```

Extend the existing allow-list/cardinality test in `fulfillment_metrics_test.go`: find the test that asserts allowed outbox results (grep `allowedOutboxResults` / `RecordOutboxEvent`) and add `"reaped"` to its accepted set, mirroring how the existing values are asserted.

- [ ] **Step 2: Add the reaper constants and the reap method to the worker**

In `apps/api-server/internal/worker/orchestration_worker.go`, extend the constants block:

```go
// orchestrationBatchLimit caps how many due outbox rows are processed per run.
const orchestrationBatchLimit = 50

// staleClaimTimeout is the visibility timeout for claimed outbox rows (OPE-534).
// A claim is transient — processEvent runs synchronously and parked events are
// re-queued to 'pending' in the same flow — so a claim older than this is evidence
// of a worker that crashed between ClaimDue and Mark*. Far above any legitimate
// dispatch duration (outbound HTTP is client-timeout-bounded), far below
// operator-noticeable starvation.
const staleClaimTimeout = 10 * time.Minute

// reapBatchLimit caps how many stale claims are reaped per tick.
const reapBatchLimit = 50
```

Add the method (after `Run`):

```go
// reapStaleClaims requeues outbox rows stranded in 'claimed' by a worker crash
// (OPE-534): the interrupted attempt counts toward max_attempts (MarkFailedRetry
// increments the counter), dangling 'running' attempt rows are closed, and an
// exhausted event fails permanently with the standard blocker. Best-effort: any
// error is logged and never blocks the normal claim/dispatch cycle.
func (w *OrchestrationWorker) reapStaleClaims(ctx context.Context) {
	stale, err := w.repo.ListStaleClaimed(ctx, w.pool, staleClaimTimeout, reapBatchLimit)
	if err != nil {
		w.logger.Warn("orchestration stale-claim reap skipped (best-effort)", "error", err)
		return
	}
	for i := range stale {
		e := stale[i]
		log := w.logger.With(
			"correlation_id", e.IdempotencyKey,
			"event_id", e.ID,
			"event_type", e.EventType,
			"process_id", e.ProcessID,
			"claimed_at", e.ClaimedAt,
		)
		if _, aerr := w.repo.FailRunningAttempts(ctx, w.pool, e.ID, "reaped: worker crashed mid-attempt"); aerr != nil {
			log.Warn("orchestration reap: close dangling attempts failed (best-effort)", "error", aerr)
		}
		reapErrMsg := "reaped: claim exceeded visibility timeout (worker crash)"
		if e.Attempts+1 >= e.MaxAttempts {
			if merr := w.repo.MarkFailedPermanent(ctx, w.pool, e.ID, reapErrMsg); merr != nil {
				log.Error("orchestration reap: mark permanent failed", "error", merr)
				continue
			}
			w.openBlocker(ctx, e, errors.New(reapErrMsg))
			log.Warn("orchestration reap: event exhausted attempts, failed permanently")
		} else {
			if merr := w.repo.MarkFailedRetry(ctx, w.pool, e.ID, reapErrMsg, nextRetryAt(time.Now().UTC(), e.Attempts)); merr != nil {
				log.Error("orchestration reap: requeue failed", "error", merr)
				continue
			}
			log.Warn("orchestration reap: stale claim requeued")
		}
		w.recordOutcome("reaped")
	}
}
```

Call it at the START of `Run`:

```go
// Run claims and processes one batch of due outbox events. A reap pass first
// requeues claims stranded by a crashed worker (OPE-534).
func (w *OrchestrationWorker) Run(ctx context.Context) error {
	w.reapStaleClaims(ctx)
	events, err := w.repo.ClaimDue(ctx, w.pool, w.batchLimit)
	...unchanged...
```

- [ ] **Step 3: Build + unit tests + lint**

```bash
cd apps/api-server
gofmt -w -s internal/worker/orchestration_worker.go internal/obsmetrics/ internal/repository/orchestration_repository.go
NO_COLOR=1 go build ./... && go vet ./... && NO_COLOR=1 go test ./internal/obsmetrics/ ./internal/worker/ ./internal/repository/
/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m
```
Expected: build clean, tests pass, `0 issues.`

- [ ] **Step 4: Commit**

```bash
git add internal/worker/orchestration_worker.go internal/obsmetrics/
git commit -m "OPE-534: reap stale claimed outbox events at the start of each worker tick"
```

---

### Task 3: Integration tests — end-to-end reaper behavior

**Files:**
- Modify: `apps/api-server/tests/integration/orchestration_reaper_test.go` (extend)

- [ ] **Step 1: Write the end-to-end tests**

Append to `orchestration_reaper_test.go` (reuse the worker-construction pattern from `orchestration_test.go` — read it first for the exact dispatcher/worker setup; the dispatcher below registers a handler for `reaper.test` so the requeued event actually processes):

```go
// TestOrchestrationReaper_EndToEnd: a stale claimed event is requeued by the reap pass
// and processed on the SAME tick; a fresh claim is untouched; an exhausted stale claim
// fails permanently with a blocker.
func TestOrchestrationReaper_EndToEnd(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	repo := repository.NewOrchestrationRepository()
	fRepo := repository.NewFulfillmentRepository()

	disp := service.NewOrchestrationDispatcher()
	disp.Register("reaper.test", orchestrationHandlerFunc(func(ctx context.Context, e model.OrchestrationOutboxEvent) error {
		return nil // ack — we only care about the reap + reprocess mechanics
	}))
	w := worker.NewOrchestrationWorker(superPool, repo, disp, fRepo, time.Second, nil)

	// (a) stale claim, attempts available -> requeued and processed within one tick.
	stale := seedClaimedEvent(t, ctx, tenantA, 30*time.Minute, 0, 5)
	// (b) fresh claim -> untouched by the reaper.
	fresh := seedClaimedEvent(t, ctx, tenantA, 1*time.Minute, 0, 5)
	// (c) stale claim with attempts exhausted (attempts+1 >= max) -> permanent + blocker.
	exhausted := seedClaimedEvent(t, ctx, tenantA, 30*time.Minute, 4, 5)

	require.NoError(t, w.Run(ctx))

	var status string
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status FROM orchestration_outbox WHERE id = $1`, stale.ID).Scan(&status))
	assert.Equal(t, "succeeded", status, "stale claim must be requeued and processed")

	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status FROM orchestration_outbox WHERE id = $1`, fresh.ID).Scan(&status))
	assert.Equal(t, "claimed", status, "fresh claim must not be reaped")

	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status FROM orchestration_outbox WHERE id = $1`, exhausted.ID).Scan(&status))
	assert.Equal(t, "failed", status, "exhausted stale claim must fail permanently")

	// The exhausted event must have opened a blocker on its process.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		blockers, err := fRepo.ListBlockers(ctx, tx, exhausted.ProcessID)
		if err != nil {
			return err
		}
		require.Len(t, blockers, 1, "exhausted reap must open a blocker")
		assert.Contains(t, blockers[0].Description, "reaped")
		return nil
	}))
}
```

IMPORTANT adaptations the engineer must make while writing this test (read the real files — do not guess):
- `orchestrationHandlerFunc`: check how `orchestration_test.go` registers handlers on `service.NewOrchestrationDispatcher()`. If the dispatcher's `Register` takes an interface rather than a func, reuse the existing test-handler type from that file (or define a small local adapter exactly like the existing tests do). Match the actual signature.
- Timing nuance: `seedClaimedEvent` for case (a) sets `next_attempt_at` in the past? `EnqueueEvent` defaults `next_attempt_at = now()`. After the reap pass requeues with `nextRetryAt(now, 0)` = now + 30s backoff, the SAME tick's `ClaimDue` will NOT pick it up (next_attempt_at is in the future). If so, assert `status = 'pending'` with `attempts = 1` and `claimed_at IS NULL` after the first `w.Run`, then force `next_attempt_at = now()` via superPool and run `w.Run` again to assert it processes to `succeeded`. PREFER this two-step honest version over fighting the backoff — adjust the (a) assertions accordingly.

- [ ] **Step 2: Run the new tests + the FULL integration suite (regression)**

```bash
cd apps/api-server
docker exec openoms-postgres-1 psql -U openoms -c "DROP DATABASE IF EXISTS openoms_reaper" && docker exec openoms-postgres-1 psql -U openoms -c "CREATE DATABASE openoms_reaper"
DATABASE_URL="postgres://openoms:openoms-dev-password@localhost:5433/openoms_reaper?sslmode=disable" NO_COLOR=1 go test -tags integration -run TestOrchestrationReaper ./tests/integration/ -count=1 -v 2>&1 | tail -20
DATABASE_URL="postgres://openoms:openoms-dev-password@localhost:5433/openoms_reaper?sslmode=disable" NO_COLOR=1 go test -tags integration ./tests/integration/ -count=1 2>&1 | tail -3
```
Expected: new tests PASS; full suite `ok` (no regression — particularly the existing `TestOrchestration_OutboxWorkerLifecycle`, whose claims are always fresh and must be unaffected by the reap pass).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/orchestration_reaper_test.go
git commit -m "OPE-534: end-to-end reaper integration tests (requeue, fresh-claim safety, exhaustion blocker)"
```

---

### Task 4: Documentation

**Files:**
- Modify: `docs/system-documentation.md` (§8 orchestration outbox fact-sheet — the block describing OrchestrationWorker retry behavior, around the line starting `Retry:`)

- [ ] **Step 1: Add the reaper line to the orchestration fact-sheet**

In the OPE-415 orchestration section, extend the fact-sheet (after the `Retry:` line) with:

```
Reaper:      każdy tick zaczyna się od reap-pass (OPE-534): wiersze 'claimed' starsze niż
             10 min (crash workera między claim a mark) wracają do 'pending' z backoffem
             (przerwana próba liczy się do attempts); wyczerpane attempts → failed +
             fulfillment_blocker; wiszące 'running' attempts zamykane jako failed.
```

(Polish — this file is the Polish system documentation; match the surrounding style.)

- [ ] **Step 2: Commit**

```bash
git add docs/system-documentation.md
git commit -m "OPE-534: document the outbox stale-claim reaper"
```

---

### Final validation (all must pass before handoff)

```bash
cd apps/api-server
gofmt -l internal/ tests/ cmd/            # expect: empty
NO_COLOR=1 go build ./...                  # expect: clean
go vet ./... && go vet -tags integration ./tests/...
NO_COLOR=1 go test ./...                   # expect: all pass
/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m   # expect: 0 issues.
# full integration suite on the scratch DB (commands in Task 3 Step 2)
```
