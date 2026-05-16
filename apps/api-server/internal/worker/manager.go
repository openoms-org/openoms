package worker

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
)

// Worker is the interface for background workers.
type Worker interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

type workerLock interface {
	Acquire(ctx context.Context, workerName string, ttl time.Duration) (string, error)
	Extend(ctx context.Context, workerName string, token string, ttl time.Duration) (bool, error)
	Release(workerName string, token string)
}

// Manager manages background workers.
type Manager struct {
	pool                *pgxpool.Pool
	workers             []Worker
	wg                  sync.WaitGroup
	cancel              context.CancelFunc
	logger              *slog.Logger
	lock                workerLock
	lockRenewIntervalFn func(time.Duration) time.Duration
}

// NewManager creates a new worker Manager.
// If redisClient is nil, the manager operates in single-pod mode (no distributed locking).
func NewManager(pool *pgxpool.Pool, logger *slog.Logger, redisClient ...*redis.Client) *Manager {
	m := &Manager{
		pool:                pool,
		logger:              logger,
		lockRenewIntervalFn: workerLockRenewInterval,
	}
	if len(redisClient) > 0 && redisClient[0] != nil {
		m.lock = NewDistributedLock(redisClient[0], "openoms")
		logger.Info("worker manager using distributed locking (Redis)")
	}
	return m
}

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

func (m *Manager) lockRenewInterval(ttl time.Duration) time.Duration {
	if m.lockRenewIntervalFn == nil {
		return workerLockRenewInterval(ttl)
	}
	return m.lockRenewIntervalFn(ttl)
}

// Register adds a worker to the manager's run list.
func (m *Manager) Register(w Worker) {
	m.workers = append(m.workers, w)
}

// Start launches all registered workers in background goroutines.
func (m *Manager) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)
	m.logger.Info("worker manager starting", "workers", len(m.workers))
	for _, w := range m.workers {
		m.wg.Add(1)
		worker := w
		asyncutil.SafeGo(func() { m.runWorker(ctx, worker) })
	}
	m.logger.Info("worker manager started")
}

// Stop signals all workers to stop and waits for them to finish.
func (m *Manager) Stop() {
	m.logger.Info("worker manager stopping")
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	m.logger.Info("worker manager stopped")
}

func (m *Manager) runWorker(ctx context.Context, w Worker) {
	defer m.wg.Done()

	var running atomic.Bool

	m.logger.Info("worker started", "name", w.Name(), "interval", w.Interval())

	// Immediate first run on startup
	m.guardedRun(ctx, w, &running)

	ticker := time.NewTicker(w.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("worker stopped", "name", w.Name())
			return
		case <-ticker.C:
			m.guardedRun(ctx, w, &running)
		}
	}
}

// guardedRun ensures only one execution of a worker at a time, both in-process
// (via atomic.Bool) and across pods (via a renewable Redis lease).
func (m *Manager) guardedRun(ctx context.Context, w Worker, running *atomic.Bool) {
	// Distributed lock: prevent multiple pods from running the same worker.
	// The lock is acquired as a renewable lease and extended while the worker
	// run is active. If Redis errors or ownership is lost, the run context is
	// cancelled so cooperative workers stop before another pod proceeds.
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
			return // Another pod holds the lock
		default:
			runCtx, runCancel = context.WithCancel(ctx)
			renewDone = m.renewDistributedLock(runCtx, w.Name(), token, lockTTL, m.lockRenewInterval(lockTTL), runCancel)
			defer func() {
				runCancel()
				<-renewDone
				m.lock.Release(w.Name(), token)
			}()
		}
	}

	// In-process guard: prevent overlapping executions within the same pod.
	if !running.CompareAndSwap(false, true) {
		m.logger.Warn("worker still running, skipping tick", "name", w.Name())
		return
	}
	defer running.Store(false)
	m.safeRun(runCtx, w)
}

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

func (m *Manager) safeRun(ctx context.Context, w Worker) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("worker panicked", "worker", w.Name(), "error", r, "stack", string(debug.Stack()))
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("worker", w.Name())
				sentry.CurrentHub().RecoverWithContext(ctx, r)
			})
		}
	}()
	if err := w.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			m.logger.Info("worker run stopped", "name", w.Name(), "error", err)
			return
		}
		m.logger.Error("worker run failed", "name", w.Name(), "error", err)
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("worker", w.Name())
			scope.SetLevel(sentry.LevelError)
			sentry.CaptureException(err)
		})
	}
}
