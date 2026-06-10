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
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// seedOrderWithStatus inserts an order for a tenant via the superuser pool and
// schedules its cleanup. Returns the new order id.
func seedOrderWithStatus(t *testing.T, ctx context.Context, tenantID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO orders (id, tenant_id, customer_name, status) VALUES ($1,$2,$3,$4)`,
		id, tenantID, "Backfill Customer", status)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id) })
	return id
}

// countProcessesForOrder returns how many fulfillment processes exist for an order
// (RLS-scoped to the tenant).
func countProcessesForOrder(t *testing.T, ctx context.Context, tenantID, orderID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment_processes WHERE order_id = $1`, orderID).Scan(&n)
	}))
	return n
}

// countOrderCreatedEventsForOrder returns how many order.created outbox events exist
// for an order, matched by the deterministic idempotency key (RLS-scoped).
func countOrderCreatedEventsForOrder(t *testing.T, ctx context.Context, tenantID, orderID uuid.UUID) int {
	t.Helper()
	idem := service.EventOrderCreated + ":" + orderID.String()
	var n int
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT count(*) FROM orchestration_outbox WHERE idempotency_key = $1`, idem).Scan(&n)
	}))
	return n
}

func runBackfill(t *testing.T, ctx context.Context, svc *service.FulfillmentBackfillService, tenantID uuid.UUID, opts service.BackfillOptions) service.BackfillReport {
	t.Helper()
	var report service.BackfillReport
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		r, e := svc.BackfillActiveOrderProcesses(ctx, tx, tenantID, opts)
		report = r
		return e
	}))
	return report
}

// TestFulfillmentBackfill_WriteRun_Idempotent exercises the OPE-423a backfill against
// a real database: a write run creates exactly one process + one order.created event
// per eligible (non-terminal, process-less) order; terminal orders and orders that
// already have a process are skipped; and a RERUN is a no-op (no duplicate rows).
func TestFulfillmentBackfill_WriteRun_Idempotent(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)

	// Eligible: non-terminal, no process.
	eligible := []uuid.UUID{
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusNew),
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusProcessing),
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusShipped),
	}
	// Terminal: must be excluded.
	terminal := []uuid.UUID{
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusCompleted),
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusCancelled),
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusRefunded),
	}
	// Already-has-process: a non-terminal order whose process we create up front.
	alreadyHave := seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusConfirmed)

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()

	// Pre-create the process for alreadyHave via the live (enabled) service path so
	// the backfill sees a genuine existing process to skip.
	live := service.NewFulfillmentService(true, fRepo, orchRepo)
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, e := live.EnsureProcessForOrder(ctx, tx, tenantA, alreadyHave)
		require.NotNil(t, proc)
		return e
	}))

	svc := service.NewFulfillmentBackfillService(fRepo, orchRepo)

	// --- Write run ---
	report := runBackfill(t, ctx, svc, tenantA, service.BackfillOptions{DryRun: false})
	assert.Equal(t, len(eligible), report.Scanned, "only eligible orders are scanned (terminal + already-have excluded)")
	assert.Equal(t, len(eligible), report.NeedingProcess)
	assert.Equal(t, len(eligible), report.Created)
	assert.Equal(t, 0, report.Skipped, "already-have order is excluded by the eligibility query, not even scanned")
	assert.Equal(t, 0, report.Errors)

	// Exactly one process + one order.created event per eligible order.
	for _, id := range eligible {
		assert.Equal(t, 1, countProcessesForOrder(t, ctx, tenantA, id), "one process per eligible order")
		assert.Equal(t, 1, countOrderCreatedEventsForOrder(t, ctx, tenantA, id), "one order.created event per eligible order")
	}
	// Terminal orders are never given a process.
	for _, id := range terminal {
		assert.Equal(t, 0, countProcessesForOrder(t, ctx, tenantA, id), "terminal order gets no process")
	}
	// Already-have order keeps its single pre-existing process (no duplicate).
	assert.Equal(t, 1, countProcessesForOrder(t, ctx, tenantA, alreadyHave))
	assert.Equal(t, 1, countOrderCreatedEventsForOrder(t, ctx, tenantA, alreadyHave))

	// --- Rerun: pure no-op, no new rows ---
	rerun := runBackfill(t, ctx, svc, tenantA, service.BackfillOptions{DryRun: false})
	assert.Equal(t, 0, rerun.Scanned, "rerun finds nothing still-missing")
	assert.Equal(t, 0, rerun.NeedingProcess)
	assert.Equal(t, 0, rerun.Created)
	for _, id := range eligible {
		assert.Equal(t, 1, countProcessesForOrder(t, ctx, tenantA, id), "rerun creates no duplicate process")
		assert.Equal(t, 1, countOrderCreatedEventsForOrder(t, ctx, tenantA, id), "rerun creates no duplicate event")
	}
}

// TestFulfillmentBackfill_DryRun_NoWrites verifies a dry run counts the eligible
// orders but writes NOTHING to the database.
func TestFulfillmentBackfill_DryRun_NoWrites(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)

	eligible := []uuid.UUID{
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusNew),
		seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusReadyToShip),
	}
	// A terminal order present to confirm it is not counted either.
	seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusCancelled)

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	svc := service.NewFulfillmentBackfillService(fRepo, orchRepo)

	report := runBackfill(t, ctx, svc, tenantA, service.BackfillOptions{DryRun: true})
	assert.Equal(t, len(eligible), report.Scanned)
	assert.Equal(t, len(eligible), report.NeedingProcess, "dry run counts what a write run would create")
	assert.Equal(t, 0, report.Created)

	// No writes happened.
	for _, id := range eligible {
		assert.Equal(t, 0, countProcessesForOrder(t, ctx, tenantA, id), "dry run creates no process")
		assert.Equal(t, 0, countOrderCreatedEventsForOrder(t, ctx, tenantA, id), "dry run enqueues no event")
	}
}

// TestFulfillmentBackfill_TenantIsolation verifies the backfill is RLS-isolated:
// a write run for tenant A never touches tenant B's process-less orders, and vice
// versa. Each tenant's pass is scoped to its own orders.
func TestFulfillmentBackfill_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderA := seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusNew)
	orderB := seedOrderWithStatus(t, ctx, tenantB, model.OrderStatusNew)

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	svc := service.NewFulfillmentBackfillService(fRepo, orchRepo)

	// Backfill tenant A only.
	reportA := runBackfill(t, ctx, svc, tenantA, service.BackfillOptions{DryRun: false})
	assert.Equal(t, 1, reportA.Scanned, "tenant A pass sees only tenant A's order")
	assert.Equal(t, 1, reportA.Created)

	assert.Equal(t, 1, countProcessesForOrder(t, ctx, tenantA, orderA), "tenant A order backfilled")
	assert.Equal(t, 0, countProcessesForOrder(t, ctx, tenantB, orderB), "tenant B order untouched by tenant A pass")

	// Now backfill tenant B: it still sees its order as missing (isolation held).
	reportB := runBackfill(t, ctx, svc, tenantB, service.BackfillOptions{DryRun: false})
	assert.Equal(t, 1, reportB.Scanned, "tenant B pass sees only tenant B's still-missing order")
	assert.Equal(t, 1, reportB.Created)
	assert.Equal(t, 1, countProcessesForOrder(t, ctx, tenantB, orderB))
}
