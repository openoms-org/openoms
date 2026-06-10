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

// newUnitService builds an enabled+wired FulfillmentService for the OPE-418 unit /
// step / blocker / aggregation recording methods, mirroring the production wiring.
func newUnitService() *service.FulfillmentService {
	return service.NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository()).
		WithRecording(appPool, repository.NewFulfillmentAttemptRepository())
}

// TestFulfillmentUnits_SingleWarehousePickPack exercises the warehouse path against
// a real DB: a warehouse unit is created once (idempotent), pick/pack steps are
// recorded and upserted (attempts increment, not duplicated), and the order's
// process aggregates to completed when the unit succeeds.
func TestFulfillmentUnits_SingleWarehousePickPack(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Warehouse Customer")

	svc := newUnitService()
	fRepo := repository.NewFulfillmentRepository()

	// EnsureUnit is idempotent: two calls yield the same unit.
	unit1 := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeWarehouse, "", map[string]any{"source": "pick_pack"})
	require.NotNil(t, unit1)
	unit2 := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeWarehouse, "", nil)
	require.NotNil(t, unit2)
	assert.Equal(t, unit1.ID, unit2.ID, "EnsureUnit must reuse the existing warehouse unit")

	// Pick step succeeded, unit running -> aggregate in_progress.
	svc.RecordStep(ctx, tenantA, unit1.ID, model.StepPickItems, model.FulfillmentStatusSucceeded, nil)
	svc.RecordUnitTransition(ctx, tenantA, unit1.ID, model.FulfillmentStatusRunning)
	proc := svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusInProgress, proc.AggregateStatus)

	// Re-recording the same step upserts (attempts increment), does not duplicate.
	svc.RecordStep(ctx, tenantA, unit1.ID, model.StepPickItems, model.FulfillmentStatusSucceeded, nil)
	// Pack step + unit succeeded -> aggregate completed.
	svc.RecordStep(ctx, tenantA, unit1.ID, model.StepPackItems, model.FulfillmentStatusSucceeded, nil)
	svc.RecordUnitTransition(ctx, tenantA, unit1.ID, model.FulfillmentStatusSucceeded)
	proc = svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusCompleted, proc.AggregateStatus)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, units, 1, "exactly one warehouse unit")
		steps, e := fRepo.ListSteps(ctx, tx, unit1.ID)
		if e != nil {
			return e
		}
		require.Len(t, steps, 2, "pick + pack steps, no duplicates")
		var pick *model.FulfillmentStep
		for i := range steps {
			if steps[i].StepKey == model.StepPickItems {
				pick = &steps[i]
			}
		}
		require.NotNil(t, pick)
		assert.Equal(t, 2, pick.Attempts, "re-recording the pick step increments attempts")
		return nil
	}))
}

// TestFulfillmentUnits_DropshipManualAndUnsupported covers the dropship paths:
// manual/portal suppliers record a waiting preflight step; an integration-linked
// supplier with no provider raises a typed capability blocker on its unit.
func TestFulfillmentUnits_DropshipManualAndUnsupported(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Dropship Customer")

	svc := newUnitService()
	fRepo := repository.NewFulfillmentRepository()

	// Manual supplier: dropship unit waiting on operator, preflight step waiting.
	manualSupplier := uuid.New().String()
	manualUnit := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeDropship, manualSupplier,
		map[string]any{"supplier_id": manualSupplier, "capability": "manual"})
	require.NotNil(t, manualUnit)
	svc.RecordStep(ctx, tenantA, manualUnit.ID, model.StepPreflightSupplierOrder, model.FulfillmentStatusWaitingExternal, nil)
	svc.RecordUnitTransition(ctx, tenantA, manualUnit.ID, model.FulfillmentStatusWaitingExternal)

	// Unsupported supplier (integration but no provider): blocked unit + blocker.
	unsupSupplier := uuid.New().String()
	unsupUnit := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeDropship, unsupSupplier,
		map[string]any{"supplier_id": unsupSupplier, "capability": "unsupported"})
	require.NotNil(t, unsupUnit)
	assert.NotEqual(t, manualUnit.ID, unsupUnit.ID, "distinct suppliers get distinct dropship units")
	svc.RecordUnitTransition(ctx, tenantA, unsupUnit.ID, model.FulfillmentStatusBlocked)
	blocker := svc.CreateSupplierBlocker(ctx, tenantA, orderID, &unsupUnit.ID,
		model.BlockerIntegrationCapabilityMissing, "no registered order provider")
	require.NotNil(t, blocker)
	assert.Equal(t, model.BlockerIntegrationCapabilityMissing, blocker.Code)
	require.NotNil(t, blocker.UnitID)
	assert.Equal(t, unsupUnit.ID, *blocker.UnitID)

	// Aggregate: a blocked unit dominates -> blocked / action_required.
	proc := svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusBlocked, proc.AggregateStatus)
	assert.Equal(t, model.ProcessHealthActionRequired, proc.HealthStatus)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Len(t, units, 2, "two dropship units, one per supplier")
		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, blockers, 1, "exactly one capability blocker")
		assert.Equal(t, "capability", blockers[0].Category)
		return nil
	}))
}

