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

// --- Create Variant Tests ---

func TestVariantService_Create_ValidationError_EmptyName(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name: "",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestVariantService_Create_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name: "   ",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestVariantService_Create_ValidationError_NegativeStockQuantity(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name:          "Valid Name",
		StockQuantity: -1,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "stock_quantity")
}

func TestVariantService_Create_ValidationError_NegativePriceOverride(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	negPrice := -5.0
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name:          "Valid Name",
		PriceOverride: &negPrice,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "price_override")
}

func TestVariantService_Create_ValidationError_NameTooLong(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name: longName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestVariantService_Create_ValidationError_SKUTooLong(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	longSKU := strings.Repeat("x", 101)
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name: "Valid Name",
		SKU:  &longSKU,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "sku")
}

func TestVariantService_Create_ValidationError_EANTooLong(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	longEAN := strings.Repeat("9", 51)
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), model.CreateVariantRequest{
		Name: "Valid Name",
		EAN:  &longEAN,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "ean")
}

// --- Update Variant Tests ---

func TestVariantService_Update_ValidationError_NoFieldsProvided(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "at least one field")
}

func TestVariantService_Update_ValidationError_EmptyName(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	emptyName := ""
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		Name: &emptyName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestVariantService_Update_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	wsName := "   "
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		Name: &wsName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestVariantService_Update_ValidationError_NegativeStockQuantity(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	negStock := -10
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		StockQuantity: &negStock,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "stock_quantity")
}

func TestVariantService_Update_ValidationError_NegativePriceOverride(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	negPrice := -1.0
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		PriceOverride: &negPrice,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "price_override")
}

func TestVariantService_Update_ValidationError_NameTooLong(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		Name: &longName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestVariantService_Update_ValidationError_SKUTooLong(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	longSKU := strings.Repeat("x", 101)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		SKU: &longSKU,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "sku")
}

func TestVariantService_Update_ValidationError_EANTooLong(t *testing.T) {
	svc := NewVariantService(nil, nil, nil, nil)

	longEAN := strings.Repeat("9", 51)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateVariantRequest{
		EAN: &longEAN,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "ean")
}
