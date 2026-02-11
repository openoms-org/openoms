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

func TestShipmentService_Create_ValidationError_MissingOrderID(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		Provider: "inpost",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestShipmentService_Create_ValidationError_MissingProvider(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		OrderID: uuid.New(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}

func TestShipmentService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateShipmentRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestShipmentService_TransitionStatus_ValidationError_EmptyStatus(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.ShipmentStatusTransitionRequest{Status: ""}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}
