package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenant acquires a connection, sets the tenant context using a
// PARAMETERIZED query (never fmt.Sprintf), executes the callback, and commits.
// This is the ONLY way application code should run tenant-scoped queries.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// CRITICAL: Parameterized query — prevents SQL injection
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)",
		tenantID.String(),
	); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
