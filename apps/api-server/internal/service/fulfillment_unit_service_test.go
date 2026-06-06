package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// TestClassifySupplierCapability covers the pure supplier-capability derivation
// used to decide dropship automation vs manual/portal fallback vs unsupported.
func TestClassifySupplierCapability(t *testing.T) {
	intID := uuid.New()

	// Integration linked + a registered provider -> API.
	assert.Equal(t, SupplierCapAPI,
		ClassifySupplierCapability(&model.Supplier{IntegrationID: &intID}, true))

	// Integration linked but NO provider for its format -> Unsupported (blocker case).
	assert.Equal(t, SupplierCapUnsupported,
		ClassifySupplierCapability(&model.Supplier{IntegrationID: &intID}, false))

	// No integration, portal enabled -> Portal (manual handoff via portal).
	assert.Equal(t, SupplierCapPortal,
		ClassifySupplierCapability(&model.Supplier{PortalEnabled: true}, false))

	// No integration, no portal -> Manual (out-of-band).
	assert.Equal(t, SupplierCapManual,
		ClassifySupplierCapability(&model.Supplier{}, false))

	// providerRegistered is ignored without an integration.
	assert.Equal(t, SupplierCapManual,
		ClassifySupplierCapability(&model.Supplier{}, true))

	// Nil supplier defaults to manual (best-effort caller path).
	assert.Equal(t, SupplierCapManual, ClassifySupplierCapability(nil, true))
}

// TestAggregateProcessState covers the pure mixed-order aggregation precedence.
func TestAggregateProcessState(t *testing.T) {
	unit := func(status string) model.FulfillmentUnit {
		return model.FulfillmentUnit{Status: status}
	}

	tests := []struct {
		name          string
		units         []model.FulfillmentUnit
		wantAggregate string
		wantHealth    string
	}{
		{
			name:          "no units is new/ok",
			units:         nil,
			wantAggregate: model.ProcessStatusNew,
			wantHealth:    model.ProcessHealthOK,
		},
		{
			name:          "blocked dominates everything",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusSucceeded), unit(model.FulfillmentStatusBlocked), unit(model.FulfillmentStatusRunning)},
			wantAggregate: model.ProcessStatusBlocked,
			wantHealth:    model.ProcessHealthActionRequired,
		},
		{
			name:          "failed (no blocked) is in_progress/system_error",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusFailed), unit(model.FulfillmentStatusRunning)},
			wantAggregate: model.ProcessStatusInProgress,
			wantHealth:    model.ProcessHealthSystemError,
		},
		{
			name:          "waiting_external (no blocked/failed) is waiting/warning",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusWaitingExternal), unit(model.FulfillmentStatusSucceeded)},
			wantAggregate: model.ProcessStatusWaitingExternal,
			wantHealth:    model.ProcessHealthWarning,
		},
		{
			name:          "running (no blocked/failed/waiting) is in_progress/ok",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusRunning), unit(model.FulfillmentStatusReady)},
			wantAggregate: model.ProcessStatusInProgress,
			wantHealth:    model.ProcessHealthOK,
		},
		{
			name:          "all terminal with a survivor is completed",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusSucceeded), unit(model.FulfillmentStatusSkipped), unit(model.FulfillmentStatusCancelled)},
			wantAggregate: model.ProcessStatusCompleted,
			wantHealth:    model.ProcessHealthOK,
		},
		{
			name:          "all units cancelled is cancelled (not completed)",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusCancelled), unit(model.FulfillmentStatusCancelled)},
			wantAggregate: model.ProcessStatusCancelled,
			wantHealth:    model.ProcessHealthOK,
		},
		{
			name:          "all pending/ready is ready",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusPending), unit(model.FulfillmentStatusReady)},
			wantAggregate: model.ProcessStatusReady,
			wantHealth:    model.ProcessHealthOK,
		},
		{
			name:          "mixed warehouse succeeded + dropship waiting is waiting/warning",
			units:         []model.FulfillmentUnit{unit(model.FulfillmentStatusSucceeded), unit(model.FulfillmentStatusWaitingExternal)},
			wantAggregate: model.ProcessStatusWaitingExternal,
			wantHealth:    model.ProcessHealthWarning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAgg, gotHealth := AggregateProcessState(tt.units)
			assert.Equal(t, tt.wantAggregate, gotAgg)
			assert.Equal(t, tt.wantHealth, gotHealth)
		})
	}
}

