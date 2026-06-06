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

// TestOrderFulfillment_EnsureProcessForOrder exercises the OPE-416 fulfillment
// command bridge against a real database: EnsureProcessForOrder creates the
// fulfillment process AND enqueues an order.created orchestration event in one
// tenant transaction, is idempotent on re-invocation, and the OrderCreatedHandler
// advances the process from new -> validating.
func TestOrderFulfillment_EnsureProcessForOrder(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)

	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,$3)`,
		orderID, tenantA, "Fulfillment Order Customer")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID) })

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	// Enabled service — this is the on path the production flag gates.
	svc := service.NewFulfillmentService(true, fRepo, orchRepo)

	// --- First invocation: creates the process + enqueues the event ---
	var procID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, e := svc.EnsureProcessForOrder(ctx, tx, tenantA, orderID)
		if e != nil {
			return e
		}
		require.NotNil(t, proc)
		assert.Equal(t, orderID, proc.OrderID)
		assert.Equal(t, model.ProcessStatusNew, proc.AggregateStatus)
		assert.Equal(t, model.ProcessHealthOK, proc.HealthStatus)
		procID = proc.ID
		return nil
	}))

	// The outbox event was enqueued in the same tenant context, attached to the process.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1, "exactly one order.created event is enqueued")
		assert.Equal(t, service.EventOrderCreated, events[0].EventType)
		assert.Equal(t, model.OutboxStatusPending, events[0].Status)
		assert.Equal(t, service.EventOrderCreated+":"+orderID.String(), events[0].IdempotencyKey)
		return nil
	}))

	// --- Second invocation: idempotent — same process, no second event ---
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, e := svc.EnsureProcessForOrder(ctx, tx, tenantA, orderID)
		if e != nil {
			return e
		}
		require.NotNil(t, proc)
		assert.Equal(t, procID, proc.ID, "re-invocation returns the existing process")
		return nil
	}))
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Len(t, events, 1, "idempotent: no duplicate order.created event")
		return nil
	}))

	// --- OrderCreatedHandler advances the process new -> validating ---
	handler := service.NewOrderCreatedHandler(superPool, fRepo)
	var enqueued model.OrchestrationOutboxEvent
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		enqueued = events[0]
		return nil
	}))
	require.NoError(t, handler.Handle(ctx, enqueued))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, e := fRepo.GetProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Equal(t, model.ProcessStatusValidating, proc.AggregateStatus, "handler transitions new -> validating")
		assert.Equal(t, model.ProcessHealthOK, proc.HealthStatus)
		return nil
	}))

	// Handler is idempotent on re-delivery: a non-new process is left untouched.
	require.NoError(t, handler.Handle(ctx, enqueued))
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, e := fRepo.GetProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Equal(t, model.ProcessStatusValidating, proc.AggregateStatus)
		return nil
	}))
}

// TestOrderFulfillment_Disabled_NoOp verifies the disabled service writes nothing
// to the database — no process, no outbox event.
func TestOrderFulfillment_Disabled_NoOp(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)

	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,$3)`,
		orderID, tenantA, "Disabled Fulfillment Customer")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID) })

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	svc := service.NewFulfillmentService(false, fRepo, orchRepo)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		proc, e := svc.EnsureProcessForOrder(ctx, tx, tenantA, orderID)
		if e != nil {
			return e
		}
		assert.Nil(t, proc, "disabled service is a no-op")
		return nil
	}))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		_, e := fRepo.GetProcessByOrder(ctx, tx, orderID)
		assert.ErrorIs(t, e, pgx.ErrNoRows, "disabled service creates no process")
		return nil
	}))
}
