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

// seedFulfillmentOrder seeds an order + its fulfillment process for a tenant and
// returns (orderID, processID). The process is what the OPE-417 recording methods
// correlate provider attempts/steps/blockers to.
func seedFulfillmentOrder(t *testing.T, ctx context.Context, tenantID uuid.UUID, customer string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,$3)`,
		orderID, tenantID, customer)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID) })

	fRepo := repository.NewFulfillmentRepository()
	var processID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		proc, e := fRepo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantID, OrderID: orderID})
		if e != nil {
			return e
		}
		processID = proc.ID
		return nil
	}))
	return orderID, processID
}

// TestFulfillmentProviderAttempts_RecordingAndIsolation exercises the OPE-417
// provider-attempt table against a real database: best-effort recording correlates
// to the fulfillment process, emits a step event, and RLS isolates the rows per
// tenant.
func TestFulfillmentProviderAttempts_RecordingAndIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Attempt Customer")

	attemptRepo := repository.NewFulfillmentAttemptRepository()
	svc := service.NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository()).
		WithRecording(appPool, attemptRepo)
	require.True(t, svc.Enabled())

	// --- Record a successful create_shipment attempt (best-effort) ---
	correlation := uuid.New().String()
	attempt := svc.RecordProviderAttempt(ctx, service.ProviderAttemptInput{
		TenantID:       tenantA,
		OrderID:        orderID,
		Provider:       "inpost",
		Operation:      model.ProviderOpCreateShipment,
		Status:         model.ProviderAttemptSucceeded,
		CorrelationID:  correlation,
		RequestID:      "EXT-99",
		ResultRedacted: map[string]any{"has_tracking": true},
	})
	require.NotNil(t, attempt, "enabled+wired recording returns the persisted attempt")
	assert.Equal(t, model.ProviderAttemptSucceeded, attempt.Status)
	require.NotNil(t, attempt.ProcessID, "attempt is correlated to the fulfillment process")
	assert.Equal(t, processID, *attempt.ProcessID)

	// The attempt is visible under tenant A and attached to the process.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		got, e := attemptRepo.ListByProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, got, 1)
		assert.Equal(t, "inpost", got[0].Provider)
		assert.Equal(t, model.ProviderOpCreateShipment, got[0].Operation)
		assert.Equal(t, "EXT-99", got[0].RequestID)

		byCorr, e := attemptRepo.ListByCorrelation(ctx, tx, correlation, "inpost", model.ProviderOpCreateShipment)
		if e != nil {
			return e
		}
		assert.Len(t, byCorr, 1, "attempt is also retrievable by correlation id")
		return nil
	}))

	// --- Emit a fulfillment step: an outbox event lands on the process ---
	orchRepo := repository.NewOrchestrationRepository()
	svc.EmitFulfillmentStep(ctx, tenantA, orderID, model.StepCreateShipment,
		model.FulfillmentStatusSucceeded, correlation, map[string]any{"provider": "inpost"})
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1, "exactly one fulfillment.step event emitted")
		assert.Equal(t, service.EventFulfillmentStep, events[0].EventType)
		return nil
	}))

	// --- A failed tracking sync creates a typed blocker (best-effort) ---
	svc.RecordTrackingSyncFailure(ctx, tenantA, orderID, "inpost", correlation,
		assertErr("503 service unavailable"))
	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, blockers, 1, "carrier outage creates exactly one typed blocker")
		assert.Equal(t, model.BlockerIntegrationCapabilityDegraded, blockers[0].Code)
		assert.True(t, model.IsValidBlockerCode(blockers[0].Code))
		return nil
	}))

	// --- RLS: tenant B sees none of tenant A's attempts ---
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		got, e := attemptRepo.ListByProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, got, "tenant B must not see tenant A's provider attempts")
		byCorr, e := attemptRepo.ListByCorrelation(ctx, tx, correlation, "inpost", model.ProviderOpCreateShipment)
		if e != nil {
			return e
		}
		assert.Empty(t, byCorr, "tenant B must not see tenant A's attempts by correlation")
		return nil
	}))

	// --- RLS WITH CHECK: tenant B cannot insert an attempt owned by tenant A ---
	writeErr := database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		_, e := attemptRepo.RecordAttempt(ctx, tx, model.ProviderAttempt{
			TenantID:  tenantA, // foreign tenant
			Provider:  "inpost",
			Operation: model.ProviderOpCreateShipment,
			Status:    model.ProviderAttemptSucceeded,
		})
		return e
	})
	require.Error(t, writeErr, "RLS must reject inserting an attempt owned by another tenant")
}

// TestFulfillmentProviderAttempts_TrackingRawPreserved verifies an unmapped carrier
// status is recorded as an unsupported-capability outcome — raw preserved, no
// blocker, no step.
func TestFulfillmentProviderAttempts_TrackingRawPreserved(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Tracking Customer")

	attemptRepo := repository.NewFulfillmentAttemptRepository()
	orchRepo := repository.NewOrchestrationRepository()
	fRepo := repository.NewFulfillmentRepository()
	svc := service.NewFulfillmentService(true, fRepo, orchRepo).WithRecording(appPool, attemptRepo)

	// Unmapped provider status: raw preserved, recorded as succeeded outcome.
	svc.RecordTrackingSyncAttempt(ctx, tenantA, orderID, "dhl", uuid.New().String(),
		model.NewTrackingStatusMapping("EXOTIC_CARRIER_STATE", "", false), model.ProviderAttemptSucceeded)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		got, e := attemptRepo.ListByProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, got, 1)
		assert.Equal(t, model.ProviderOpSyncTracking, got[0].Operation)
		assert.Equal(t, "EXOTIC_CARRIER_STATE", got[0].RawProviderStatus, "raw provider status must be preserved")
		assert.Equal(t, model.ProviderAttemptSucceeded, got[0].Status, "unmapped status is an outcome, not a failure")

		// No blocker and no step event for an unsupported-capability outcome.
		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, blockers, "unmapped status must not create a blocker")
		events, e := orchRepo.ListByProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, events, "unmapped status must not emit a step event")
		return nil
	}))
}

// TestFulfillmentProviderAttempts_DisabledNoOp verifies the disabled service writes
// nothing to the provider-attempts table — full no-op when gated OFF.
func TestFulfillmentProviderAttempts_DisabledNoOp(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Disabled Attempt Customer")

	attemptRepo := repository.NewFulfillmentAttemptRepository()
	svc := service.NewFulfillmentService(false, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository()).
		WithRecording(appPool, attemptRepo)
	assert.False(t, svc.Enabled())

	// Disabled: every recording call is a no-op.
	assert.Nil(t, svc.RecordProviderAttempt(ctx, service.ProviderAttemptInput{
		TenantID: tenantA, OrderID: orderID, Provider: "inpost", Operation: model.ProviderOpCreateShipment,
		Status: model.ProviderAttemptSucceeded,
	}))
	svc.EmitFulfillmentStep(ctx, tenantA, orderID, model.StepCreateShipment, model.FulfillmentStatusSucceeded, "c", nil)
	svc.RecordTrackingSyncFailure(ctx, tenantA, orderID, "inpost", "c", assertErr("boom"))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		got, e := attemptRepo.ListByProcess(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, got, "disabled service records no provider attempts")
		return nil
	}))
}

// assertErr is a tiny helper to build an error from a message in tests.
func assertErr(msg string) error { return &simpleErr{msg} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }
