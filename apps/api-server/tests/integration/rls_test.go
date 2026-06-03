//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
)

// TestRLS_CustomersCrossTenantIsolation proves the customers RLS policy isolates
// tenants for the non-superuser openoms_app role: a customer created for tenant A
// is invisible to tenant B's context, visible to tenant A's context, and invisible
// when no tenant context is set (FORCE ROW LEVEL SECURITY + the tenant_id policy).
func TestRLS_CustomersCrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	custA := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO customers (id, tenant_id, name, email) VALUES ($1, $2, $3, $4)`,
		custA, tenantA, "Customer A", "a@example.com")
	require.NoError(t, err, "superuser seed of tenant A customer")
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customers WHERE id = $1", custA) })

	countAs := func(t *testing.T, tenant uuid.UUID) int {
		t.Helper()
		var n int
		err := database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM customers WHERE id = $1", custA).Scan(&n)
		})
		require.NoError(t, err)
		return n
	}

	assert.Equal(t, 0, countAs(t, tenantB), "tenant B must NOT see tenant A's customer (RLS isolation)")
	assert.Equal(t, 1, countAs(t, tenantA), "tenant A must see its own customer")

	// No tenant context (a raw pool query, which the app never does — it always uses
	// WithTenant). The custom GUC app.current_tenant_id is either unset (NULL -> the
	// policy predicate is NULL -> 0 rows) or, on a pooled connection that previously ran
	// WithTenant, PostgreSQL's placeholder default '' -> the policy's ''::uuid cast errors.
	// Either way the row is NOT revealed — both outcomes are safe (no cross-tenant leak);
	// it must never return the row.
	var nNoCtx int
	err = appPool.QueryRow(ctx, "SELECT count(*) FROM customers WHERE id = $1", custA).Scan(&nNoCtx)
	if err == nil {
		assert.Equal(t, 0, nNoCtx, "with no tenant context the app role must not reveal the row (FORCE RLS)")
	}
}

// TestRLS_CustomersWriteIsolation proves the policy also blocks cross-tenant writes:
// under tenant B's context the app role cannot INSERT a row carrying tenant A's id.
func TestRLS_CustomersWriteIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	rogue := uuid.New()
	err := database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO customers (id, tenant_id, name, email) VALUES ($1, $2, $3, $4)`,
			rogue, tenantA, "Rogue", "rogue@example.com")
		return e
	})
	require.Error(t, err, "RLS WITH CHECK must reject inserting a row for another tenant")
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customers WHERE id = $1", rogue) })

	// And confirm nothing was actually written (superuser bypasses RLS to verify).
	var n int
	require.NoError(t, superPool.QueryRow(ctx, "SELECT count(*) FROM customers WHERE id = $1", rogue).Scan(&n))
	assert.Equal(t, 0, n, "no rogue cross-tenant row should exist")
}
