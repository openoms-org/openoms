package service

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestApplyFeedFiltersExcludesProductWithoutCanonicalStock(t *testing.T) {
	// Legacy stock_quantity is not decremented on shipment, so it can claim
	// stock that the warehouse no longer holds. The feed must follow the
	// canonical value and drop the product.
	products := []model.Product{
		{ID: uuid.New(), Name: "Sold out", StockQuantity: 5, AvailableStock: 0},
	}

	filtered := applyFeedFilters(products, &model.ProductFeedConfig{ExcludeOutOfStock: true})

	if len(filtered) != 0 {
		t.Fatalf("len(filtered) = %d, want 0 — product has no canonical stock", len(filtered))
	}
}

func TestApplyFeedFiltersKeepsProductWithCanonicalStockOnly(t *testing.T) {
	// The mirror case: legacy column says zero, the warehouse says otherwise.
	products := []model.Product{
		{ID: uuid.New(), Name: "In stock", StockQuantity: 0, AvailableStock: 4},
	}

	filtered := applyFeedFilters(products, &model.ProductFeedConfig{ExcludeOutOfStock: true})

	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1 — product has canonical stock", len(filtered))
	}
}

func TestApplyFeedFiltersExcludesConfiguredCategories(t *testing.T) {
	excluded := "Gadgets"
	kept := "Tools"
	products := []model.Product{
		{ID: uuid.New(), Name: "Excluded", Category: &excluded, AvailableStock: 3},
		{ID: uuid.New(), Name: "Kept", Category: &kept, AvailableStock: 3},
	}

	filtered := applyFeedFilters(products, &model.ProductFeedConfig{
		ExcludedCategories: []string{excluded},
	})

	if len(filtered) != 1 || filtered[0].Name != "Kept" {
		t.Fatalf("filtered = %+v, want only the product outside the excluded category", filtered)
	}
}

func TestApplyFeedFiltersKeepsOutOfStockWhenNotConfigured(t *testing.T) {
	products := []model.Product{
		{ID: uuid.New(), Name: "Sold out", StockQuantity: 5, AvailableStock: 0},
	}

	filtered := applyFeedFilters(products, &model.ProductFeedConfig{ExcludeOutOfStock: false})

	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1 — exclusion is off", len(filtered))
	}
}

func TestAvailableStockInBatchesSplitsAndMerges(t *testing.T) {
	// A feed can cover the whole catalogue and AvailableStockBatch builds one
	// placeholder per ID, so the lookup is chunked. Every ID must still come
	// back, and no chunk may exceed the batch size.
	const total = 2500
	const batchSize = 1000

	ids := make([]uuid.UUID, total)
	for i := range ids {
		ids[i] = uuid.New()
	}

	var chunkSizes []int
	stock, err := availableStockInBatches(ids, batchSize, func(chunk []uuid.UUID) (map[uuid.UUID]int, error) {
		chunkSizes = append(chunkSizes, len(chunk))
		out := make(map[uuid.UUID]int, len(chunk))
		for _, id := range chunk {
			out[id] = 1
		}
		return out, nil
	})
	if err != nil {
		t.Fatalf("availableStockInBatches returned error: %v", err)
	}

	if len(stock) != total {
		t.Fatalf("len(stock) = %d, want %d — batches were not merged", len(stock), total)
	}
	for _, id := range ids {
		if _, ok := stock[id]; !ok {
			t.Fatalf("id %s missing from merged result", id)
		}
	}
	want := []int{1000, 1000, 500}
	if fmt.Sprint(chunkSizes) != fmt.Sprint(want) {
		t.Fatalf("chunk sizes = %v, want %v", chunkSizes, want)
	}
}

func TestAvailableStockInBatchesPropagatesError(t *testing.T) {
	ids := []uuid.UUID{uuid.New()}

	_, err := availableStockInBatches(ids, 10, func(_ []uuid.UUID) (map[uuid.UUID]int, error) {
		return nil, fmt.Errorf("boom")
	})
	if err == nil {
		t.Fatal("expected the fetch error to propagate")
	}
}

func TestAvailableStockInBatchesHandlesEmptyInput(t *testing.T) {
	stock, err := availableStockInBatches(nil, 10, func(_ []uuid.UUID) (map[uuid.UUID]int, error) {
		t.Fatal("fetch must not be called for an empty ID set")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("availableStockInBatches returned error: %v", err)
	}
	if len(stock) != 0 {
		t.Fatalf("len(stock) = %d, want 0", len(stock))
	}
}
