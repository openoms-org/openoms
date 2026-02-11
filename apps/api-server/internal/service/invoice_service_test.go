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

func TestInvoiceService_Create_ValidationError_MissingOrderID(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateInvoiceRequest{
		Provider: "fakturownia",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestInvoiceService_Create_ValidationError_MissingProvider(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateInvoiceRequest{
		OrderID: uuid.New(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}
