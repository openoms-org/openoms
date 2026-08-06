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

// A product with no sales history must still report canonical available stock. It is the
// product whose purchasing decision is least certain, so forecasting it off the stale
// products.stock_quantity column is the worst place to be inconsistent.
func TestForecastService_ZeroSalesProductData_UsesCanonicalStock(t *testing.T) {
	productID := uuid.New()
	sku := "SKU-1"

	repo := newMockProductRepo()
	repo.byID[productID] = &model.Product{
		ID:            productID,
		Name:          "Widget",
		SKU:           &sku,
		Price:         19.99,
		StockQuantity: 42, // legacy, stale
	}
	repo.availableStock = map[uuid.UUID]int{productID: 7} // canonical

	s := &ForecastService{productRepo: repo}

	pd, err := s.zeroSalesProductData(context.Background(), nil, productID)

	require.NoError(t, err)
	assert.Equal(t, 7, pd.Stock)
	assert.NotEqual(t, 42, pd.Stock)
	assert.Equal(t, productID, pd.ProductID)
	assert.Equal(t, "Widget", pd.ProductName)
	assert.Equal(t, sku, pd.SKU)
	assert.InDelta(t, 19.99, pd.Price, 0.001)
}

// Consistency guard: products with sales read their stock from
// ProductRepo.AvailableStockBatch (fetchSalesHistory), so the zero-sales branch must go
// through the very same call rather than a second, divergent source.
func TestForecastService_ZeroSalesProductData_ReadsSameSourceAsProductsWithSales(t *testing.T) {
	productID := uuid.New()

	repo := newMockProductRepo()
	repo.byID[productID] = &model.Product{ID: productID, Name: "Widget", StockQuantity: 42}
	repo.availableStock = map[uuid.UUID]int{productID: 7}

	s := &ForecastService{productRepo: repo}

	_, err := s.zeroSalesProductData(context.Background(), nil, productID)
	require.NoError(t, err)

	require.Len(t, repo.availableStockCalls, 1, "must resolve stock via AvailableStockBatch")
	assert.Equal(t, []uuid.UUID{productID}, repo.availableStockCalls[0])
}

// A product absent from warehouse_stock resolves to 0 from the canonical read rather
// than silently falling back to the legacy column in the service layer (the fallback to
// products.stock_quantity lives in the repository query itself).
func TestForecastService_ZeroSalesProductData_NoCanonicalRowMeansZero(t *testing.T) {
	productID := uuid.New()

	repo := newMockProductRepo()
	repo.byID[productID] = &model.Product{ID: productID, Name: "Widget", StockQuantity: 42}

	s := &ForecastService{productRepo: repo}

	pd, err := s.zeroSalesProductData(context.Background(), nil, productID)

	require.NoError(t, err)
	assert.Equal(t, 0, pd.Stock)
}

func TestForecastService_ZeroSalesProductData_MissingProduct(t *testing.T) {
	s := &ForecastService{productRepo: newMockProductRepo()}

	pd, err := s.zeroSalesProductData(context.Background(), nil, uuid.New())

	assert.Nil(t, pd)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestForecastService_ZeroSalesProductData_PropagatesStockError(t *testing.T) {
	productID := uuid.New()
	stockErr := errors.New("boom")

	repo := newMockProductRepo()
	repo.byID[productID] = &model.Product{ID: productID, Name: "Widget"}
	repo.availableStockErr = stockErr

	s := &ForecastService{productRepo: repo}

	pd, err := s.zeroSalesProductData(context.Background(), nil, productID)

	assert.Nil(t, pd)
	assert.ErrorIs(t, err, stockErr)
}
