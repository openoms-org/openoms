# OPE-291 Worker Lock Renewal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent long-running background workers from losing their Redis distributed lock before the worker run finishes.

**Architecture:** Keep the existing Redis `SET NX` lock acquisition model, but treat it as a renewable lease while a worker run is active. The worker manager will renew the lease periodically, cancel the worker context if renewal proves the lease was lost, and skip a run instead of proceeding when the distributed lock layer returns an error.

**Tech Stack:** Go 1.25, `github.com/redis/go-redis/v9`, worker package unit tests with `testify`, OpenOMS public repo local CI.

---

## Scope

This plan fixes `OPE-291` in the public repo. It does not change marketplace order idempotency because `OPE-203` already added the partial unique index and `CreateIfExternalIDNotExists` path. The remaining production risk is the worker manager lease lifecycle: a slow poll can exceed `Interval + 30s`, allowing another pod to acquire the same worker lock.

## Files

- Modify: `apps/api-server/internal/worker/distributed_lock.go`
  - Add an atomic Redis lease extension primitive.
  - Keep the UUID ownership check pattern used by release.
- Modify: `apps/api-server/internal/worker/distributed_lock_test.go`
  - Cover nil/single-pod behavior for the new extension method.
- Modify: `apps/api-server/internal/worker/manager.go`
  - Add a small lock interface for testability.
  - Add lease TTL and renewal interval helpers.
  - Start a renewal loop for acquired distributed locks.
  - Cancel the worker run if the lock is lost.
  - Skip worker execution when acquiring the distributed lock errors.
- Modify: `apps/api-server/internal/worker/manager_test.go`
  - Add fake lock tests for fail-closed behavior and lease-loss cancellation.
  - Add helper tests for TTL/renewal interval calculations.
- Modify: `docs/system-documentation.md`
  - Update the worker infrastructure note from one-shot SETNX lock to renewable Redis lease.

## Task 0: Branch and Issue State

**Files:**
- No code files.

- [ ] **Step 1: Create the feature branch before editing**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git checkout -b fix/OPE-291-worker-lock-renewal
```

Expected: branch `fix/OPE-291-worker-lock-renewal` is checked out.

- [ ] **Step 2: Move Linear issue to In Progress**

Move `OPE-291` to `In Progress` before implementation begins.

## Task 1: Add Manager Lock Abstraction and Timing Tests

**Files:**
- Modify: `apps/api-server/internal/worker/manager.go`
- Modify: `apps/api-server/internal/worker/manager_test.go`

- [ ] **Step 1: Add failing helper tests**

Add these tests near the worker manager overlap-prevention tests in `apps/api-server/internal/worker/manager_test.go`:

```go
func TestWorkerLockTTL_UsesIntervalBuffer(t *testing.T) {
	assert.Equal(t, 75*time.Second, workerLockTTL(45*time.Second))
	assert.Equal(t, 150*time.Second, workerLockTTL(2*time.Minute))
}

func TestWorkerLockRenewInterval_IsBounded(t *testing.T) {
	assert.Equal(t, 25*time.Second, workerLockRenewInterval(75*time.Second))
	assert.Equal(t, 30*time.Second, workerLockRenewInterval(24*time.Hour))
	assert.Equal(t, 10*time.Millisecond, workerLockRenewInterval(15*time.Millisecond))
}
```

- [ ] **Step 2: Run the failing tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run 'TestWorkerLock(TTL|RenewInterval)' -count=1
```

Expected: FAIL because `workerLockTTL` and `workerLockRenewInterval` are not defined.

- [ ] **Step 3: Add the lock interface and timing helpers**

In `apps/api-server/internal/worker/manager.go`, add this near the `Worker` interface:

```go
type workerLock interface {
	Acquire(ctx context.Context, workerName string, ttl time.Duration) (string, error)
	Extend(ctx context.Context, workerName string, token string, ttl time.Duration) (bool, error)
	Release(workerName string, token string)
}
```

Change the manager field:

```go
lock workerLock
```

Add these helpers below `NewManager`:

```go
func workerLockTTL(interval time.Duration) time.Duration {
	return interval + 30*time.Second
}

func workerLockRenewInterval(ttl time.Duration) time.Duration {
	interval := ttl / 3
	if interval < 10*time.Millisecond {
		return 10 * time.Millisecond
	}
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}
```

