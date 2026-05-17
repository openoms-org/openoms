# OPE-315 Worker Batch Limits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent tracking and delayed-action workers from loading unbounded work queues into memory during one tick.

**Architecture:** Keep the fix small and local to the worker/repository boundary. The tracking poller should query only a stable, bounded batch of trackable shipments. The delayed-action worker should own the batch-size decision and pass it to the repository, replacing the current hidden `LIMIT 100` literal with an explicit parameter.

**Tech Stack:** Go 1.25, pgx/v5, OpenOMS worker package, repository package, testify tests.

---

## Scope Notes

- `DelayedActionRepository.ListPending` already contains `LIMIT 100` on current `origin/main`; this issue is partially stale for delayed actions.
- The remaining bug is `TrackingPoller.Run`, which still selects all active trackable shipments without `LIMIT`.
- We still improve delayed actions by making the batch limit explicit and tested, because the worker currently cannot document or tune its own batch behavior.
- No DB migration is planned. This PR limits per-tick memory and work. A future performance task can add a partial index if production query plans show it is needed.

## Files

- Modify: `apps/api-server/internal/worker/tracking_poller.go`
  - Add `trackingPollerBatchLimit`.
  - Extract the tracking SQL into a small helper so the limit/order can be unit-tested.
  - Change the SQL to use active integrations with credentials, stable ordering, and `LIMIT $1`.
- Create: `apps/api-server/internal/worker/tracking_poller_test.go`
  - Regression test that the tracking query has stable ordering and parameterized batch limit.
- Modify: `apps/api-server/internal/repository/interfaces.go`
  - Change `DelayedActionRepo.ListPending(ctx, tx)` to `ListPending(ctx, tx, limit)`.
- Modify: `apps/api-server/internal/automation/engine.go`
  - Keep the interface signature in sync even though the engine does not call `ListPending`.
- Modify: `apps/api-server/internal/repository/delayed_action_repository.go`
  - Add `defaultDelayedActionPendingLimit`.
  - Change `ListPending` to accept a limit argument, default invalid limits to the safe default, add stable `ORDER BY execute_at ASC, id ASC`, and use `LIMIT $1`.
- Modify: `apps/api-server/internal/worker/delayed_action_worker.go`
  - Add `delayedActionBatchLimit`.
  - Pass the constant to `ListPending`.
  - Log when a full batch was processed so backlog behavior is visible.
- Modify: `apps/api-server/internal/worker/delayed_action_worker_test.go`
  - Regression test for the worker batch limit constant if useful.
- Create: `apps/api-server/internal/repository/delayed_action_repository_test.go`
  - Regression test that `ListPending` uses the caller-provided `LIMIT` argument, stable ordering, and default fallback for invalid limits.
- Docs: no public system documentation update expected because API, DB schema, and user-facing behavior do not change.

## Implementation Tasks

### Task 1: Tracking poller RED test

**Files:**
- Create: `apps/api-server/internal/worker/tracking_poller_test.go`

- [ ] **Step 1: Add failing query regression test**

Add a test in package `worker`:

```go
func TestTrackableShipmentsQueryUsesStableBatchLimit(t *testing.T) {
	query := trackableShipmentsQuery()

	assert.Contains(t, query, "ORDER BY s.updated_at ASC, s.id ASC")
	assert.Contains(t, query, "LIMIT $1")
	assert.NotContains(t, query, "LEFT JOIN integrations")
	assert.Equal(t, 100, trackingPollerBatchLimit)
}
```

- [ ] **Step 2: Run RED**

Run:

```bash
cd apps/api-server
go test ./internal/worker -run TestTrackableShipmentsQueryUsesStableBatchLimit -count=1
```

Expected: fail to compile because `trackableShipmentsQuery` and `trackingPollerBatchLimit` do not exist.

### Task 2: Tracking poller GREEN implementation

**Files:**
- Modify: `apps/api-server/internal/worker/tracking_poller.go`

- [ ] **Step 1: Add limit constant and query helper**

Add near the top of the file:

```go
const trackingPollerBatchLimit = 100
```

Add this helper:

```go
func trackableShipmentsQuery() string {
	return `SELECT s.id, s.tenant_id, s.provider, s.tracking_number, s.status, s.carrier_data,
	        i.credentials, i.settings
	 FROM shipments s
	 JOIN integrations i ON i.id = s.integration_id
	   AND i.status = 'active'
	   AND i.credentials IS NOT NULL
	   AND i.credentials <> '""'::jsonb
	   AND i.credentials <> '{}'::jsonb
	   AND i.credentials <> 'null'::jsonb
	 WHERE s.tracking_number IS NOT NULL
	   AND s.tracking_number <> ''
	   AND s.status NOT IN ('delivered', 'returned', 'failed', 'cancelled')
	 ORDER BY s.updated_at ASC, s.id ASC
	 LIMIT $1`
}
```

- [ ] **Step 2: Use the helper in `Run`**

Replace the inline `w.pool.Query` SQL with:

```go
rows, err := w.pool.Query(ctx, trackableShipmentsQuery(), trackingPollerBatchLimit)
```

- [ ] **Step 3: Run GREEN**

Run:

```bash
cd apps/api-server
go test ./internal/worker -run TestTrackableShipmentsQueryUsesStableBatchLimit -count=1
```

Expected: pass.

### Task 3: Delayed action repository RED test

**Files:**
- Create: `apps/api-server/internal/repository/delayed_action_repository_test.go`

- [ ] **Step 1: Add fake tx/rows and failing tests**

Add tests that call the future `ListPending(ctx, tx, limit)` signature:

