//go:build integration

package integration

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// availableStockOf reads the canonical available stock through the resolver, the same
// way the API and the sync paths do.
func availableStockOf(t *testing.T, ctx context.Context, tenant, product uuid.UUID) int {
	t.Helper()
	repo := repository.NewProductRepository()
	var available int
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		avail, err := repo.AvailableStockBatch(ctx, tx, []uuid.UUID{product})
		if err != nil {
			return err
		}
		available = avail[product]
		return nil
	}))
	return available
}

// TestProductStock_LegacyColumnIsNotAvailability is the CORR-08 pin: for a
// warehouse-tracked product, editing the legacy products.stock_quantity column does not
// change what the product actually has available, so it must not drive a stock-sync
// decision. The column is only availability for products with no warehouse rows.
func TestProductStock_LegacyColumnIsNotAvailability(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Warehouse Tracked", 50, false)

	// No warehouse row yet: the legacy column IS the availability (documented fallback).
	_, err := superPool.Exec(ctx, `UPDATE products SET stock_quantity = 9 WHERE id = $1`, product)
	require.NoError(t, err)
	assert.Equal(t, 9, availableStockOf(t, ctx, tenant, product),
		"with no warehouse rows the legacy column is the fallback")

	// Once the warehouse tracks it, availability comes from quantity - reserved and the
	// legacy column is irrelevant, however stale or large it is.
	seedStock(t, ctx, tenant, warehouse, product, 4, 1)
	assert.Equal(t, 3, availableStockOf(t, ctx, tenant, product),
		"warehouse-tracked availability is quantity - reserved")

	_, err = superPool.Exec(ctx, `UPDATE products SET stock_quantity = 999 WHERE id = $1`, product)
	require.NoError(t, err)
	assert.Equal(t, 3, availableStockOf(t, ctx, tenant, product),
		"the legacy column cannot inflate availability for a warehouse-tracked product")
}

// TestProductStock_UpdateDoesNotSyncWhenAvailabilityUnchanged verifies the gate on the
// manual-edit stock sync: editing the legacy column on a warehouse-tracked product
// leaves availability alone, so no stock-change event is recorded. Driving the gate off
// the column (as it used to) fired a relist/deactivate decision on a number the
// warehouse never sees.
func TestProductStock_UpdateDoesNotSyncWhenAvailabilityUnchanged(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	warehouse := seedWarehouse(t, ctx, tenant)

	productSvc := newStockSyncProductService(appPool)
	created, err := productSvc.Create(ctx, tenant, model.CreateProductRequest{
		Name: "Edited Widget", Price: 10, StockQty: 0,
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", created.ID) })
	seedStock(t, ctx, tenant, warehouse, created.ID, 5, 0)

	before := countStockSyncEvents(t, ctx, created.ID)

	newQty := 42
	_, err = productSvc.Update(ctx, tenant, created.ID, model.UpdateProductRequest{StockQuantity: &newQty}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	assert.Equal(t, 5, availableStockOf(t, ctx, tenant, created.ID), "availability is still the warehouse figure")
	assert.Equal(t, before, countStockSyncEvents(t, ctx, created.ID),
		"a legacy-column edit that changes nothing available records no stock change")
}

// TestProductStock_UpdateSyncsWhenAvailabilityMoves verifies the other half of the gate:
// for a product with no warehouse rows the legacy column IS availability, so editing it
// does move availability and a stock change is recorded.
func TestProductStock_UpdateSyncsWhenAvailabilityMoves(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)

	productSvc := newStockSyncProductService(appPool)
	created, err := productSvc.Create(ctx, tenant, model.CreateProductRequest{
		Name: "Untracked Widget", Price: 10, StockQty: 0,
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", created.ID) })

	newQty := 7
	_, err = productSvc.Update(ctx, tenant, created.ID, model.UpdateProductRequest{StockQuantity: &newQty}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	assert.Equal(t, 7, availableStockOf(t, ctx, tenant, created.ID))
	require.Eventually(t, func() bool { return countStockSyncEvents(t, ctx, created.ID) > 0 },
		4*time.Second, 50*time.Millisecond, "an availability change is recorded as a stock change")

	var oldQty, recordedNewQty, available int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT old_quantity, new_quantity, available_quantity FROM stock_sync_events
		 WHERE product_id = $1 ORDER BY created_at DESC LIMIT 1`, created.ID).
		Scan(&oldQty, &recordedNewQty, &available))
	assert.Equal(t, 0, oldQty)
	assert.Equal(t, 7, recordedNewQty, "the recorded quantities are the canonical availability")
}

func countStockSyncEvents(t *testing.T, ctx context.Context, product uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM stock_sync_events WHERE product_id = $1`, product).Scan(&n))
	return n
}

// newStockSyncProductService builds a ProductService wired to a real StockSyncService so
// the stock-change gate is exercised end to end.
func newStockSyncProductService(pool *pgxpool.Pool) *service.ProductService {
	productRepo := repository.NewProductRepository()
	svc := service.NewProductService(productRepo, repository.NewAuditRepository(), pool, nil)
	svc.SetStockSyncService(newStockSyncService(pool))
	return svc
}

// newStockSyncService builds the real StockSyncService (no marketplace credentials, so
// propagation stops at recording the event).
func newStockSyncService(pool *pgxpool.Pool) *service.StockSyncService {
	return service.NewStockSyncService(
		repository.NewStockSyncChannelRepository(),
		repository.NewStockSyncEventRepository(),
		repository.NewProductRepository(),
		repository.NewAuditRepository(),
		repository.NewProductListingRepository(),
		repository.NewIntegrationRepository(),
		pool, nil, nil, slog.Default(),
	)
}
