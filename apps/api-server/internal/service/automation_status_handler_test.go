package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// fakeAutomationOrders is a minimal AutomationOrderTransitioner for handler tests.
type fakeAutomationOrders struct {
	order           *model.Order
	getErr          error
	transitionErr   error
	transitionCalls int
	lastReq         model.StatusTransitionRequest
}

func (f *fakeAutomationOrders) Get(_ context.Context, _, _ uuid.UUID) (*model.Order, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.order, nil
}

func (f *fakeAutomationOrders) TransitionStatus(_ context.Context, _, _ uuid.UUID, req model.StatusTransitionRequest, _ uuid.UUID, _ string) (*model.Order, error) {
	f.transitionCalls++
	f.lastReq = req
	if f.transitionErr != nil {
		return nil, f.transitionErr
	}
	out := *f.order
	out.Status = req.Status
	return &out, nil
}

func setStatusEvent(orderID uuid.UUID, target string) model.OrchestrationOutboxEvent {
	return model.OrchestrationOutboxEvent{
		TenantID:  uuid.New(),
		ProcessID: uuid.New(),
		EventType: "automation.set_status",
		Payload: map[string]any{
			"order_id":      orderID.String(),
			"target_status": target,
			"rule_id":       uuid.New().String(),
			"action_index":  0,
		},
	}
}

// TestAutomationStatusHandler_Transitions verifies the happy path: an order not yet
// in the target status is transitioned exactly once, and — with no force opt-in in
// the payload — the tenant's transition graph is enforced.
func TestAutomationStatusHandler_Transitions(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeAutomationOrders{order: &model.Order{ID: orderID, Status: "new"}}
	h := NewAutomationStatusTransitionHandler(orders)

	err := h.Handle(context.Background(), setStatusEvent(orderID, "confirmed"))
	require.NoError(t, err)
	assert.Equal(t, 1, orders.transitionCalls, "transitions once toward the target")
	assert.False(t, orders.lastReq.Force, "no opt-in in the payload -> tenant graph enforced")
}

// TestAutomationStatusHandler_ForceOptIn verifies the rule's force opt-in travels
// through the outbox payload and is applied to the transition.
func TestAutomationStatusHandler_ForceOptIn(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeAutomationOrders{order: &model.Order{ID: orderID, Status: "new"}}
	h := NewAutomationStatusTransitionHandler(orders)

	event := model.OrchestrationOutboxEvent{
		TenantID:  uuid.New(),
		ProcessID: uuid.New(),
		EventType: "automation.set_status",
		Payload: map[string]any{
			"order_id":      orderID.String(),
			"target_status": "delivered",
			"rule_id":       uuid.New().String(),
			"action_index":  0,
			"force":         true,
		},
	}

	require.NoError(t, h.Handle(context.Background(), event))
	require.Equal(t, 1, orders.transitionCalls)
	assert.True(t, orders.lastReq.Force, "payload opt-in forces the transition")
}

// TestAutomationStatusHandler_IdempotentSkip verifies the core idempotency rule:
// an order already in the target status is a no-op — NO transition is attempted, so
// the per-transition side effects (and the self-recursion loop) do not re-fire.
func TestAutomationStatusHandler_IdempotentSkip(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeAutomationOrders{order: &model.Order{ID: orderID, Status: "confirmed"}}
	h := NewAutomationStatusTransitionHandler(orders)

	err := h.Handle(context.Background(), setStatusEvent(orderID, "confirmed"))
	require.NoError(t, err)
	assert.Equal(t, 0, orders.transitionCalls, "already-in-target is a no-op, no re-transition")
}

// TestAutomationStatusHandler_UnknownStatus_Permanent verifies an unknown target
// status is a permanent failure (so the worker opens a blocker instead of retrying
// forever).
func TestAutomationStatusHandler_UnknownStatus_Permanent(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeAutomationOrders{
		order:         &model.Order{ID: orderID, Status: "new"},
		transitionErr: ErrUnknownStatus,
	}
	h := NewAutomationStatusTransitionHandler(orders)

	err := h.Handle(context.Background(), setStatusEvent(orderID, "definitely-not-a-status"))
	require.Error(t, err)
	assert.True(t, model.IsPermanent(err), "unknown status must be a permanent error")
}

// TestAutomationStatusHandler_ValidationError_Permanent verifies a validation
// failure (e.g. empty status reaching the service) is permanent.
func TestAutomationStatusHandler_ValidationError_Permanent(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeAutomationOrders{
		order:         &model.Order{ID: orderID, Status: "new"},
		transitionErr: NewValidationError(errors.New("status is required")),
	}
	h := NewAutomationStatusTransitionHandler(orders)

	err := h.Handle(context.Background(), setStatusEvent(orderID, "confirmed"))
	require.Error(t, err)
	assert.True(t, model.IsPermanent(err), "validation failure must be a permanent error")
}

// TestAutomationStatusHandler_OrderNotFound_Permanent verifies a missing order is
// permanent (it can never be transitioned).
func TestAutomationStatusHandler_OrderNotFound_Permanent(t *testing.T) {
	orders := &fakeAutomationOrders{getErr: ErrOrderNotFound}
	h := NewAutomationStatusTransitionHandler(orders)

	err := h.Handle(context.Background(), setStatusEvent(uuid.New(), "confirmed"))
	require.Error(t, err)
	assert.True(t, model.IsPermanent(err), "missing order must be a permanent error")
}

// TestAutomationStatusHandler_TransientLoadError_Retryable verifies a non-sentinel
// load error is retryable (NOT permanent) so the worker re-queues it.
func TestAutomationStatusHandler_TransientLoadError_Retryable(t *testing.T) {
	orders := &fakeAutomationOrders{getErr: errors.New("connection reset")}
	h := NewAutomationStatusTransitionHandler(orders)

	err := h.Handle(context.Background(), setStatusEvent(uuid.New(), "confirmed"))
	require.Error(t, err)
	assert.False(t, model.IsPermanent(err), "a transient load error must be retryable")
}

// TestAutomationStatusHandler_BadPayload_Permanent verifies a malformed payload is
// permanent (retry cannot fix it).
func TestAutomationStatusHandler_BadPayload_Permanent(t *testing.T) {
	h := NewAutomationStatusTransitionHandler(&fakeAutomationOrders{})

	cases := []model.OrchestrationOutboxEvent{
		{EventType: "automation.set_status", Payload: nil},
		{EventType: "automation.set_status", Payload: map[string]any{"order_id": "not-a-uuid", "target_status": "confirmed"}},
		{EventType: "automation.set_status", Payload: map[string]any{"order_id": uuid.New().String(), "target_status": ""}},
	}
	for i, ev := range cases {
		err := h.Handle(context.Background(), ev)
		require.Error(t, err, "case %d", i)
		assert.True(t, model.IsPermanent(err), "case %d: malformed payload must be permanent", i)
	}
}