// TestFulfillmentUnits_BackorderResolveOnReceipt covers the backorder path: a
// backorder unit waits, then the goods-receipt resolves it to succeeded.
func TestFulfillmentUnits_BackorderResolveOnReceipt(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, _ := seedFulfillmentOrder(t, ctx, tenantA, "Backorder Customer")

	svc := newUnitService()

	bo := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeBackorder, "", map[string]any{"source": "stock_low"})
	require.NotNil(t, bo)
	svc.RecordStep(ctx, tenantA, bo.ID, model.StepCreatePurchaseOrder, model.FulfillmentStatusSucceeded, nil)
	svc.RecordUnitTransition(ctx, tenantA, bo.ID, model.FulfillmentStatusWaitingExternal)

	proc := svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusWaitingExternal, proc.AggregateStatus)

	// Receipt arrives — resolve the backorder unit.
	svc.RecordStep(ctx, tenantA, bo.ID, model.StepReceivePurchaseOrder, model.FulfillmentStatusSucceeded, nil)
	svc.RecordUnitTransition(ctx, tenantA, bo.ID, model.FulfillmentStatusSucceeded)
	proc = svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusCompleted, proc.AggregateStatus)
}

// TestFulfillmentUnits_MixedOrderAggregation covers a mixed order: a warehouse unit
// that succeeds alongside a dropship unit still waiting on the supplier aggregates
// to waiting_external (not completed) until the dropship unit resolves.
func TestFulfillmentUnits_MixedOrderAggregation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Mixed Customer")

	svc := newUnitService()
	fRepo := repository.NewFulfillmentRepository()

	wh := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeWarehouse, "", nil)
	require.NotNil(t, wh)
	ds := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeDropship, uuid.New().String(), nil)
	require.NotNil(t, ds)

	// Warehouse done, dropship still waiting on supplier.
	svc.RecordUnitTransition(ctx, tenantA, wh.ID, model.FulfillmentStatusSucceeded)
	svc.RecordUnitTransition(ctx, tenantA, ds.ID, model.FulfillmentStatusWaitingExternal)
	proc := svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusWaitingExternal, proc.AggregateStatus, "mixed order not complete while dropship waits")
	assert.Equal(t, model.ProcessHealthWarning, proc.HealthStatus)

	// Supplier ships — dropship succeeds; whole order completes.
	svc.RecordUnitTransition(ctx, tenantA, ds.ID, model.FulfillmentStatusSucceeded)
	proc = svc.AggregateMixedOrder(ctx, tenantA, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusCompleted, proc.AggregateStatus)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Len(t, units, 2, "one warehouse + one dropship unit")
		return nil
	}))
}

// TestFulfillmentUnits_RLSIsolation verifies units/steps/blockers are tenant-scoped:
// tenant B cannot see tenant A's units, and the RLS WITH CHECK rejects a
// cross-tenant unit insert.
func TestFulfillmentUnits_RLSIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "RLS Customer")

	svc := newUnitService()
	fRepo := repository.NewFulfillmentRepository()

	unit := svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeWarehouse, "", nil)
	require.NotNil(t, unit)
	svc.RecordStep(ctx, tenantA, unit.ID, model.StepPickItems, model.FulfillmentStatusSucceeded, nil)

	// Tenant B sees nothing of tenant A's process units.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, units, "tenant B must not see tenant A's fulfillment units")
		return nil
	}))

	// RLS WITH CHECK: tenant B cannot insert a unit owned by tenant A on A's process.
	writeErr := database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		_, e := fRepo.CreateUnit(ctx, tx, model.FulfillmentUnit{
			TenantID:  tenantA, // foreign tenant
			ProcessID: processID,
			UnitType:  model.UnitTypeWarehouse,
		})
		return e
	})
	require.Error(t, writeErr, "RLS must reject inserting a unit owned by another tenant")
}

// TestFulfillmentUnits_DisabledNoOp verifies the disabled service writes no units /
// steps / blockers — full no-op when gated OFF.
func TestFulfillmentUnits_DisabledNoOp(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Disabled Units Customer")

	svc := service.NewFulfillmentService(false, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository()).
		WithRecording(appPool, repository.NewFulfillmentAttemptRepository())
	assert.False(t, svc.Enabled())

	assert.Nil(t, svc.EnsureUnit(ctx, tenantA, orderID, model.UnitTypeWarehouse, "", nil))
	svc.RecordUnitTransition(ctx, tenantA, uuid.New(), model.FulfillmentStatusRunning)
	assert.Nil(t, svc.RecordStep(ctx, tenantA, uuid.New(), model.StepPickItems, model.FulfillmentStatusSucceeded, nil))
	assert.Nil(t, svc.CreateSupplierBlocker(ctx, tenantA, orderID, nil, model.BlockerIntegrationCapabilityMissing, "x"))
	assert.Nil(t, svc.AggregateMixedOrder(ctx, tenantA, orderID))

	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, units, "disabled service records no units")
		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, blockers, "disabled service records no blockers")
		// The process aggregate is left untouched at its seeded default.
		proc, e := fRepo.GetProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Equal(t, model.ProcessStatusNew, proc.AggregateStatus, "disabled service does not re-aggregate")
		return nil
	}))
}
