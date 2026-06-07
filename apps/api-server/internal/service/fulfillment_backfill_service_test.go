package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// stubBackfillRepo is an in-memory stand-in for both the process and event repos
// used by FulfillmentBackfillService. It records what would be written so unit
// tests can assert dry-run does no writes and write-run mirrors EnsureProcessForOrder.
type stubBackfillRepo struct {
	// eligible is what ListOrderIDsMissingProcess returns (the still-missing set).
	eligible []uuid.UUID
	// existing marks orders that already have a process (GetProcessByOrder hit).
	existing map[uuid.UUID]bool

	// recorded side effects.
	createdProcesses []uuid.UUID // order ids a process was created for
	enqueuedEvents   []model.OrchestrationOutboxEvent

	// lastTerminal captures the terminal-status list passed to the eligibility query.
	lastTerminal []string
}

func (s *stubBackfillRepo) ListOrderIDsMissingProcess(_ context.Context, _ pgx.Tx, terminalStatuses []string, limit int) ([]uuid.UUID, error) {
	s.lastTerminal = terminalStatuses
	if limit > 0 && len(s.eligible) > limit {
		return s.eligible[:limit], nil
	}
	return s.eligible, nil
}

func (s *stubBackfillRepo) GetProcessByOrder(_ context.Context, _ pgx.Tx, orderID uuid.UUID) (*model.FulfillmentProcess, error) {
	if s.existing[orderID] {
		return &model.FulfillmentProcess{ID: uuid.New(), OrderID: orderID}, nil
	}
	return nil, pgx.ErrNoRows
}

func (s *stubBackfillRepo) CreateProcess(_ context.Context, _ pgx.Tx, p model.FulfillmentProcess) (*model.FulfillmentProcess, error) {
	s.createdProcesses = append(s.createdProcesses, p.OrderID)
	out := p
	out.ID = uuid.New()
	out.AggregateStatus = model.ProcessStatusNew
	out.HealthStatus = model.ProcessHealthOK
	return &out, nil
}

func (s *stubBackfillRepo) EnqueueEvent(_ context.Context, _ repository.Querier, e model.OrchestrationOutboxEvent) (bool, *model.OrchestrationOutboxEvent, error) {
	s.enqueuedEvents = append(s.enqueuedEvents, e)
	out := e
	return true, &out, nil
}

func newBackfillServiceWithStub(stub *stubBackfillRepo) *FulfillmentBackfillService {
	return &FulfillmentBackfillService{fulfillment: stub, orchestration: stub}
}

