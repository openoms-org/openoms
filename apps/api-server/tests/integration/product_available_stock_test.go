//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAvailableStockBatch_HybridWarehouseWithFallback verifies the canonical product
// availability read: warehouse_stock (SUM quantity - reserved) with a products.stock_quantity
// fallback only for products that have no warehouse rows.
//
//	P-WH    stock_quantity=100, warehouse 5-1 -> expect 4   (warehouse wins over stale 100)
//	P-SUP   stock_quantity=7,   no warehouse  -> expect 7   (fallback to products.stock_quantity)
//	P-SOLD  stock_quantity=50,  warehouse 3-3 -> expect 0   (warehouse rows exist & empty -> 0, not fallback 50)
func TestAvailableStockBatch_HybridWarehouseWithFallback(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	repo := repository.NewProductRepository()

	pWH := uuid.New()
	pSup := uuid.New()
	pSoldOut := uuid.New()
	whID := uuid.New()

	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		for _, p := range []struct {
			id    uuid.UUID
			sku   string
			stock int
		}{
			{pWH, "P-WH", 100},
			{pSup, "P-SUP", 7},
			{pSoldOut, "P-SOLD", 50},
		} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO products (id, tenant_id, sku, name, price, stock_quantity)
				 VALUES ($1,$2,$3,$4,0,$5)`,
				p.id, tenant, p.sku, p.sku, p.stock); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO warehouses (id, tenant_id, name, is_default) VALUES ($1,$2,'WH',true)`,
			whID, tenant); err != nil {
			return err
		}
		for _, ws := range []struct {
			product            uuid.UUID
			quantity, reserved int
		}{
			{pWH, 5, 1},
			{pSoldOut, 3, 3},
		} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO warehouse_stock (id, tenant_id, warehouse_id, product_id, quantity, reserved)
				 VALUES ($1,$2,$3,$4,$5,$6)`,
				uuid.New(), tenant, whID, ws.product, ws.quantity, ws.reserved); err != nil {
				return err
			}
		}
		return nil
	}))

	var got map[uuid.UUID]int
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		var err error
		got, err = repo.AvailableStockBatch(ctx, tx, []uuid.UUID{pWH, pSup, pSoldOut})
		return err
	}))

	require.Len(t, got, 3)
	assert.Equal(t, 4, got[pWH], "warehouse-managed must use warehouse_stock 5-1=4, not stale stock_quantity 100")
	assert.Equal(t, 7, got[pSup], "supplier-managed (no warehouse rows) must fall back to products.stock_quantity 7")
	assert.Equal(t, 0, got[pSoldOut], "warehouse rows exist but empty (3-3) must be 0, not fall back to stock_quantity 50")
}
