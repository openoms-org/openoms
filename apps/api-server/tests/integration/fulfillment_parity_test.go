//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// seedShipmentWithStatus inserts a shipment for an order with a specific status via
// the superuser pool. Used to exercise the legacy "problem shipment" parity branch.
// (seedOrderWithStatus is shared with the backfill test.)
func seedShipmentWithStatus(t *testing.T, ctx context.Context, tenantID, orderID uuid.UUID, status string) {
	t.Helper()
	shipID := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO shipments (id, tenant_id, order_id, provider, status) VALUES ($1,$2,$3,$4,$5)`,
		shipID, tenantID, orderID, "inpost", status)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM shipments WHERE id = $1", shipID) })
}

// TestFulfillmentParity_CountsAndCoverage seeds a known mix of orders (terminal /
// non-terminal, with / without a process, on_hold, problem-shipment) for one tenant
// and asserts the parity report's population, coverage gap, coverage ratio, verdict
// and legacy problem-order count are all computed correctly against the real DB.
func TestFulfillmentParity_CountsAndCoverage(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)

	// --- Non-terminal orders WITH a process (covered) ---
	o1 := seedOrderWithStatus(t, ctx, tenant, model.OrderStatusProcessing)
	seedProcessWithStatus(t, ctx, tenant, o1, model.ProcessStatusInProgress, model.ProcessHealthOK)
	o2 := seedOrderWithStatus(t, ctx, tenant, model.OrderStatusOnHold)
	// on_hold IS a legacy problem order; give it a blocked process so it is ALSO a
	// process-backed exception.
	seedProcessWithStatus(t, ctx, tenant, o2, model.ProcessStatusBlocked, model.ProcessHealthActionRequired)

	// --- Non-terminal orders WITHOUT a process (the coverage gap) ---
	o3 := seedOrderWithStatus(t, ctx, tenant, model.OrderStatusNew)
	o4 := seedOrderWithStatus(t, ctx, tenant, model.OrderStatusReadyToShip)
	seedShipmentWithStatus(t, ctx, tenant, o4, "failed") // legacy problem via shipment
	_ = o3

	// --- Terminal orders: must be EXCLUDED from the non-terminal population entirely ---
	seedOrderWithStatus(t, ctx, tenant, model.OrderStatusCompleted)
	seedOrderWithStatus(t, ctx, tenant, model.OrderStatusCancelled)
	seedOrderWithStatus(t, ctx, tenant, model.OrderStatusRefunded)

	svc := newReadService()

	// Use a threshold of 1.0 so the partial-coverage case (2 of 4) is clearly NOT met.
	report, err := svc.ParityReport(ctx, tenant, 1.0)
	require.NoError(t, err)

	// Population: 4 non-terminal orders (o1..o4); the 3 terminal ones excluded.
	assert.Equal(t, 4, report.NonTerminalOrders, "only non-terminal orders count")
	// Two of them have a process (o1, o2); two do not (o3, o4).
	assert.Equal(t, 2, report.OrdersMissingProcess, "o3 + o4 lack a process")
	assert.Equal(t, 2, report.FulfillmentProcesses, "exactly the two seeded processes")
	assert.InDelta(t, 0.5, report.ProcessCoverage, 1e-9, "(4-2)/4 = 0.5")
	assert.Equal(t, 1.0, report.CoverageThreshold)
	assert.False(t, report.ProcessCoverageMet, "0.5 coverage is below the 1.0 threshold")

	// Legacy problem orders: o2 (on_hold) + o4 (failed shipment) = 2. Terminal orders
	// and the healthy covered order (o1) do not count.
	assert.Equal(t, 2, report.LegacyProblemOrders, "on_hold + failed-shipment order")

	// Process-backed exceptions: only o2's process is blocked; o1's is healthy
	// in_progress. So exactly 1.
	assert.Equal(t, 1, report.ProcessBackedExceptions, "the one blocked process")
}

// TestFulfillmentParity_FullCoverageMeetsGate proves that once every non-terminal
// order has a process, coverage reaches 1.0 and the default-threshold gate is met.
func TestFulfillmentParity_FullCoverageMeetsGate(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)

	for i := 0; i < 3; i++ {
		o := seedOrderWithStatus(t, ctx, tenant, model.OrderStatusProcessing)
		seedProcessWithStatus(t, ctx, tenant, o, model.ProcessStatusInProgress, model.ProcessHealthOK)
	}

	svc := newReadService()
	report, err := svc.ParityReport(ctx, tenant, 0) // 0 -> default threshold
	require.NoError(t, err)

	assert.Equal(t, 3, report.NonTerminalOrders)
	assert.Equal(t, 0, report.OrdersMissingProcess)
	assert.InDelta(t, 1.0, report.ProcessCoverage, 1e-9)
	assert.True(t, report.ProcessCoverageMet, "full coverage meets the default gate")
}

// TestFulfillmentParity_ZeroOrders covers the empty-tenant edge case: no orders ->
// coverage 1.0 (vacuously covered), gate met, all counts zero.
func TestFulfillmentParity_ZeroOrders(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)

	svc := newReadService()
	report, err := svc.ParityReport(ctx, tenant, 0)
	require.NoError(t, err)

	assert.Equal(t, 0, report.NonTerminalOrders)
	assert.Equal(t, 0, report.FulfillmentProcesses)
	assert.Equal(t, 0, report.OrdersMissingProcess)
	assert.InDelta(t, 1.0, report.ProcessCoverage, 1e-9)
	assert.True(t, report.ProcessCoverageMet, "an empty tenant is vacuously covered")
}

// TestFulfillmentParity_CrossTenantIsolation is the RLS proof: tenant A's parity
// report must NEVER count tenant B's orders, processes, missing-process gap or
// problem orders, and vice versa.
func TestFulfillmentParity_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	// Tenant A: 2 non-terminal orders, 1 covered by a process, 1 missing.
	a1 := seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusProcessing)
	seedProcessWithStatus(t, ctx, tenantA, a1, model.ProcessStatusInProgress, model.ProcessHealthOK)
	seedOrderWithStatus(t, ctx, tenantA, model.OrderStatusNew)

	// Tenant B: a DIFFERENT shape — 3 non-terminal orders, all missing a process, one
	// on_hold (a legacy problem). If RLS leaked, A's report would absorb these.
	seedOrderWithStatus(t, ctx, tenantB, model.OrderStatusNew)
	seedOrderWithStatus(t, ctx, tenantB, model.OrderStatusConfirmed)
	seedOrderWithStatus(t, ctx, tenantB, model.OrderStatusOnHold)

	svc := newReadService()

	reportA, err := svc.ParityReport(ctx, tenantA, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, reportA.NonTerminalOrders, "A counts ONLY its own 2 orders, not B's 3")
	assert.Equal(t, 1, reportA.OrdersMissingProcess, "A's single uncovered order")
	assert.Equal(t, 1, reportA.FulfillmentProcesses, "A's single process")
	assert.Equal(t, 0, reportA.LegacyProblemOrders, "A has no on_hold/problem order; B's must not leak")

	reportB, err := svc.ParityReport(ctx, tenantB, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, reportB.NonTerminalOrders, "B counts ONLY its own 3 orders, not A's 2")
	assert.Equal(t, 3, reportB.OrdersMissingProcess, "all of B's orders lack a process")
	assert.Equal(t, 0, reportB.FulfillmentProcesses, "B has no processes; A's must not leak")
	assert.Equal(t, 1, reportB.LegacyProblemOrders, "B's single on_hold order")
}