```go
func TestDelayedActionRepositoryListPendingUsesCallerLimit(t *testing.T) {
	tx := &captureQueryTx{rows: emptyRows{}}
	repo := NewDelayedActionRepository()

	_, err := repo.ListPending(context.Background(), tx, 37)

	require.NoError(t, err)
	assert.Contains(t, tx.query, "ORDER BY execute_at ASC, id ASC")
	assert.Contains(t, tx.query, "LIMIT $1")
	require.Len(t, tx.args, 1)
	assert.Equal(t, 37, tx.args[0])
}

func TestDelayedActionRepositoryListPendingDefaultsInvalidLimit(t *testing.T) {
	tx := &captureQueryTx{rows: emptyRows{}}
	repo := NewDelayedActionRepository()

	_, err := repo.ListPending(context.Background(), tx, 0)

	require.NoError(t, err)
	require.Len(t, tx.args, 1)
	assert.Equal(t, defaultDelayedActionPendingLimit, tx.args[0])
}
```

The helper types should implement only the pgx methods needed by this test, following the existing `captureExecTx` style in `audit_repository_test.go`.

- [ ] **Step 2: Run RED**

Run:

```bash
cd apps/api-server
go test ./internal/repository -run 'TestDelayedActionRepositoryListPending' -count=1
```

Expected: fail to compile because `ListPending` still accepts two arguments and `defaultDelayedActionPendingLimit` does not exist.

### Task 4: Delayed action GREEN implementation

**Files:**
- Modify: `apps/api-server/internal/repository/delayed_action_repository.go`
- Modify: `apps/api-server/internal/repository/interfaces.go`
- Modify: `apps/api-server/internal/automation/engine.go`
- Modify: `apps/api-server/internal/worker/delayed_action_worker.go`

- [ ] **Step 1: Change repository interface signatures**

Change both `DelayedActionRepo` interfaces:

```go
ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]model.DelayedAction, error)
```

- [ ] **Step 2: Implement parameterized repository limit**

Add:

```go
const defaultDelayedActionPendingLimit = 100
```

Change `ListPending`:

```go
func (r *DelayedActionRepository) ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]model.DelayedAction, error) {
	if limit <= 0 {
		limit = defaultDelayedActionPendingLimit
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, action_index, order_id, execute_at,
		        executed, executed_at, error, attempt_count, last_attempt_at,
		        created_at, action_data, event_data
		 FROM automation_delayed_actions
		 WHERE execute_at <= NOW() AND NOT executed
		 ORDER BY execute_at ASC, id ASC
		 LIMIT $1`,
		limit,
	)
	...
}
```

- [ ] **Step 3: Pass the worker limit**

In `delayed_action_worker.go`, add:

```go
const delayedActionBatchLimit = 100
```

Change:

```go
pending, err = w.delayedRepo.ListPending(ctx, tx, delayedActionBatchLimit)
```

After the processing log, optionally add:

```go
if len(pending) == delayedActionBatchLimit {
	w.logger.Info("delayed action worker: processed full batch", "batch_limit", delayedActionBatchLimit)
}
```

- [ ] **Step 4: Run GREEN**

Run:

```bash
cd apps/api-server
go test ./internal/repository -run 'TestDelayedActionRepositoryListPending' -count=1
go test ./internal/worker -run 'TestTrackableShipmentsQueryUsesStableBatchLimit|TestPlanDelayedActionFailure' -count=1
```

Expected: pass.

### Task 5: Self-review and validation

**Files:**
- All modified files above.

- [ ] **Step 1: Format**

Run:

```bash
cd .
gofmt -w -s apps/api-server/internal/worker/tracking_poller.go \
  apps/api-server/internal/worker/tracking_poller_test.go \
  apps/api-server/internal/worker/delayed_action_worker.go \
  apps/api-server/internal/worker/delayed_action_worker_test.go \
  apps/api-server/internal/repository/delayed_action_repository.go \
  apps/api-server/internal/repository/delayed_action_repository_test.go \
  apps/api-server/internal/repository/interfaces.go \
  apps/api-server/internal/automation/engine.go
```

- [ ] **Step 2: Self-review**

Run:

```bash
cd .
git diff --check
git diff --stat
git diff
```

Expected: whitespace clean; diff limited to OPE-315 scope and this plan.

- [ ] **Step 3: Targeted tests**

Run:

```bash
cd apps/api-server
go test ./internal/worker ./internal/repository ./internal/automation -count=1
```

Expected: pass.

- [ ] **Step 4: Full local CI after commit**

After committing the implementation, run:

```bash
cd .
./scripts/local-ci.sh
```

Expected: pass and `/tmp/openoms-local-ci-full-results.txt` contains `STATUS=pass` for the clean `HEAD`.

## Risk And Rollback

- Risk: A fixed tracking batch can leave a very large backlog to be processed over multiple ticks. This is intentional; it prevents one tick from exhausting memory or worker time.
- Risk: Filtering tracking rows to active integrations with credentials changes log behavior for shipments without usable credentials. Those shipments were not trackable before either; the worker already skipped them.
- Rollback: revert the PR. No schema changes or data migrations are involved.
- Follow-up: if production query plans show slow scans, create a separate migration for a partial index covering active tracking candidates.

## Self-Review

- Spec coverage: tracking poller receives a hard `LIMIT`; delayed-action worker receives an explicit, tested batch limit.
- Placeholder scan: no TBD or deferred implementation placeholders.
- Type consistency: repository and automation interfaces use the same `ListPending(ctx, tx, limit int)` signature.