// TestFulfillmentUnits_DisabledNoOp verifies all OPE-418 unit/step/blocker methods
// are complete no-ops when the service is disabled — they neither touch the (nil)
// pool nor return a recorded entity, so wiring them in is zero behavior change.
func TestFulfillmentUnits_DisabledNoOp(t *testing.T) {
	svc := NewFulfillmentService(false, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	assert.False(t, svc.Enabled())

	ctx := context.Background()
	tenantID, orderID, unitID := uuid.New(), uuid.New(), uuid.New()

	assert.Nil(t, svc.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeWarehouse, "", nil))
	assert.Nil(t, svc.RecordStep(ctx, tenantID, unitID, model.StepPickItems, model.FulfillmentStatusSucceeded, nil))
	assert.Nil(t, svc.CreateSupplierBlocker(ctx, tenantID, orderID, &unitID, model.BlockerIntegrationCapabilityMissing, "x"))
	assert.Nil(t, svc.AggregateMixedOrder(ctx, tenantID, orderID))
	// Void methods must not panic with a nil pool when disabled.
	svc.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusRunning)
}

// TestFulfillmentUnits_EnabledButUnwiredNoOp verifies that an enabled service with
// no recording pool wired (recordingReady() == false) is still a no-op — it never
// dereferences the nil pool.
func TestFulfillmentUnits_EnabledButUnwiredNoOp(t *testing.T) {
	svc := NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	assert.True(t, svc.Enabled())

	ctx := context.Background()
	tenantID, orderID, unitID := uuid.New(), uuid.New(), uuid.New()

	assert.Nil(t, svc.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeDropship, "k", nil))
	assert.Nil(t, svc.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusReady, nil))
	assert.Nil(t, svc.CreateSupplierBlocker(ctx, tenantID, orderID, nil, model.BlockerSupplierAvailabilityUnknown, "x"))
	assert.Nil(t, svc.AggregateMixedOrder(ctx, tenantID, orderID))
	svc.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusRunning)
}

// TestFulfillmentUnits_RecordSkipsInvalidEnums verifies invalid unit types / step
// keys / statuses / blocker codes are skipped rather than panicking or persisting.
func TestFulfillmentUnits_RecordSkipsInvalidEnums(t *testing.T) {
	svc := NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	ctx := context.Background()
	tenantID, orderID, unitID := uuid.New(), uuid.New(), uuid.New()

	// Even if recording were wired, these would short-circuit on the invalid enum
	// before any DB access; here they short-circuit on recordingReady first. Either
	// way the contract is: no panic, nil result.
	assert.Nil(t, svc.EnsureUnit(ctx, tenantID, orderID, "not_a_unit_type", "", nil))
	assert.Nil(t, svc.RecordStep(ctx, tenantID, unitID, "not_a_step", model.FulfillmentStatusReady, nil))
	assert.Nil(t, svc.CreateSupplierBlocker(ctx, tenantID, orderID, nil, "not_a_blocker_code", "x"))
	// uuid.Nil unit id is a guarded no-op.
	svc.RecordUnitTransition(ctx, tenantID, uuid.Nil, model.FulfillmentStatusRunning)
}

// TestCloneMeta verifies cloneMeta copies and never aliases the caller's map.
func TestCloneMeta(t *testing.T) {
	src := map[string]any{"a": 1}
	out := cloneMeta(src)
	out["b"] = 2
	assert.Equal(t, 1, src["a"])
	_, leaked := src["b"]
	assert.False(t, leaked, "cloneMeta must not mutate the source map")

	assert.NotNil(t, cloneMeta(nil), "cloneMeta(nil) returns a usable empty map")
}

// TestUnitMetaKey verifies the dedupe-key extraction round-trips through the JSONB
// any decoding shape (map[string]any).
func TestUnitMetaKey(t *testing.T) {
	assert.Equal(t, "supplier-1", unitMetaKey(map[string]any{"key": "supplier-1"}))
	assert.Equal(t, "", unitMetaKey(map[string]any{"other": "x"}))
	assert.Equal(t, "", unitMetaKey(nil))
	assert.Equal(t, "", unitMetaKey("not-a-map"))
}
