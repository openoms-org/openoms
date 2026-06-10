package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// TestFulfillmentStepHandler_AcksEveryEvent verifies the OPE-513 contract: a
// fulfillment.step event is an observability-only step acknowledgement, so the
// handler must ack (return nil) for ANY payload shape — never error, never panic.
// An error here would make the worker fail the event and open a spurious
// fulfillment blocker on a perfectly healthy operation.
func TestFulfillmentStepHandler_AcksEveryEvent(t *testing.T) {
	h := NewFulfillmentStepHandler()

	cases := []struct {
		name  string
		event model.OrchestrationOutboxEvent
	}{
		{"zero-value event", model.OrchestrationOutboxEvent{}},
		{"typical step payload", model.OrchestrationOutboxEvent{
			ID:             uuid.New(),
			TenantID:       uuid.New(),
			ProcessID:      uuid.New(),
			EventType:      EventFulfillmentStep,
			IdempotencyKey: EventFulfillmentStep + ":proc:create_shipment:succeeded:ship-1",
			Payload: map[string]any{
				"step_key": model.StepCreateShipment,
				"status":   model.FulfillmentStatusSucceeded,
				"provider": "inpost",
			},
		}},
		{"nil payload", model.OrchestrationOutboxEvent{
			EventType: EventFulfillmentStep,
			Payload:   nil,
		}},
		{"non-map payload", model.OrchestrationOutboxEvent{
			EventType: EventFulfillmentStep,
			Payload:   "not-a-map",
		}},
		{"unexpected payload keys", model.OrchestrationOutboxEvent{
			EventType: EventFulfillmentStep,
			Payload:   map[string]any{"surprise": []int{1, 2, 3}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				assert.NoError(t, h.Handle(context.Background(), tc.event))
			})
		})
	}
}
