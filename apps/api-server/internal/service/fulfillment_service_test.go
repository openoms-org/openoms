package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// TestFulfillmentService_Disabled_NoOp verifies that a disabled service is a
// complete no-op: EnsureProcessForOrder returns (nil, nil) without touching the
// repositories or the (nil) transaction — the guarantee that wiring it in is
// zero behavior change until the flag is on.
func TestFulfillmentService_Disabled_NoOp(t *testing.T) {
	svc := NewFulfillmentService(false, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	assert.False(t, svc.Enabled())

	// tx is nil: if the disabled path touched it, this would panic.
	proc, err := svc.EnsureProcessForOrder(context.Background(), nil, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, proc)
}

// TestFulfillmentService_Enabled reports enabled when constructed with the flag on.
func TestFulfillmentService_Enabled(t *testing.T) {
	svc := NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	assert.True(t, svc.Enabled())
}
