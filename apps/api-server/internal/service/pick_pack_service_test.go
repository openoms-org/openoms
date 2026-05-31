package service

import (
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
)

// CompleteSession must transition orders to a status that is valid in the order
// state machine. Sessions are created only from orders in "processing" status,
// so the completed status must both exist and be reachable from "processing".
// Guards against regressing to the invalid "packed" status (OPE-471).
func TestPickPackCompletedOrderStatus_IsValidAndReachableFromProcessing(t *testing.T) {
	cfg := model.DefaultOrderStatusConfig()

	defined := false
	for _, s := range cfg.Statuses {
		if s.Key == pickPackCompletedOrderStatus {
			defined = true
			break
		}
	}
	assert.True(t, defined, "pick-pack completed status %q must be a defined order status", pickPackCompletedOrderStatus)

	assert.Contains(t, cfg.Transitions["processing"], pickPackCompletedOrderStatus,
		"a processing order must be allowed to transition to the pick-pack completed status")

	assert.NotEqual(t, "packed", pickPackCompletedOrderStatus,
		"'packed' is not a valid order state-machine status")
}
