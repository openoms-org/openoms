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
)

// Worker is the interface for background workers.
type Worker interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

// Manager manages background workers.
type Manager struct {
	pool    *pgxpool.Pool
	workers []Worker
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	logger  *slog.Logger
	lock    *DistributedLock
}

// NewManager creates a new worker Manager.
// If redisClient is nil, the manager operates in single-pod mode (no distributed locking).
func NewManager(pool *pgxpool.Pool, logger *slog.Logger, redisClient ...*redis.Client) *Manager {
	m := &Manager{
		pool:   pool,
		logger: logger,
	}
	if len(redisClient) > 0 && redisClient[0] != nil {
		m.lock = NewDistributedLock(redisClient[0], "openoms")
		logger.Info("worker manager using distributed locking (Redis)")
	}
	return m
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
		go m.runWorker(ctx, w)
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
// (via atomic.Bool) and across pods (via Redis SETNX distributed lock).
func (m *Manager) guardedRun(ctx context.Context, w Worker, running *atomic.Bool) {
	// Distributed lock: prevent multiple pods from running the same worker.
	// TTL = worker interval + 30s buffer so the lock outlives a single execution
	// but auto-expires if the pod crashes.
	if m.lock != nil {
		token, err := m.lock.Acquire(ctx, w.Name(), w.Interval()+30*time.Second)
		switch {
		case err != nil:
			m.logger.Warn("distributed lock error, proceeding anyway", "worker", w.Name(), "error", err)
		case token == "":
			return // Another pod holds the lock
		default:
			defer m.lock.Release(w.Name(), token)
		}
	}

	// In-process guard: prevent overlapping executions within the same pod.
	if !running.CompareAndSwap(false, true) {
		m.logger.Warn("worker still running, skipping tick", "name", w.Name())
		return
	}
	defer running.Store(false)
	m.safeRun(ctx, w)
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
