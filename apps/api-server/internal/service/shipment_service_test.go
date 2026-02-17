package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestShipmentService_Create_ValidationError_MissingOrderID(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		Provider: "inpost",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestShipmentService_Create_ValidationError_MissingProvider(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		OrderID: uuid.New(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}

func TestShipmentService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateShipmentRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestShipmentService_TransitionStatus_ValidationError_EmptyStatus(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.ShipmentStatusTransitionRequest{Status: ""}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestShipmentService_TransitionStatus_ValidationError_WhitespaceStatus(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.ShipmentStatusTransitionRequest{Status: "   "}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "status")
}

func TestShipmentService_Create_ValidationError_BothMissing(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	// Both OrderID (uuid.Nil) and Provider ("") are missing — should fail on order_id first
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		// OrderID defaults to uuid.Nil, Provider defaults to ""
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestShipmentService_Update_ValidationError_NegativeWeightAccepted(t *testing.T) {
	// UpdateShipmentRequest does NOT validate weight values (no negative check).
	// Verify that the model validation passes for a negative weight — it only
	// checks that at least one field is provided.
	negWeight := -5.0
	req := model.UpdateShipmentRequest{
		Weight: &negWeight,
	}
	err := req.Validate()
	assert.NoError(t, err, "negative weight should pass validation — no range check exists in UpdateShipmentRequest")
}

func TestShipmentService_Create_ValidationError_ProviderTooLong(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil)

	// CreateShipmentRequest.Validate() checks validateMaxLength("provider", ..., 100)
	var longProvider strings.Builder
	for range 101 {
		longProvider.WriteString("x")
	}

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		OrderID:  uuid.New(),
		Provider: longProvider.String(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}
