// Package database provides PostgreSQL connection helpers.
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolOptions controls pgxpool sizing for different database roles.
type PoolOptions struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultPoolOptions returns the pool sizing used for the RLS-scoped app pool.
func DefaultPoolOptions() PoolOptions {
	return PoolOptions{
		MaxConns:          20,
		MinConns:          2,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// WorkerPoolOptions returns pool sizing for the privileged cross-tenant worker pool.
//
// IMPORTANT: unlike the app pool (transaction pooler, port 6543, 200 clients), this pool
// connects through the Supabase SESSION-mode pooler (port 5432), which is hard-limited to
// pool_size: 15 clients TOTAL across every pod. Exceeding it makes new connections fail
// immediately with `FATAL: (EMAXCONNSESSION) max clients reached in session mode`, which at
// startup crashes the pod (run() returns after the connect retries) and, during a blue-green
// rollout, aborts the deploy. So this pool must stay small and must NOT hold eager idle
// connections: MinConns=0 means a pod only opens a session connection when a worker actually
// runs (plus a transient one for the startup Ping), instead of permanently reserving some.
// Aggregate worst case stays well under 15: blue+green API pods (lazy, ~1 ping each) + the
// worker Deployment (≤5) ≈ 10 of 15. set_config(..., is_local=true) per transaction keeps
// tenant scoping correct. (POOLCONN-01 wrongly assumed the transaction pooler here and set
// MinConns=2/MaxConns=10, which exhausted the 15-client session cap during deploys.)
func WorkerPoolOptions() PoolOptions {
	return PoolOptions{
		MaxConns:          5,
		MinConns:          0,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// Connect opens a pgx connection pool for the given PostgreSQL URL.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return ConnectWithOptions(ctx, databaseURL, DefaultPoolOptions())
}

// ConnectWithOptions opens a pgx connection pool with explicit sizing options.
func ConnectWithOptions(ctx context.Context, databaseURL string, opts PoolOptions) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	config.MaxConns = opts.MaxConns
	config.MinConns = opts.MinConns
	config.MaxConnLifetime = opts.MaxConnLifetime
	config.MaxConnIdleTime = opts.MaxConnIdleTime
	config.HealthCheckPeriod = opts.HealthCheckPeriod

	// Register json.RawMessage as JSONB so pgx sends it as text (valid JSON)
	// instead of bytea hex encoding. Required for simple_protocol mode (Supabase pooler).
	config.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		tm := conn.TypeMap()
		tm.RegisterType(&pgtype.Type{
			Name:  "jsonb",
			OID:   pgtype.JSONBOID,
			Codec: &pgtype.JSONBCodec{Marshal: json.Marshal, Unmarshal: json.Unmarshal},
		})
		tm.RegisterDefaultPgType(json.RawMessage{}, "jsonb")
		return nil
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}

// ConnectWithRetry opens a pool, retrying transient connect/ping failures with
// exponential backoff (baseDelay, 2x, 4x, ...). Startup must tolerate a brief
// database hiccup — e.g. a momentary Supabase pooler blip during a blue-green
// rollout, when extra pods connect at once — rather than crash the pod: a crash
// turns a transient blip into a CrashLoopBackOff that can outlast the deploy's
// pre-promotion gate and abort an otherwise healthy release. A genuinely down
// database still fails after attempts are exhausted, so a real outage is not
// masked indefinitely.
func ConnectWithRetry(ctx context.Context, databaseURL string, opts PoolOptions, attempts int, baseDelay time.Duration) (*pgxpool.Pool, error) {
	return connectWithRetry(ctx, attempts, baseDelay, func(c context.Context) (*pgxpool.Pool, error) {
		return ConnectWithOptions(c, databaseURL, opts)
	})
}

// connectWithRetry is the testable core of ConnectWithRetry: it retries connect
// independently of how the pool is actually opened.
func connectWithRetry(ctx context.Context, attempts int, baseDelay time.Duration, connect func(context.Context) (*pgxpool.Pool, error)) (*pgxpool.Pool, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		pool, err := connect(ctx)
		if err == nil {
			return pool, nil
		}
		lastErr = err
		if attempt == attempts {
			break
		}
		delay := baseDelay * time.Duration(1<<(attempt-1))
		slog.Warn("database connect failed, retrying",
			"attempt", attempt, "max_attempts", attempts, "retry_in", delay.String(), "error", err)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("database connect failed after %d attempts: %w", attempts, lastErr)
}
