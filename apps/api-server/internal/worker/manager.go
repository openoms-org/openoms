package worker

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Worker is the interface for background workers.
type Worker interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

// Manager manages background workers.
// TODO: For multi-instance deployment, use Redis SETNX-based distributed
// locking to ensure only one instance runs each worker at a time.
type Manager struct {
	pool    *pgxpool.Pool
	workers []Worker
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	logger  *slog.Logger
}

func NewManager(pool *pgxpool.Pool, logger *slog.Logger) *Manager {
	return &Manager{
		pool:   pool,
		logger: logger,
	}
}

func (m *Manager) Register(w Worker) {
	m.workers = append(m.workers, w)
}

func (m *Manager) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)
	m.logger.Info("worker manager starting", "workers", len(m.workers))
	for _, w := range m.workers {
		m.wg.Add(1)
		go m.runWorker(ctx, w)
	}
	m.logger.Info("worker manager started")
}

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

// guardedRun ensures only one execution of a worker at a time.
func (m *Manager) guardedRun(ctx context.Context, w Worker, running *atomic.Bool) {
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
		}
	}()
	if err := w.Run(ctx); err != nil {
		m.logger.Error("worker run failed", "name", w.Name(), "error", err)
	}
}
