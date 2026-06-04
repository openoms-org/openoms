//go:build integration

package integration

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
)

type fakeOrchHandler func(model.OrchestrationOutboxEvent) error

func (f fakeOrchHandler) Handle(_ context.Context, e model.OrchestrationOutboxEvent) error {
	return f(e)
}

// TestOrchestration_OutboxWorkerLifecycle exercises the OPE-415 outbox + worker
// end to end against a real database: idempotent enqueue, cross-tenant claim,
// and success / retry / permanent / no-handler outcomes (the last two opening
// fulfillment blockers).
func TestOrchestration_OutboxWorkerLifecycle(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)

	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,$3)`, orderID, tenantA, "Orch Customer")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID) })

	orchRepo := repository.NewOrchestrationRepository()
	fRepo := repository.NewFulfillmentRepository()

	// A fulfillment process to attach the outbox events to.
	var procID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		p, e := fRepo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantA, OrderID: orderID})
		if e == nil {
			procID = p.ID
		}
		return e
	}))

	enqueue := func(eventType, idem string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
			created, e, err := orchRepo.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
				TenantID: tenantA, ProcessID: procID, EventType: eventType, IdempotencyKey: idem,
			})
			if err != nil {
				return err
			}
			assert.True(t, created, "first enqueue creates the event")
			id = e.ID
			return nil
		}))
		return id
	}
	successID := enqueue("test.success", "idem-success")
	retryID := enqueue("test.retry", "idem-retry")
	permID := enqueue("test.permanent", "idem-perm")
	noHandlerID := enqueue("test.unknown", "idem-unknown")

	// Idempotent: re-enqueueing the same key does not create a second row.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		created, _, err := orchRepo.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
			TenantID: tenantA, ProcessID: procID, EventType: "test.success", IdempotencyKey: "idem-success",
		})
		assert.False(t, created, "duplicate idempotency key must not create a second row")
		return err
	}))

	// Dispatcher with fake handlers (test.unknown is intentionally unregistered).
	disp := service.NewOrchestrationDispatcher()
	disp.Register("test.success", fakeOrchHandler(func(model.OrchestrationOutboxEvent) error { return nil }))
	disp.Register("test.retry", fakeOrchHandler(func(model.OrchestrationOutboxEvent) error { return errors.New("transient") }))
	disp.Register("test.permanent", fakeOrchHandler(func(model.OrchestrationOutboxEvent) error { return model.Permanent(errors.New("unrecoverable")) }))

	// Worker uses superPool (bypasses RLS) like the production worker pool.
	w := worker.NewOrchestrationWorker(superPool, orchRepo, disp, fRepo, time.Second, slog.Default())
	before := time.Now().UTC()
	require.NoError(t, w.Run(ctx))

	get := func(id uuid.UUID) *model.OrchestrationOutboxEvent {
		e, err := orchRepo.GetEvent(ctx, superPool, id)
		require.NoError(t, err)
		return e
	}

	assert.Equal(t, model.OutboxStatusSucceeded, get(successID).Status)

	retried := get(retryID)
	assert.Equal(t, model.OutboxStatusPending, retried.Status, "retryable failure re-queues")
	assert.Equal(t, 1, retried.Attempts)
	assert.True(t, retried.NextAttemptAt.After(before), "retry backs off into the future")

	assert.Equal(t, model.OutboxStatusFailed, get(permID).Status, "permanent failure -> failed")
	assert.Equal(t, model.OutboxStatusFailed, get(noHandlerID).Status, "no handler -> permanent failure")

	// Permanent + no-handler each open a fulfillment blocker on the process.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		blockers, err := fRepo.ListBlockers(ctx, tx, procID)
		if err != nil {
			return err
		}
		assert.Len(t, blockers, 2, "permanent and no-handler events each open a blocker")
		return nil
	}))
}
