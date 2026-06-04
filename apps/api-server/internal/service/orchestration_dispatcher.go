package service

import (
	"context"
	"fmt"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// OrchestrationHandler executes one fulfillment side effect for an outbox event.
// A handler returns nil on success, a model.PermanentError (via model.Permanent)
// for non-retryable failures, or any other error to request a retry.
type OrchestrationHandler interface {
	Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error
}

// OrchestrationDispatcher routes outbox events to per-event-type handlers.
// Handlers are registered by the command/integration layers (OPE-416+).
type OrchestrationDispatcher struct {
	handlers map[string]OrchestrationHandler
}

// NewOrchestrationDispatcher creates an empty dispatcher.
func NewOrchestrationDispatcher() *OrchestrationDispatcher {
	return &OrchestrationDispatcher{handlers: map[string]OrchestrationHandler{}}
}

// Register binds a handler to an event type. The last registration wins.
func (d *OrchestrationDispatcher) Register(eventType string, h OrchestrationHandler) {
	d.handlers[eventType] = h
}

// Dispatch routes an event to its handler. An unregistered event type is a
// permanent failure (it would otherwise be retried forever); the worker turns
// that into a visible fulfillment blocker.
func (d *OrchestrationDispatcher) Dispatch(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	h, ok := d.handlers[event.EventType]
	if !ok {
		return model.Permanent(fmt.Errorf("no orchestration handler registered for event_type %q", event.EventType))
	}
	return h.Handle(ctx, event)
}