- [ ] **Step 4: Run the helper tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run 'TestWorkerLock(TTL|RenewInterval)' -count=1
```

Expected: PASS.

## Task 2: Add Redis Lease Extension Primitive

**Files:**
- Modify: `apps/api-server/internal/worker/distributed_lock.go`
- Modify: `apps/api-server/internal/worker/distributed_lock_test.go`

- [ ] **Step 1: Add failing tests for nil/single-pod extension behavior**

Add these tests to `apps/api-server/internal/worker/distributed_lock_test.go`:

```go
func TestDistributedLock_ExtendNilStruct(t *testing.T) {
	var dl *DistributedLock

	ok, err := dl.Extend(context.Background(), "test-worker", "no-redis", time.Second)
	require.NoError(t, err)
	assert.True(t, ok, "nil lock extension should succeed in single-pod mode")
}

func TestDistributedLock_ExtendNilClient(t *testing.T) {
	dl := NewDistributedLock(nil, "openoms")

	ok, err := dl.Extend(context.Background(), "test-worker", "no-redis", time.Second)
	require.NoError(t, err)
	assert.True(t, ok, "nil Redis client extension should succeed in single-pod mode")
}
```

- [ ] **Step 2: Run the failing tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run 'TestDistributedLock_Extend' -count=1
```

Expected: FAIL because `DistributedLock.Extend` is not defined.

- [ ] **Step 3: Add the Redis extension script and method**

In `apps/api-server/internal/worker/distributed_lock.go`, add this Lua script after `releaseScript`:

```go
var extendScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)
```

Add this method after `Acquire`:

```go
func (d *DistributedLock) Extend(ctx context.Context, workerName string, token string, ttl time.Duration) (bool, error) {
	if d == nil || d.client == nil || token == "no-redis" {
		return true, nil
	}
	if token == "" {
		return false, nil
	}
	if ttl <= 0 {
		return false, fmt.Errorf("distributed lock: TTL must be positive, got %v", ttl)
	}
	key := fmt.Sprintf("%s:worker-lock:%s", d.prefix, workerName)
	result, err := extendScript.Run(ctx, d.client, []string{key}, token, ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
```

- [ ] **Step 4: Run the extension tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run 'TestDistributedLock_(Extend|NilStruct|NilClient|InvalidTTL|NewDistributedLock)' -count=1
```

Expected: PASS.

## Task 3: Renew Locks During Worker Runs and Fail Closed on Lock Errors

**Files:**
- Modify: `apps/api-server/internal/worker/manager.go`
- Modify: `apps/api-server/internal/worker/manager_test.go`

- [ ] **Step 1: Add fake lock test helper**

Add this helper to `apps/api-server/internal/worker/manager_test.go` near `stubWorker`:

```go
type fakeWorkerLock struct {
	acquireToken string
	acquireErr   error
	extendOK    atomic.Bool
	extendErr   atomic.Value
	released     atomic.Bool
}

func newFakeWorkerLock(token string) *fakeWorkerLock {
	f := &fakeWorkerLock{acquireToken: token}
	f.extendOK.Store(true)
	return f
}

func (f *fakeWorkerLock) Acquire(_ context.Context, _ string, _ time.Duration) (string, error) {
	return f.acquireToken, f.acquireErr
}

func (f *fakeWorkerLock) Extend(_ context.Context, _ string, _ string, _ time.Duration) (bool, error) {
	if err, ok := f.extendErr.Load().(error); ok && err != nil {
		return false, err
	}
	return f.extendOK.Load(), nil
}

func (f *fakeWorkerLock) Release(_ string, _ string) {
	f.released.Store(true)
}
```

- [ ] **Step 2: Add failing fail-closed acquisition test**

Add this test:

```go
func TestGuardedRun_SkipsExecutionWhenDistributedLockErrors(t *testing.T) {
	var running atomic.Bool
	var ran atomic.Bool

	w := &stubWorker{
		name:     "lock-error-worker",
		interval: 50 * time.Millisecond,
		runFn: func(_ context.Context) error {
			ran.Store(true)
			return nil
		},
	}

	lock := newFakeWorkerLock("")
	lock.acquireErr = errors.New("redis unavailable")

	m := NewManager(nil, slog.Default())
	m.lock = lock

	m.guardedRun(context.Background(), w, &running)

	assert.False(t, ran.Load(), "worker must not run when distributed lock acquisition fails")
}
```

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestGuardedRun_SkipsExecutionWhenDistributedLockErrors -count=1
```

