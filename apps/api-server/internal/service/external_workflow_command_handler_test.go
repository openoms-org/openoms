package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// fakeExternalWorkflowOrders records the requests the command handler issues.
type fakeExternalWorkflowOrders struct {
	order           *model.Order
	transitionCalls int
	lastStatusReq   model.StatusTransitionRequest
	updateCalls     int
	lastUpdateReq   model.UpdateOrderRequest
}

func (f *fakeExternalWorkflowOrders) Get(_ context.Context, _, _ uuid.UUID) (*model.Order, error) {
	return f.order, nil
}

func (f *fakeExternalWorkflowOrders) TransitionStatus(_ context.Context, _, _ uuid.UUID, req model.StatusTransitionRequest, _ uuid.UUID, _ string) (*model.Order, error) {
	f.transitionCalls++
	f.lastStatusReq = req
	out := *f.order
	out.Status = req.Status
	return &out, nil
}

func (f *fakeExternalWorkflowOrders) Update(_ context.Context, _, _ uuid.UUID, req model.UpdateOrderRequest, _ uuid.UUID, _ string) (*model.Order, error) {
	f.updateCalls++
	f.lastUpdateReq = req
	return f.order, nil
}

func externalWorkflowCommandEvent(orderID uuid.UUID, command, value string) model.OrchestrationOutboxEvent {
	return model.OrchestrationOutboxEvent{
		TenantID:  uuid.New(),
		ProcessID: uuid.New(),
		EventType: EventExternalWorkflowCommand,
		Payload: map[string]any{
			"order_id": orderID.String(),
			"command":  command,
			"value":    value,
		},
	}
}

// TestExternalWorkflowCommandHandler_SetStatus_RespectsTenantGraph verifies a
// callback-submitted set_status is graph-validated: the external engine is a third
// party and has no way to force an edge the tenant does not allow.
func TestExternalWorkflowCommandHandler_SetStatus_RespectsTenantGraph(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeExternalWorkflowOrders{order: &model.Order{ID: orderID, Status: "new"}}
	h := NewExternalWorkflowCommandHandler(orders)

	require.NoError(t, h.Handle(context.Background(), externalWorkflowCommandEvent(orderID, "set_status", "confirmed")))
	require.Equal(t, 1, orders.transitionCalls)
	assert.Equal(t, "confirmed", orders.lastStatusReq.Status)
	assert.False(t, orders.lastStatusReq.Force, "callbacks never bypass the tenant transition graph")
}

// TestExternalWorkflowCommandHandler_SetStatus_AlreadyInTarget verifies the
// idempotent skip still holds (no transition, no error).
func TestExternalWorkflowCommandHandler_SetStatus_AlreadyInTarget(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeExternalWorkflowOrders{order: &model.Order{ID: orderID, Status: "confirmed"}}
	h := NewExternalWorkflowCommandHandler(orders)

	require.NoError(t, h.Handle(context.Background(), externalWorkflowCommandEvent(orderID, "set_status", "confirmed")))
	assert.Equal(t, 0, orders.transitionCalls, "already-in-target is a no-op")
}

// TestExternalWorkflowCommandHandler_NonWhitelistedCommand_Permanent verifies a
// command outside the whitelist fails permanently instead of being retried.
func TestExternalWorkflowCommandHandler_NonWhitelistedCommand_Permanent(t *testing.T) {
	orderID := uuid.New()
	orders := &fakeExternalWorkflowOrders{order: &model.Order{ID: orderID, Status: "new"}}
	h := NewExternalWorkflowCommandHandler(orders)

	err := h.Handle(context.Background(), externalWorkflowCommandEvent(orderID, "delete_everything", "x"))
	require.Error(t, err)
	assert.True(t, model.IsPermanent(err))
	assert.Equal(t, 0, orders.transitionCalls)
}
