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

func TestOrderService_Create_ValidationError_MissingCustomerName(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateOrderRequest{
		CustomerName: "",
		TotalAmount:  10,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "customer_name")
}

func TestOrderService_Create_ValidationError_NegativeAmount(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateOrderRequest{
		CustomerName: "John",
		TotalAmount:  -1,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestOrderService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateOrderRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestOrderService_TransitionStatus_ValidationError_EmptyStatus(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.StatusTransitionRequest{Status: ""}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestOrderService_BulkTransitionStatus_ValidationError_EmptyOrders(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.BulkTransitionStatus(context.Background(), uuid.New(), model.BulkStatusTransitionRequest{
		OrderIDs: []uuid.UUID{},
		Status:   "confirmed",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestOrderService_BulkTransitionStatus_ValidationError_EmptyStatus(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.BulkTransitionStatus(context.Background(), uuid.New(), model.BulkStatusTransitionRequest{
		OrderIDs: []uuid.UUID{uuid.New()},
		Status:   "",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestStringOrEmpty(t *testing.T) {
	assert.Equal(t, "", stringOrEmpty(nil))
	s := "hello"
	assert.Equal(t, "hello", stringOrEmpty(&s))
}
