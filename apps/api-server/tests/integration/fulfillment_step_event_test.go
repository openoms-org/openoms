//go:build integration

package integration

import (
	"context"
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

// TestFulfillmentStepEvent_WorkerAcksWithoutBlocker covers OPE-513 end to end:
// a fulfillment.step event emitted on a SUCCESSFUL provider operation (via the
// real recording service through the transactional outbox) must be drained to
// "succeeded" by the orchestration worker — with FulfillmentStepHandler
// registered exactly as in production wiring — WITHOUT opening a fulfillment
// blocker. Before the fix the dispatcher had no handler for the event type, so
// every healthy step event failed permanently and opened a spurious blocker.
func TestFulfillmentStepEvent_WorkerAcksWithoutBlocker(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderID, procID := seedFulfillmentOrder(t, ctx, tenant, "Step Ack Customer")

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()

	// Production enqueue path: the enabled recording service emits the step event
	// in its own tenant transaction (best-effort, exactly like the shipment/label/
	// tracking success paths do).
	svc := service.NewFulfillmentService(true, fRepo, orchRepo).
		WithRecording(appPool, repository.NewFulfillmentAttemptRepository())
	svc.EmitFulfillmentStep(ctx, tenant, orderID,
		model.StepCreateShipment, model.FulfillmentStatusSucceeded, "ship-513",
		map[string]any{"provider": "inpost"})

	// The fulfillment.step event is pending on the process before the worker runs.
	var eventID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1, "EmitFulfillmentStep enqueued exactly one event")
		assert.Equal(t, service.EventFulfillmentStep, events[0].EventType)
		assert.Equal(t, model.OutboxStatusPending, events[0].Status)
		eventID = events[0].ID
		return nil
	}))

	// Run the REAL worker on the privileged pool with the production dispatcher
	// wiring for this event type (cmd/server/main.go registers the handler
	// unconditionally inside the orchestration-worker block).
	disp := service.NewOrchestrationDispatcher()
	disp.Register(service.EventFulfillmentStep, service.NewFulfillmentStepHandler())
	w := worker.NewOrchestrationWorker(superPool, orchRepo, disp, fRepo, time.Second, slog.Default())
	require.NoError(t, w.Run(ctx))

	// Outcome: the event is acked (succeeded), NOT failed.
	ev, err := orchRepo.GetEvent(ctx, superPool, eventID)
	require.NoError(t, err)
	assert.Equal(t, model.OutboxStatusSucceeded, ev.Status, "fulfillment.step must be acked by the worker")

	// And NO spurious fulfillment blocker was opened on the healthy process.
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		blockers, e := fRepo.ListBlockers(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Empty(t, blockers, "no blocker may be opened for an acked fulfillment.step event")
		return nil
	}))
}
