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

func TestReturnService_Create_ValidationError_MissingOrderID(t *testing.T) {
	svc := NewReturnService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateReturnRequest{
		Reason:       "defective",
		RefundAmount: 50,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestReturnService_Create_ValidationError_MissingReason(t *testing.T) {
	svc := NewReturnService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateReturnRequest{
		OrderID: uuid.New(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "reason")
}

func TestReturnService_Create_ValidationError_NegativeRefund(t *testing.T) {
	svc := NewReturnService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateReturnRequest{
		OrderID:      uuid.New(),
		Reason:       "defective",
		RefundAmount: -1,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestReturnService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewReturnService(nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateReturnRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestReturnService_TransitionStatus_ValidationError_EmptyStatus(t *testing.T) {
	svc := NewReturnService(nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.ReturnStatusRequest{Status: ""}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "status")
}
