//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// seedVariant inserts a product variant and returns its id.
func seedVariant(t *testing.T, ctx context.Context, tenant, product uuid.UUID, sku string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO product_variants (id, tenant_id, product_id, sku, name) VALUES ($1,$2,$3,$4,$5)`,
		id, tenant, product, sku, "Variant "+sku)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM product_variants WHERE id = $1", id) })
	return id
}

// seedVariantStock inserts a warehouse_stock row bound to a variant.
func seedVariantStock(t *testing.T, ctx context.Context, tenant, warehouse, product, variant uuid.UUID, qty, reserved int) {
	t.Helper()
	_, err := superPool.Exec(ctx,
		`INSERT INTO warehouse_stock (tenant_id, warehouse_id, product_id, variant_id, quantity, reserved)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		tenant, warehouse, product, variant, qty, reserved)
	require.NoError(t, err)
}

func readVariantStock(t *testing.T, ctx context.Context, warehouse, product, variant uuid.UUID) (qty, reserved int) {
	t.Helper()
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT quantity, reserved FROM warehouse_stock
		 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id = $3`,
		warehouse, product, variant).Scan(&qty, &reserved))
	return qty, reserved
}

func readVariantlessStock(t *testing.T, ctx context.Context, warehouse, product uuid.UUID) (qty, reserved int) {
	t.Helper()
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT quantity, reserved FROM warehouse_stock
		 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id IS NULL`,
		warehouse, product).Scan(&qty, &reserved))
	return qty, reserved
}

// TestOrderStock_VariantRowsAreReservedAndShipped is the CORR-03 regression: a product
// whose warehouse stock is held on variant rows advertised availability (the canonical
// resolver sums every row) but reserved and decremented nothing, because the stock
// writes were filtered to `variant_id IS NULL` and matched no row at all.
func TestOrderStock_VariantRowsAreReservedAndShipped(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Variant Widget", 30, false)
	variant := seedVariant(t, ctx, tenant, product, "VW-RED")
	seedVariantStock(t, ctx, tenant, warehouse, product, variant, 20, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":4,"price":30}]`, product))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Variant Buyer", Items: items, TotalAmount: 120, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	qty, reserved := readVariantStock(t, ctx, warehouse, product, variant)
	assert.Equal(t, 20, qty)
	assert.Equal(t, 4, reserved, "the variant row is reserved")

	_, err = svc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: model.OrderStatusShipped, Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		q, r := readVariantStock(t, ctx, warehouse, product, variant)
		return q == 16 && r == 0
	}, 4*time.Second, 50*time.Millisecond, "shipping decrements the variant row and clears its reservation")
}

// TestOrderStock_VariantAndVariantlessRowsBothDrain verifies a product holding both a
// variant-less row and a variant row drains across them, rather than silently ignoring
// one of the two.
func TestOrderStock_VariantAndVariantlessRowsBothDrain(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Mixed Widget", 12, false)
	variant := seedVariant(t, ctx, tenant, product, "MW-BLUE")
	seedStock(t, ctx, tenant, warehouse, product, 3, 0)
	seedVariantStock(t, ctx, tenant, warehouse, product, variant, 5, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":7,"price":12}]`, product))
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Mixed Buyer", Items: items, TotalAmount: 84, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, plainReserved := readVariantlessStock(t, ctx, warehouse, product)
	_, variantReserved := readVariantStock(t, ctx, warehouse, product, variant)
	assert.Equal(t, 7, plainReserved+variantReserved, "the 7 ordered units are reserved across both rows")
	assert.LessOrEqual(t, plainReserved, 3, "no row is reserved beyond what it holds")
	assert.LessOrEqual(t, variantReserved, 5)
}

// TestOrderStock_VariantReservationReleasedOnCancel verifies cancelling releases the
// variant row's reservation instead of leaving stock permanently claimed.
func TestOrderStock_VariantReservationReleasedOnCancel(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Cancel Variant Widget", 8, false)
	variant := seedVariant(t, ctx, tenant, product, "CVW-1")
	seedVariantStock(t, ctx, tenant, warehouse, product, variant, 10, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":3,"price":8}]`, product))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Cancel Variant Buyer", Items: items, TotalAmount: 24, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, reserved := readVariantStock(t, ctx, warehouse, product, variant)
	require.Equal(t, 3, reserved)

	_, err = svc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: model.OrderStatusCancelled}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		q, r := readVariantStock(t, ctx, warehouse, product, variant)
		return q == 10 && r == 0
	}, 4*time.Second, 50*time.Millisecond, "cancelling releases the variant row's reservation, quantity untouched")
}
