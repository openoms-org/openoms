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

func TestPopulateComponentStock_UsesCanonicalAvailableStock(t *testing.T) {
	wh := uuid.New()
	sup := uuid.New()
	sold := uuid.New()

	repo := newMockProductRepo()
	repo.availableStock = map[uuid.UUID]int{wh: 4, sup: 7, sold: 0}

	s := &BundleService{productRepo: repo}
	components := []model.ProductBundle{
		{ComponentProductID: wh, ComponentStock: 100},
		{ComponentProductID: sup, ComponentStock: 7},
		{ComponentProductID: sold, ComponentStock: 50},
	}

	require.NoError(t, s.populateComponentStock(context.Background(), nil, components))

	assert.Equal(t, 4, components[0].ComponentStock, "warehouse-managed must use AvailableStockBatch 4, not leftover stock_quantity 100")
	assert.Equal(t, 7, components[1].ComponentStock, "no warehouse rows falls back inside AvailableStockBatch")
	assert.Equal(t, 0, components[2].ComponentStock, "empty warehouse rows must be 0, not leftover stock_quantity 50")
	require.Len(t, repo.availableStockCalls, 1)
	assert.Equal(t, []uuid.UUID{wh, sup, sold}, repo.availableStockCalls[0])
}

func TestPopulateComponentStock_MissingCanonicalRowMeansZero(t *testing.T) {
	id := uuid.New()
	repo := newMockProductRepo()
	s := &BundleService{productRepo: repo}
	components := []model.ProductBundle{{ComponentProductID: id, ComponentStock: 42}}

	require.NoError(t, s.populateComponentStock(context.Background(), nil, components))

	assert.Equal(t, 0, components[0].ComponentStock)
}

func TestPopulateComponentStock_EmptyListDoesNotQuery(t *testing.T) {
	repo := newMockProductRepo()
	s := &BundleService{productRepo: repo}

	require.NoError(t, s.populateComponentStock(context.Background(), nil, nil))

	assert.Empty(t, repo.availableStockCalls)
}

func TestPopulateComponentStock_PropagatesStockError(t *testing.T) {
	stockErr := errors.New("boom")
	repo := newMockProductRepo()
	repo.availableStockErr = stockErr
	s := &BundleService{productRepo: repo}
	components := []model.ProductBundle{{ComponentProductID: uuid.New(), ComponentStock: 9}}

	err := s.populateComponentStock(context.Background(), nil, components)

	assert.ErrorIs(t, err, stockErr)
	assert.Equal(t, 9, components[0].ComponentStock)
}

func TestAssembleableBundles(t *testing.T) {
	t.Run("min of floors", func(t *testing.T) {
		got := assembleableBundles([]model.ProductBundle{
			{ComponentStock: 4, Quantity: 1},
			{ComponentStock: 7, Quantity: 2},
			{ComponentStock: 9, Quantity: 3},
		})
		assert.Equal(t, 3, got)
	})
	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, 0, assembleableBundles(nil))
	})
	t.Run("zero quantity skipped", func(t *testing.T) {
		got := assembleableBundles([]model.ProductBundle{
			{ComponentStock: 100, Quantity: 0},
			{ComponentStock: 8, Quantity: 2},
		})
		assert.Equal(t, 4, got)
	})
	t.Run("all skipped", func(t *testing.T) {
		assert.Equal(t, 0, assembleableBundles([]model.ProductBundle{
			{ComponentStock: 5, Quantity: 0},
		}))
	})
	t.Run("sold out component zeros the bundle", func(t *testing.T) {
		got := assembleableBundles([]model.ProductBundle{
			{ComponentStock: 4, Quantity: 1},
			{ComponentStock: 7, Quantity: 2},
			{ComponentStock: 0, Quantity: 1},
		})
		assert.Equal(t, 0, got)
	})
}
