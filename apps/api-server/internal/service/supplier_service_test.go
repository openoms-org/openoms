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

func TestSupplierService_Create_ValidationError_MissingName(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateSupplierRequest{
		FeedFormat: "iof",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestSupplierService_Create_ValidationError_InvalidFeedFormat(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateSupplierRequest{
		Name:       "Test Supplier",
		FeedFormat: "xml",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "feed_format")
}

func TestSupplierService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateSupplierRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}
