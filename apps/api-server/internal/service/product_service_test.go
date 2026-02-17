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

// --- Additional Create Tests ---

func TestProductService_Create_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:  "   ",
		Price: 10.0,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestProductService_Create_ValidationError_InvalidSource(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:   "Valid Product",
		Price:  10.0,
		Source: "invalid_source",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "source")
}

func TestProductService_Create_ValidationError_NegativeStockQuantity(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:     "Valid Product",
		Price:    10.0,
		StockQty: -5,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "stock_quantity")
}

func TestProductService_Create_ValidationError_NameTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:  longName,
		Price: 10.0,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestProductService_Create_ValidationError_SKUTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longSKU := strings.Repeat("x", 101)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:  "Valid Product",
		Price: 10.0,
		SKU:   &longSKU,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "sku")
}

func TestProductService_Create_ValidationError_EANTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longEAN := strings.Repeat("9", 51)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:  "Valid Product",
		Price: 10.0,
		EAN:   &longEAN,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "ean")
}

func TestProductService_Create_ValidationError_DescriptionShortTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longDesc := strings.Repeat("d", 1001)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:             "Valid Product",
		Price:            10.0,
		DescriptionShort: longDesc,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "description_short")
}

func TestProductService_Create_ValidationError_DescriptionLongTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longDesc := strings.Repeat("d", 10001)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:            "Valid Product",
		Price:           10.0,
		DescriptionLong: longDesc,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "description_long")
}

func TestProductService_Create_ValidationError_CategoryTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longCat := strings.Repeat("c", 101)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateProductRequest{
		Name:     "Valid Product",
		Price:    10.0,
		Category: &longCat,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "category")
}

// --- Additional Update Tests ---

func TestProductService_Update_ValidationError_NoFieldsProvided(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "at least one field")
}

func TestProductService_Update_ValidationError_InvalidSource(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	badSource := "ebay"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		Source: &badSource,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "source")
}

func TestProductService_Update_ValidationError_NegativeStockQuantity(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	negStock := -1
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		StockQuantity: &negStock,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "stock_quantity")
}

func TestProductService_Update_ValidationError_NameTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		Name: &longName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestProductService_Update_ValidationError_SKUTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longSKU := strings.Repeat("x", 101)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		SKU: &longSKU,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "sku")
}

func TestProductService_Update_ValidationError_EANTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longEAN := strings.Repeat("9", 51)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		EAN: &longEAN,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "ean")
}

func TestProductService_Update_ValidationError_DescriptionShortTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longDesc := strings.Repeat("d", 1001)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		DescriptionShort: &longDesc,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "description_short")
}

func TestProductService_Update_ValidationError_DescriptionLongTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longDesc := strings.Repeat("d", 10001)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		DescriptionLong: &longDesc,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "description_long")
}

func TestProductService_Update_ValidationError_CategoryTooLong(t *testing.T) {
	svc := NewProductService(nil, nil, nil, nil)

	longCat := strings.Repeat("c", 101)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateProductRequest{
		Category: &longCat,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "category")
}