Expected: FAIL because current code logs the lock error and proceeds.

- [ ] **Step 3: Add failing lease-loss cancellation test**

Add this test:

```go
func TestGuardedRun_CancelsWorkerWhenDistributedLockRenewalIsLost(t *testing.T) {
	var running atomic.Bool
	workerStarted := make(chan struct{})
	workerStopped := make(chan struct{})

	w := &stubWorker{
		name:     "renewal-loss-worker",
		interval: 15 * time.Millisecond,
		runFn: func(ctx context.Context) error {
			close(workerStarted)
			<-ctx.Done()
			close(workerStopped)
			return ctx.Err()
		},
	}

	lock := newFakeWorkerLock("lease-token")
	m := NewManager(nil, slog.Default())
	m.lock = lock

	go m.guardedRun(context.Background(), w, &running)

	require.Eventually(t, func() bool {
		select {
		case <-workerStarted:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)

	lock.extendOK.Store(false)

	require.Eventually(t, func() bool {
		select {
		case <-workerStopped:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)

	require.Eventually(t, func() bool { return lock.released.Load() }, time.Second, 5*time.Millisecond)
}
```

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestGuardedRun_CancelsWorkerWhenDistributedLockRenewalIsLost -count=1
```

Expected: FAIL because the manager does not renew locks or cancel the worker context on lease loss.

- [ ] **Step 4: Add failing renewal-error cancellation test**

Add this test:

```go
func TestGuardedRun_CancelsWorkerWhenDistributedLockRenewalErrors(t *testing.T) {
	var running atomic.Bool
	workerStarted := make(chan struct{})
	workerStopped := make(chan struct{})

	w := &stubWorker{
		name:     "renewal-error-worker",
		interval: 15 * time.Millisecond,
		runFn: func(ctx context.Context) error {
			close(workerStarted)
			<-ctx.Done()
			close(workerStopped)
			return ctx.Err()
		},
	}

	lock := newFakeWorkerLock("lease-token")
	lock.extendErr.Store(errors.New("redis timeout"))
	m := NewManager(nil, slog.Default())
	m.lock = lock

	go m.guardedRun(context.Background(), w, &running)

	require.Eventually(t, func() bool {
		select {
		case <-workerStarted:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)

	require.Eventually(t, func() bool {
		select {
		case <-workerStopped:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)
}
```

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestGuardedRun_CancelsWorkerWhenDistributedLockRenewalErrors -count=1
```

Expected: FAIL because the manager does not renew locks or cancel the worker context on renewal errors.

- [ ] **Step 5: Implement fail-closed lock acquisition and renewal loop**

In `guardedRun`, replace the distributed lock block with this structure:

```go
runCtx := ctx
var runCancel context.CancelFunc
var renewDone <-chan struct{}

if m.lock != nil {
	lockTTL := workerLockTTL(w.Interval())
	token, err := m.lock.Acquire(ctx, w.Name(), lockTTL)
	switch {
	case err != nil:
		m.logger.Error("distributed lock error, skipping worker run", "worker", w.Name(), "error", err)
		return
	case token == "":
		return
	default:
		runCtx, runCancel = context.WithCancel(ctx)
		renewDone = m.renewDistributedLock(runCtx, w.Name(), token, lockTTL, workerLockRenewInterval(lockTTL), runCancel)
		defer func() {
			runCancel()
			<-renewDone
			m.lock.Release(w.Name(), token)
		}()
	}
}
```

Then change the final worker execution call:

```go
m.safeRun(runCtx, w)
```

Add this method below `guardedRun`:

```go
func (m *Manager) renewDistributedLock(
	ctx context.Context,
	workerName string,
	token string,
	ttl time.Duration,
	interval time.Duration,
	cancel context.CancelFunc,
) <-chan struct{} {
	done := make(chan struct{})
	asyncutil.SafeGo(func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ok, err := m.lock.Extend(ctx, workerName, token, ttl)
				if err != nil {
					m.logger.Error("distributed lock renewal failed, cancelling worker run",
						"worker", workerName,
						"error", err,
					)
					cancel()
					return
				}
				if !ok {
					m.logger.Error("distributed lock lost, cancelling worker run",
						"worker", workerName,
					)
					cancel()
					return
				}
			}
		}
	})
	return done
}
```

- [ ] **Step 6: Run manager tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run 'TestGuardedRun|TestWorkerLock|TestDistributedLock' -count=1
```

Expected: PASS.

## Task 4: Update Worker Documentation

**Files:**
- Modify: `docs/system-documentation.md`

- [ ] **Step 1: Find current worker lock wording**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
rg -n "distributed_lock|SETNX|Lua release|multi-instance|worker" docs/system-documentation.md
```

- [ ] **Step 2: Update wording**

Replace the worker infrastructure sentence that currently describes Redis `SETNX` plus Lua release with:

```markdown
Worker infrastructure: `manager.go` (lifecycle), `marketplace_order_poller.go` (shared polling logic), `tenant_iterator.go` (per-tenant execution), `distributed_lock.go` (Redis `SET NX` worker leases with UUID ownership, periodic Lua-based renewal while a run is active, and Lua-based release for multi-instance safety)
```

- [ ] **Step 3: Verify docs diff**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff -- docs/system-documentation.md
```

Expected: only the worker infrastructure sentence changes.

## Task 5: Validation and PR

**Files:**
- Validate all modified files.

- [ ] **Step 1: Format Go files**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
gofmt -w -s apps/api-server/internal/worker/distributed_lock.go apps/api-server/internal/worker/distributed_lock_test.go apps/api-server/internal/worker/manager.go apps/api-server/internal/worker/manager_test.go
```

- [ ] **Step 2: Run targeted worker tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -count=1
```

Expected: PASS.

- [ ] **Step 3: Run API tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Run public quick CI**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh --quick
```

Expected: PASS.

- [ ] **Step 5: Self-review diff**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
git diff --stat
git diff
```

Expected:
- no whitespace errors,
- no unrelated files,
- no secrets,
- lock errors skip worker execution instead of proceeding,
- lock renewal cancels worker context when lease is lost,
- docs updated.

- [ ] **Step 6: Full local CI before push**

Run after committing:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: PASS and fresh `/tmp/openoms-local-ci-full-results.txt` for current clean `HEAD`.

- [ ] **Step 7: Commit and PR**

Commit and push:

```bash
cd /Users/rafs/praca/openoms-dev/public
git add apps/api-server/internal/worker/distributed_lock.go apps/api-server/internal/worker/distributed_lock_test.go apps/api-server/internal/worker/manager.go apps/api-server/internal/worker/manager_test.go docs/system-documentation.md
git commit -m "OPE-291: renew worker distributed locks"
git push -u origin fix/OPE-291-worker-lock-renewal
```

PR title:

```text
OPE-291: renew worker distributed locks
```

PR body must include:

```markdown
## Summary
- renew Redis worker leases while a worker run is active
- cancel a worker run when its distributed lease is lost
- skip worker execution when distributed lock acquisition fails

## Test plan
- [x] `cd apps/api-server && go test ./internal/worker -count=1`
- [x] `cd apps/api-server && go test ./... -count=1`
- [x] `./scripts/local-ci.sh --quick`
- [x] `./scripts/local-ci.sh`

## Docs updated
- [x] `docs/system-documentation.md` — updated worker distributed-lock lifecycle wording
```

## Risk and Rollback

Risk: A Redis outage will now skip worker ticks instead of allowing duplicate multi-pod execution. That is intentional for production correctness: delayed polling is safer than duplicate order/import/automation side effects. If this causes unexpected availability issues, rollback is the PR revert; workers return to one-shot lock TTL behavior.

Risk: Worker cancellation depends on workers checking `ctx.Done()`. Many worker loops already do this after prior worker hardening, but a worker blocked inside an external provider call may finish its current provider call before stopping. The renewable lease prevents normal long runs from losing the lock; cancellation is the safety path when renewal fails or another owner appears.

Rollback: Revert the PR and redeploy through the standard public release and enterprise deploy workflow.

## Self-Review Checklist

- [ ] The plan covers the remaining `OPE-291` risk after `OPE-203`.
- [ ] The plan does not change order uniqueness or migration behavior.
- [ ] Distributed lock acquisition errors no longer proceed with worker execution.
- [ ] Lease renewal uses UUID ownership checks before extending TTL.
- [ ] Worker context is cancelled when renewal fails or returns ownership lost.
- [ ] Tests cover helper timing, lock acquisition errors, and lease-loss cancellation.
- [ ] Documentation reflects renewable worker leases.
