package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// EventOrderCreated is the orchestration event type emitted when an order's
// fulfillment process is first created. It is the entry point of the fulfillment
// state machine and is consumed by OrderCreatedHandler.
const EventOrderCreated = "order.created"

// FulfillmentService bridges order creation to the fulfillment + orchestration
// foundation (OPE-414/415). On order creation it ensures a fulfillment process
// exists and enqueues the initial orchestration event — all inside the caller's
// order transaction, so the process and outbox event are committed atomically
// with the order. It is gated by a config flag and is a complete no-op when
// disabled, so wiring it in carries zero behavior change until enabled.
type FulfillmentService struct {
	enabled       bool
	fulfillment   *repository.FulfillmentRepository
	orchestration *repository.OrchestrationRepository
}

// NewFulfillmentService creates a FulfillmentService. When enabled is false every
// method is a no-op (returns nil, nil), so callers can wire it unconditionally.
func NewFulfillmentService(enabled bool, fulfillment *repository.FulfillmentRepository, orchestration *repository.OrchestrationRepository) *FulfillmentService {
	return &FulfillmentService{
		enabled:       enabled,
		fulfillment:   fulfillment,
		orchestration: orchestration,
	}
}

// Enabled reports whether fulfillment process creation is active.
func (s *FulfillmentService) Enabled() bool { return s != nil && s.enabled }

// EnsureProcessForOrder creates the fulfillment process for an order and enqueues
// the initial "order.created" orchestration event, running inside the caller's
// tenant transaction (tx) so it commits atomically with the order. It is a no-op
// when the service is disabled, and idempotent: if a process already exists for
// the order it is returned unchanged and no duplicate event is enqueued (the
// outbox dedups on idempotency key regardless).
func (s *FulfillmentService) EnsureProcessForOrder(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID) (*model.FulfillmentProcess, error) {
	if !s.Enabled() {
		return nil, nil
	}

	// Idempotent: reuse an existing process rather than creating a second one.
	if existing, err := s.fulfillment.GetProcessByOrder(ctx, tx, orderID); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup fulfillment process for order: %w", err)
	}

	proc, err := s.fulfillment.CreateProcess(ctx, tx, model.FulfillmentProcess{
		TenantID: tenantID,
		OrderID:  orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("create fulfillment process: %w", err)
	}

	if _, _, err := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
		TenantID:       tenantID,
		ProcessID:      proc.ID,
		EventType:      EventOrderCreated,
		IdempotencyKey: EventOrderCreated + ":" + orderID.String(),
		Payload:        map[string]string{"order_id": orderID.String()},
	}); err != nil {
		return nil, fmt.Errorf("enqueue order.created event: %w", err)
	}

	return proc, nil
}
