package service

import (
	"context"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// FulfillmentStepHandler acknowledges fulfillment.step outbox events (OPE-513).
//
// fulfillment.step events are observability-only step acknowledgements: the step
// outcome was already recorded SYNCHRONOUSLY by EmitFulfillmentStep's caller on
// the success/failure paths of shipment creation, label generation and tracking
// sync. The event exists to carry that step timeline durably through the
// transactional outbox (audit/inspection of the fulfillment process), NOT to
// trigger a side effect — by the time the worker drains it there is nothing left
// to execute. The correct handling is therefore a no-op ack: returning nil lets
// the worker mark the event succeeded.
//
// Without this handler registered, the dispatcher treats fulfillment.step as an
// unregistered event type (model.PermanentError) and the worker opens a SPURIOUS
// fulfillment blocker on a perfectly healthy operation whenever
// FULFILLMENT_PROCESS_ENABLED and ORCHESTRATION_WORKER_ENABLED are both on.
type FulfillmentStepHandler struct{}

// NewFulfillmentStepHandler creates a FulfillmentStepHandler.
func NewFulfillmentStepHandler() *FulfillmentStepHandler { return &FulfillmentStepHandler{} }

// Handle acks the event unconditionally — see the type doc for why a no-op is
// correct. It never inspects the payload: any enqueued fulfillment.step event is
// a valid step acknowledgement by construction (EmitFulfillmentStep validates the
// step key before enqueueing).
func (h *FulfillmentStepHandler) Handle(_ context.Context, _ model.OrchestrationOutboxEvent) error {
	return nil
}

// Compile-time assertion that this handler satisfies OrchestrationHandler.
var _ OrchestrationHandler = (*FulfillmentStepHandler)(nil)