// TestBackfill_DryRun_CountsNoWrites verifies a dry run counts the eligible orders
// and performs ZERO writes (no process created, no event enqueued).
func TestBackfill_DryRun_CountsNoWrites(t *testing.T) {
	tenantID := uuid.New()
	o1, o2, o3 := uuid.New(), uuid.New(), uuid.New()
	stub := &stubBackfillRepo{eligible: []uuid.UUID{o1, o2, o3}}
	svc := newBackfillServiceWithStub(stub)

	report, err := svc.BackfillActiveOrderProcesses(context.Background(), nil, tenantID, BackfillOptions{DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, 3, report.Scanned)
	assert.Equal(t, 3, report.NeedingProcess, "dry run counts what a write run would create")
	assert.Equal(t, 0, report.Created)
	assert.Equal(t, 0, report.Skipped)
	assert.Equal(t, 0, report.Errors)

	assert.Empty(t, stub.createdProcesses, "dry run creates no processes")
	assert.Empty(t, stub.enqueuedEvents, "dry run enqueues no events")
}

// TestBackfill_WriteRun_CreatesProcessAndEvent verifies a write run creates exactly
// one process + one order.created event per eligible order, using the SAME
// idempotency key live creation uses, and that the canonical terminal-status list
// is passed to the eligibility query (so terminal orders are excluded at the source).
func TestBackfill_WriteRun_CreatesProcessAndEvent(t *testing.T) {
	tenantID := uuid.New()
	o1, o2 := uuid.New(), uuid.New()
	stub := &stubBackfillRepo{eligible: []uuid.UUID{o1, o2}}
	svc := newBackfillServiceWithStub(stub)

	report, err := svc.BackfillActiveOrderProcesses(context.Background(), nil, tenantID, BackfillOptions{DryRun: false})
	require.NoError(t, err)

	assert.Equal(t, 2, report.Scanned)
	assert.Equal(t, 2, report.NeedingProcess)
	assert.Equal(t, 2, report.Created)
	assert.Equal(t, 0, report.Skipped)
	assert.Equal(t, 0, report.Errors)

	require.Len(t, stub.createdProcesses, 2)
	require.Len(t, stub.enqueuedEvents, 2)
	for i, ev := range stub.enqueuedEvents {
		assert.Equal(t, tenantID, ev.TenantID)
		assert.Equal(t, EventOrderCreated, ev.EventType)
		orderID := stub.createdProcesses[i]
		assert.Equal(t, EventOrderCreated+":"+orderID.String(), ev.IdempotencyKey,
			"backfill uses the same idempotency key as live creation")
	}

	// The eligibility query must be handed the canonical terminal statuses so
	// completed/cancelled/refunded orders are filtered out at the source.
	assert.Equal(t, model.TerminalOrderStatuses, stub.lastTerminal)
}

// TestBackfill_WriteRun_SkipsAlreadyHaveProcess verifies the idempotent path: an
// order that already has a process by write-time is counted as Skipped and is never
// re-created (no duplicate process, no second event).
func TestBackfill_WriteRun_SkipsAlreadyHaveProcess(t *testing.T) {
	tenantID := uuid.New()
	missing, already := uuid.New(), uuid.New()
	stub := &stubBackfillRepo{
		eligible: []uuid.UUID{missing, already},
		existing: map[uuid.UUID]bool{already: true},
	}
	svc := newBackfillServiceWithStub(stub)

	report, err := svc.BackfillActiveOrderProcesses(context.Background(), nil, tenantID, BackfillOptions{DryRun: false})
	require.NoError(t, err)

	assert.Equal(t, 2, report.Scanned)
	assert.Equal(t, 2, report.NeedingProcess)
	assert.Equal(t, 1, report.Created, "only the truly-missing order is created")
	assert.Equal(t, 1, report.Skipped, "the already-have-process order is skipped idempotently")
	assert.Equal(t, 0, report.Errors)

	require.Len(t, stub.createdProcesses, 1)
	assert.Equal(t, missing, stub.createdProcesses[0])
	require.Len(t, stub.enqueuedEvents, 1, "no second event for the already-have order")
}

// TestBackfill_EmptyEligibleSet is the resumability terminal state: a fully
// backfilled tenant returns an all-zero report (worker idles).
func TestBackfill_EmptyEligibleSet(t *testing.T) {
	stub := &stubBackfillRepo{eligible: nil}
	svc := newBackfillServiceWithStub(stub)

	report, err := svc.BackfillActiveOrderProcesses(context.Background(), nil, uuid.New(), BackfillOptions{DryRun: false})
	require.NoError(t, err)

	assert.Equal(t, BackfillReport{}, report)
	assert.Empty(t, stub.createdProcesses)
	assert.Empty(t, stub.enqueuedEvents)
}

// TestBackfill_BatchSizeCapsScan verifies BatchSize bounds the scan (resumability:
// the rest is drained on the next call).
func TestBackfill_BatchSizeCapsScan(t *testing.T) {
	o1, o2, o3 := uuid.New(), uuid.New(), uuid.New()
	stub := &stubBackfillRepo{eligible: []uuid.UUID{o1, o2, o3}}
	svc := newBackfillServiceWithStub(stub)

	report, err := svc.BackfillActiveOrderProcesses(context.Background(), nil, uuid.New(), BackfillOptions{DryRun: true, BatchSize: 2})
	require.NoError(t, err)

	assert.Equal(t, 2, report.Scanned, "BatchSize caps how many orders a pass examines")
	assert.Equal(t, 2, report.NeedingProcess)
}

// TestIsTerminalOrderStatus is a focused check that the backfill's exclusion set is
// exactly completed/cancelled/refunded — the terminal nodes of the order lifecycle.
func TestIsTerminalOrderStatus(t *testing.T) {
	for _, s := range []string{
		model.OrderStatusCompleted, model.OrderStatusCancelled, model.OrderStatusRefunded,
	} {
		assert.True(t, model.IsTerminalOrderStatus(s), "%s must be terminal", s)
	}
	for _, s := range []string{
		model.OrderStatusNew, model.OrderStatusConfirmed, model.OrderStatusProcessing,
		model.OrderStatusReadyToShip, model.OrderStatusShipped, model.OrderStatusInTransit,
		model.OrderStatusOutForDelivery, model.OrderStatusDelivered, model.OrderStatusOnHold,
	} {
		assert.False(t, model.IsTerminalOrderStatus(s), "%s must be non-terminal", s)
	}
}
