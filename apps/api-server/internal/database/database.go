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
// ~27 background workers run concurrently (one goroutine each); a 3-connection pool
// serialized them, so several pollers firing on the same tick queued on the pool before
// even reaching the database. Production connects through the Supabase TRANSACTION pooler
// (max 200 client connections, multiplexed onto a 15-connection database pool), so client
// connections are cheap: the aggregate budget is well within limits — at most
// API maxReplicas(5) x DefaultPoolOptions.MaxConns(20) + this pool(10) = ~110 of 200 — and
// the database itself is protected by the pooler's 15-connection cap regardless of this
// value. set_config(..., is_local=true) per transaction keeps tenant scoping correct under
// transaction pooling. Bumped 3 -> 10 to remove the worker-side serialization.
func WorkerPoolOptions() PoolOptions {
	return PoolOptions{
		MaxConns:          10,
		MinConns:          2,
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
