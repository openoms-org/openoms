//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleListComponents_HybridWarehouseStockWithFallback(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := service.NewBundleService(
		repository.NewBundleRepository(),
		repository.NewProductRepository(),
		nil,
		appPool,
	)

	bundleID := uuid.New()
	compWH := uuid.New()
	compSup := uuid.New()
	compSoldOut := uuid.New()
	whID := uuid.New()

	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		for _, p := range []struct {
			id       uuid.UUID
			sku      string
			stock    int
			isBundle bool
		}{
			{bundleID, "BUNDLE-1", 0, true},
			{compWH, "COMP-WH", 100, false},
			{compSup, "COMP-SUP", 7, false},
			{compSoldOut, "COMP-SOLDOUT", 50, false},
		} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO products (id, tenant_id, sku, name, price, stock_quantity, is_bundle)
				 VALUES ($1,$2,$3,$4,0,$5,$6)`,
				p.id, tenant, p.sku, p.sku, p.stock, p.isBundle); err != nil {
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
			{compWH, 5, 1},
			{compSoldOut, 3, 3},
		} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO warehouse_stock (id, tenant_id, warehouse_id, product_id, quantity, reserved)
				 VALUES ($1,$2,$3,$4,$5,$6)`,
				uuid.New(), tenant, whID, ws.product, ws.quantity, ws.reserved); err != nil {
				return err
			}
		}
		for pos, comp := range []struct {
			product  uuid.UUID
			quantity int
		}{
			{compWH, 1},
			{compSup, 2},
			{compSoldOut, 1},
		} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO product_bundles (id, tenant_id, bundle_product_id, component_product_id, quantity, position)
				 VALUES ($1,$2,$3,$4,$5,$6)`,
				uuid.New(), tenant, bundleID, comp.product, comp.quantity, pos); err != nil {
				return err
			}
		}
		return nil
	}))

	got, err := svc.ListComponents(ctx, tenant, bundleID)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, 4, got[0].ComponentStock, "warehouse-managed must use warehouse_stock 5-1=4, not stale products.stock_quantity 100")
	assert.Equal(t, 1, got[0].Quantity)
	assert.Equal(t, 7, got[1].ComponentStock, "supplier-managed (no warehouse rows) must fall back to products.stock_quantity 7")
	assert.Equal(t, 2, got[1].Quantity)
	assert.Equal(t, 0, got[2].ComponentStock, "warehouse rows exist but empty (3-3) must be 0, NOT fall back to products.stock_quantity 50")
	assert.Equal(t, 1, got[2].Quantity)

	assemble, err := svc.CalculateBundleStock(ctx, tenant, bundleID)
	require.NoError(t, err)
	assert.Equal(t, 0, assemble, "sold-out component must zero assembleable bundles")

	var raw []model.ProductBundle
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		var listErr error
		raw, listErr = repository.NewBundleRepository().ListByBundleProduct(ctx, tx, bundleID)
		return listErr
	}))
	require.Len(t, raw, 3)
	assert.Equal(t, 0, raw[0].ComponentStock)
	assert.Equal(t, 0, raw[1].ComponentStock)
	assert.Equal(t, 0, raw[2].ComponentStock)
}
