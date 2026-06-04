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
)

// TestFulfillment_CRUDAndTenantIsolation exercises the fulfillment model
// foundation against a real database: a full create/read/update round-trip
// under tenant A, and RLS isolation proving tenant B cannot see or write A's
// fulfillment data.
func TestFulfillment_CRUDAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1, $2, $3)`,
		orderID, tenantA, "Fulfillment Customer")
	require.NoError(t, err)

	repo := repository.NewFulfillmentRepository()

	// --- Tenant A: full CRUD round-trip ---
	var procID, unitID, stepID, blockerID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, err := repo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantA, OrderID: orderID})
		if err != nil {
			return err
		}
		assert.Equal(t, model.ProcessStatusNew, proc.AggregateStatus)
		assert.Equal(t, model.ProcessHealthOK, proc.HealthStatus)
		procID = proc.ID

		unit, err := repo.CreateUnit(ctx, tx, model.FulfillmentUnit{TenantID: tenantA, ProcessID: proc.ID, UnitType: model.UnitTypeWarehouse})
		if err != nil {
			return err
		}
		unitID = unit.ID

		step, err := repo.CreateStep(ctx, tx, model.FulfillmentStep{TenantID: tenantA, UnitID: unit.ID, StepKey: model.StepValidateOrder})
		if err != nil {
			return err
		}
		stepID = step.ID

		blocker, err := repo.CreateBlocker(ctx, tx, model.FulfillmentBlocker{TenantID: tenantA, ProcessID: proc.ID, Code: model.BlockerStockSyncFailed})
		if err != nil {
			return err
		}
		assert.Equal(t, "integration", blocker.Category, "category is derived from the code")
		blockerID = blocker.ID
		return nil
	}))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		got, err := repo.GetProcessByOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		assert.Equal(t, procID, got.ID)

		units, err := repo.ListUnits(ctx, tx, procID)
		if err != nil {
			return err
		}
		require.Len(t, units, 1)
		steps, err := repo.ListSteps(ctx, tx, unitID)
		if err != nil {
			return err
		}
		require.Len(t, steps, 1)
		blockers, err := repo.ListBlockers(ctx, tx, procID)
		if err != nil {
			return err
		}
		require.Len(t, blockers, 1)

		// Updates.
		p, err := repo.UpdateProcessStatus(ctx, tx, procID, model.ProcessStatusReady, model.ProcessHealthWarning)
		if err != nil {
			return err
		}
		assert.Equal(t, model.ProcessStatusReady, p.AggregateStatus)
		if _, err := repo.UpdateUnitStatus(ctx, tx, unitID, model.FulfillmentStatusRunning); err != nil {
			return err
		}
		if _, err := repo.UpdateStepStatus(ctx, tx, stepID, model.FulfillmentStatusSucceeded, 1); err != nil {
			return err
		}
		rb, err := repo.ResolveBlocker(ctx, tx, blockerID, model.BlockerStatusResolved)
		if err != nil {
			return err
		}
		assert.Equal(t, model.BlockerStatusResolved, rb.Status)
		require.NotNil(t, rb.ResolvedAt)
		return nil
	}))

	// --- Tenant B: RLS isolation (cannot read A's process) ---
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		_, err := repo.GetProcessByOrder(ctx, tx, orderID)
		assert.ErrorIs(t, err, pgx.ErrNoRows, "tenant B must not see tenant A's process by order")
		_, err = repo.GetProcess(ctx, tx, procID)
		assert.ErrorIs(t, err, pgx.ErrNoRows, "tenant B must not see tenant A's process by id")
		procs, err := repo.ListProcesses(ctx, tx)
		if err != nil {
			return err
		}
		assert.Empty(t, procs, "tenant B sees none of tenant A's processes")
		return nil
	}))

	// --- Tenant B: RLS WITH CHECK blocks writing a row for tenant A ---
	writeErr := database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		_, e := repo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantA, OrderID: orderID})
		return e
	})
	require.Error(t, writeErr, "RLS WITH CHECK must reject inserting a row owned by another tenant")
}
