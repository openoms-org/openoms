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

func TestOrderService_Create_ValidationError_ZeroAmount(t *testing.T) {
	// TotalAmount = 0 should NOT be rejected — Validate() only rejects negative amounts.
	// Verify that the model validation passes for zero amount (boundary test).
	req := model.CreateOrderRequest{
		CustomerName: "Jan Kowalski",
		TotalAmount:  0,
	}
	err := req.Validate()
	assert.NoError(t, err, "TotalAmount=0 should pass validation — only negative is rejected")
}

func TestOrderService_Create_HTMLStrippedFromCustomerName(t *testing.T) {
	// The Create method sanitizes HTML via model.StripHTMLTags but does NOT reject it.
	// Verify: (1) validation passes, (2) StripHTMLTags removes the script tag.
	req := model.CreateOrderRequest{
		CustomerName: "<script>alert('xss')</script>Jan",
		TotalAmount:  10,
	}
	err := req.Validate()
	assert.NoError(t, err, "HTML in customer_name should pass validation — it gets stripped, not rejected")

	// Verify the stripping behavior directly
	stripped := model.StripHTMLTags(req.CustomerName)
	assert.Equal(t, "alert('xss')Jan", stripped)
	assert.NotContains(t, stripped, "<script>")
	assert.NotContains(t, stripped, "</script>")
}

func TestOrderService_TransitionStatus_ValidationError_WhitespaceStatus(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.StatusTransitionRequest{Status: "   "}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "status")
}

func TestOrderService_BulkTransitionStatus_ValidationError_TooManyOrders(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	// BulkStatusTransitionRequest.Validate() rejects > 100 orders
	ids := make([]uuid.UUID, 101)
	for i := range ids {
		ids[i] = uuid.New()
	}

	_, err := svc.BulkTransitionStatus(context.Background(), uuid.New(), model.BulkStatusTransitionRequest{
		OrderIDs: ids,
		Status:   "confirmed",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "100")
}

func TestOrderService_Update_ValidationError_NegativeAmount(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	negativeAmount := -50.0
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateOrderRequest{
		TotalAmount: &negativeAmount,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "total_amount")
}

func TestOrderService_Create_ValidationError_InvalidPriority(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateOrderRequest{
		CustomerName: "Jan Kowalski",
		TotalAmount:  10,
		Priority:     "critical", // not a valid priority
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "priority")
}

func TestOrderService_BulkTransitionStatus_ValidationError_WhitespaceStatus(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.BulkTransitionStatus(context.Background(), uuid.New(), model.BulkStatusTransitionRequest{
		OrderIDs: []uuid.UUID{uuid.New()},
		Status:   "   ",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "status")
}

func TestOrderService_Create_ValidationError_AutoCreateShipmentNoProvider(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateOrderRequest{
		CustomerName:       "Jan Kowalski",
		TotalAmount:        10,
		AutoCreateShipment: true,
		// ShipmentProvider is nil — should fail validation
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "shipment_provider")
}

func TestOrderService_Update_ValidationError_InvalidPriority(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil)

	badPriority := "extreme"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateOrderRequest{
		Priority: &badPriority,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "priority")
}

func TestStringOrEmpty(t *testing.T) {
	assert.Equal(t, "", stringOrEmpty(nil))
	s := "hello"
	assert.Equal(t, "hello", stringOrEmpty(&s))
}
