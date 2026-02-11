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

func TestProductService_Create_ValidationError_MissingName(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Price: 10.99,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestProductService_Create_ValidationError_NegativePrice(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:  "Test Product",
		Price: -5.00,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestProductService_Update_ValidationError(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	negativePrice := -10.0
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		Price: &negativePrice,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}
