package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// stubWorker implements the Worker interface for testing.
// ---------------------------------------------------------------------------

type stubWorker struct {
	name     string
	interval time.Duration
	runFn    func(ctx context.Context) error
}

func (w *stubWorker) Name() string            { return w.name }
func (w *stubWorker) Interval() time.Duration { return w.interval }
func (w *stubWorker) Run(ctx context.Context) error {
	if w.runFn != nil {
		return w.runFn(ctx)
	}
	return nil
}

type fakeWorkerLock struct {
	acquireToken string
	acquireErr   error
	extendOK     atomic.Bool
	extendErr    atomic.Value
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

// ---------------------------------------------------------------------------
// NewManager
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	logger := slog.Default()
	m := NewManager(nil, logger)

	require.NotNil(t, m)
	assert.Empty(t, m.workers)
	assert.Equal(t, logger, m.logger)
	assert.Nil(t, m.cancel, "cancel should be nil before Start")
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_SingleWorker(t *testing.T) {
	m := NewManager(nil, slog.Default())
	w := &stubWorker{name: "test-worker", interval: time.Second}

	m.Register(w)

	require.Len(t, m.workers, 1)
	assert.Equal(t, "test-worker", m.workers[0].Name())
}

func TestRegister_MultipleWorkers(t *testing.T) {
	m := NewManager(nil, slog.Default())

	w1 := &stubWorker{name: "worker-1", interval: time.Second}
	w2 := &stubWorker{name: "worker-2", interval: 2 * time.Second}
	w3 := &stubWorker{name: "worker-3", interval: 3 * time.Second}

	m.Register(w1)
	m.Register(w2)
	m.Register(w3)

	require.Len(t, m.workers, 3)
	assert.Equal(t, "worker-1", m.workers[0].Name())
	assert.Equal(t, "worker-2", m.workers[1].Name())
	assert.Equal(t, "worker-3", m.workers[2].Name())
}

// ---------------------------------------------------------------------------
// Start + Stop lifecycle
// ---------------------------------------------------------------------------

func TestStartStop_NoWorkers(_ *testing.T) {
	m := NewManager(nil, slog.Default())
	m.Start(context.Background())
	m.Stop()
	// Should not hang or panic with zero workers.
}

func TestStartStop_WorkerExecutes(t *testing.T) {
	var runCount atomic.Int32
	w := &stubWorker{
		name:     "counter",
		interval: 50 * time.Millisecond,
		runFn: func(_ context.Context) error {
			runCount.Add(1)
			return nil
		},
	}

	m := NewManager(nil, slog.Default())
	m.Register(w)
	m.Start(context.Background())

	// Worker does an immediate first run, then ticks at 50ms.
	// Wait enough time for the immediate run + at least one tick.
	time.Sleep(120 * time.Millisecond)

	m.Stop()

	count := runCount.Load()
	assert.GreaterOrEqual(t, count, int32(2), "worker should have run at least twice (immediate + tick)")
}

func TestStartStop_WorkerErrorDoesNotCrash(_ *testing.T) {
	w := &stubWorker{
		name:     "error-worker",
		interval: 50 * time.Millisecond,
		runFn: func(_ context.Context) error {
			return errors.New("simulated failure")
		},
	}

	m := NewManager(nil, slog.Default())
	m.Register(w)
	m.Start(context.Background())
	time.Sleep(80 * time.Millisecond)
	m.Stop()
	// Manager should gracefully handle worker errors without crashing.
}

func TestStartStop_WorkerPanicRecovery(t *testing.T) {
	var ranAfterPanic atomic.Bool
	runCount := 0
	w := &stubWorker{
		name:     "panicker",
		interval: 50 * time.Millisecond,
		runFn: func(_ context.Context) error {
			runCount++
			if runCount == 1 {
				panic("test panic")
			}
			ranAfterPanic.Store(true)
			return nil
		},
	}

	m := NewManager(nil, slog.Default())
	m.Register(w)
	m.Start(context.Background())

	// Wait long enough for immediate run (panic) + at least one tick (should recover).
	time.Sleep(150 * time.Millisecond)
	m.Stop()

	assert.True(t, ranAfterPanic.Load(), "worker should continue running after a panic in a previous run")
}

func TestStop_CalledBeforeStart(_ *testing.T) {
	m := NewManager(nil, slog.Default())
	// Stop without Start should not panic (cancel is nil).
	m.Stop()
}

func TestStartStop_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var runCount atomic.Int32
	w := &stubWorker{
		name:     "ctx-worker",
		interval: 10 * time.Millisecond,
		runFn: func(_ context.Context) error {
			runCount.Add(1)
			return nil
		},
	}

	m := NewManager(nil, slog.Default())
	m.Register(w)
	m.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	// Stop should return quickly since the context is already cancelled.
	done := make(chan struct{})
	go func() {
		m.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return in time after context cancellation")
	}
}

// ---------------------------------------------------------------------------
// guardedRun — overlap prevention
// ---------------------------------------------------------------------------

func TestWorkerLockTTL_UsesIntervalBuffer(t *testing.T) {
	assert.Equal(t, 75*time.Second, workerLockTTL(45*time.Second))
	assert.Equal(t, 150*time.Second, workerLockTTL(2*time.Minute))
}

func TestWorkerLockRenewInterval_IsBounded(t *testing.T) {
	assert.Equal(t, 25*time.Second, workerLockRenewInterval(75*time.Second))
	assert.Equal(t, 30*time.Second, workerLockRenewInterval(24*time.Hour))
	assert.Equal(t, 10*time.Millisecond, workerLockRenewInterval(15*time.Millisecond))
}

func TestGuardedRun_SkipsOverlappingExecution(t *testing.T) {
	var running atomic.Bool
	var started atomic.Bool
	blocker := make(chan struct{})

	w := &stubWorker{
		name:     "slow-worker",
		interval: time.Hour, // irrelevant, we call guardedRun directly
		runFn: func(_ context.Context) error {
			started.Store(true)
			<-blocker
			return nil
		},
	}

	m := NewManager(nil, slog.Default())

	// First call: blocks inside Run()
	go m.guardedRun(context.Background(), w, &running)

	// Wait for the first run to start
	require.Eventually(t, started.Load, time.Second, 5*time.Millisecond)

	// Second call while first is running: should be skipped (no-op)
	secondRan := false
	origFn := w.runFn
	w.runFn = func(ctx context.Context) error {
		secondRan = true
		return origFn(ctx)
	}
	m.guardedRun(context.Background(), w, &running)
	assert.False(t, secondRan, "second guardedRun should have been skipped while first is running")

	// Unblock the first run
	close(blocker)

	// Wait for the running flag to clear
	require.Eventually(t, func() bool { return !running.Load() }, time.Second, 5*time.Millisecond)
}

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
	m.lockRenewIntervalFn = func(time.Duration) time.Duration { return 10 * time.Millisecond }

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
	m.lockRenewIntervalFn = func(time.Duration) time.Duration { return 10 * time.Millisecond }

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

// ---------------------------------------------------------------------------
// Worker interface compliance
// ---------------------------------------------------------------------------

func TestWorkerInterface(t *testing.T) {
	w := &stubWorker{
		name:     "interface-test",
		interval: 5 * time.Second,
	}

	// Verify the stub satisfies the Worker interface.
	var _ Worker = w

	assert.Equal(t, "interface-test", w.Name())
	assert.Equal(t, 5*time.Second, w.Interval())
	assert.NoError(t, w.Run(context.Background()))
}
