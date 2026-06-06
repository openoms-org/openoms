package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// OrderCreatedHandler is the first real orchestration handler: it advances a
// freshly created fulfillment process out of the "new" state by transitioning it
// to "validating" (health ok). It is registered against EventOrderCreated and
// invoked by the orchestration worker as it drains the outbox.
type OrderCreatedHandler struct {
	pool        *pgxpool.Pool
	fulfillment *repository.FulfillmentRepository
}

// NewOrderCreatedHandler creates an OrderCreatedHandler. pool MUST be the
// privileged worker pool — the handler sets the event's tenant context itself
// via database.WithTenant.
func NewOrderCreatedHandler(pool *pgxpool.Pool, fulfillment *repository.FulfillmentRepository) *OrderCreatedHandler {
	return &OrderCreatedHandler{pool: pool, fulfillment: fulfillment}
}

// Handle transitions the event's process to "validating". It loads the process
// by id within the event's tenant context, then updates its status. A process
// that is no longer "new" is left untouched (idempotent re-delivery).
func (h *OrderCreatedHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	return database.WithTenant(ctx, h.pool, event.TenantID, func(tx pgx.Tx) error {
		proc, err := h.fulfillment.GetProcess(ctx, tx, event.ProcessID)
		if err != nil {
			return fmt.Errorf("load process %s: %w", event.ProcessID, err)
		}
		if proc.AggregateStatus != model.ProcessStatusNew {
			return nil
		}
		if _, err := h.fulfillment.UpdateProcessStatus(ctx, tx, proc.ID, model.ProcessStatusValidating, model.ProcessHealthOK); err != nil {
			return fmt.Errorf("transition process %s to validating: %w", proc.ID, err)
		}
		return nil
	})
}
