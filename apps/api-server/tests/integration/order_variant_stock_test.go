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
func seedVariant(t *testing.T, ctx context.Context, tenant, product uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO product_variants (id, tenant_id, product_id, name) VALUES ($1,$2,$3,$4)`,
		id, tenant, product, name)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM product_variants WHERE id = $1", id) })
	return id
}

// seedVariantStock inserts a warehouse_stock row scoped to a specific variant.
func seedVariantStock(t *testing.T, ctx context.Context, tenant, warehouse, product, variant uuid.UUID, qty, reserved int) {
	t.Helper()
	_, err := superPool.Exec(ctx,
		`INSERT INTO warehouse_stock (tenant_id, warehouse_id, product_id, variant_id, quantity, reserved)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		tenant, warehouse, product, variant, qty, reserved)
	require.NoError(t, err)
}

// readProductLevelStock reads the product-level (variant_id IS NULL) stock row. The
// shared readStock helper does not filter on variant_id, which is ambiguous once a
// product has both a product-level row and variant rows in the same warehouse.
func readProductLevelStock(t *testing.T, ctx context.Context, warehouse, product uuid.UUID) (qty, reserved int) {
	t.Helper()
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT quantity, reserved FROM warehouse_stock
		 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id IS NULL`,
		warehouse, product).Scan(&qty, &reserved))
	return qty, reserved
}

// readVariantStock reads the stock row for one variant.
func readVariantStock(t *testing.T, ctx context.Context, warehouse, product, variant uuid.UUID) (qty, reserved int) {
	t.Helper()
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT quantity, reserved FROM warehouse_stock
		 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id = $3`,
		warehouse, product, variant).Scan(&qty, &reserved))
	return qty, reserved
}

// TestVariantStock_ReserveAndShipHitMatchingVariantRow is the CORR-03 regression. Stock
// statements matched on `variant_id IS NULL`, so an order line naming a variant moved
// no stock at all while the walk still counted the quantity as consumed — the
// reservation silently evaporated. The line must now hit its own variant's row.
func TestVariantStock_ReserveAndShipHitMatchingVariantRow(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Shirt", 50, false)
	red := seedVariant(t, ctx, tenant, product, "Red / M")
	blue := seedVariant(t, ctx, tenant, product, "Blue / M")
	seedVariantStock(t, ctx, tenant, warehouse, product, red, 30, 0)
	seedVariantStock(t, ctx, tenant, warehouse, product, blue, 40, 0)

	items := json.RawMessage(fmt.Sprintf(
		`[{"product_id":"%s","variant_id":"%s","quantity":5,"price":50}]`, product, red))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Variant Buyer", Items: items, TotalAmount: 250, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, redReserved := readVariantStock(t, ctx, warehouse, product, red)
	_, blueReserved := readVariantStock(t, ctx, warehouse, product, blue)
	assert.Equal(t, 5, redReserved, "the ordered variant is reserved")
	assert.Equal(t, 0, blueReserved, "a sibling variant is untouched")

	_, err = svc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: "shipped", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		q, r := readVariantStock(t, ctx, warehouse, product, red)
		return q == 25 && r == 0
	}, 4*time.Second, 50*time.Millisecond, "shipping decrements the ordered variant")

	blueQty, blueReserved := readVariantStock(t, ctx, warehouse, product, blue)
	assert.Equal(t, 40, blueQty, "the sibling variant keeps its quantity")
	assert.Equal(t, 0, blueReserved)
}

// TestVariantStock_CancelReleasesMatchingVariantRow covers the release path.
func TestVariantStock_CancelReleasesMatchingVariantRow(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Hoodie", 120, false)
	large := seedVariant(t, ctx, tenant, product, "L")
	seedVariantStock(t, ctx, tenant, warehouse, product, large, 20, 0)

	items := json.RawMessage(fmt.Sprintf(
		`[{"product_id":"%s","variant_id":"%s","quantity":3,"price":120}]`, product, large))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Cancel Buyer", Items: items, TotalAmount: 360, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, reserved := readVariantStock(t, ctx, warehouse, product, large)
	require.Equal(t, 3, reserved)

	_, err = svc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: "cancelled", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		q, r := readVariantStock(t, ctx, warehouse, product, large)
		return q == 20 && r == 0
	}, 4*time.Second, 50*time.Millisecond, "cancelling releases the variant's reservation without consuming quantity")
}

// TestVariantStock_ProductLevelLinePrefersProductLevelRow verifies a line that names no
// variant still draws from the product-level row when one exists, and adjusts it once —
// not once per variant row the product happens to have.
func TestVariantStock_ProductLevelLinePrefersProductLevelRow(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Mixed Widget", 30, false)
	v1 := seedVariant(t, ctx, tenant, product, "V1")
	v2 := seedVariant(t, ctx, tenant, product, "V2")
	seedStock(t, ctx, tenant, warehouse, product, 100, 0) // product-level row
	seedVariantStock(t, ctx, tenant, warehouse, product, v1, 10, 0)
	seedVariantStock(t, ctx, tenant, warehouse, product, v2, 10, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":4,"price":30}]`, product))
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Mixed Buyer", Items: items, TotalAmount: 120, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, productReserved := readProductLevelStock(t, ctx, warehouse, product)
	assert.Equal(t, 4, productReserved, "the product-level row absorbs the whole line, exactly once")

	for _, v := range []uuid.UUID{v1, v2} {
		_, r := readVariantStock(t, ctx, warehouse, product, v)
		assert.Equal(t, 0, r, "variant rows are not touched by a product-level line")
	}
}

// TestVariantStock_VariantLineFallsBackToProductLevelRow covers the mismatch case: a
// line names a variant, but the tenant only keeps product-level stock. Drawing from the
// product-level row is what the availability figure promised (it sums every row for the
// product), and it keeps the reservation from silently disappearing.
func TestVariantStock_VariantLineFallsBackToProductLevelRow(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Untracked Variant Widget", 40, false)
	variant := seedVariant(t, ctx, tenant, product, "Only Option")
	seedStock(t, ctx, tenant, warehouse, product, 60, 0) // product-level only

	items := json.RawMessage(fmt.Sprintf(
		`[{"product_id":"%s","variant_id":"%s","quantity":7,"price":40}]`, product, variant))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Fallback Buyer", Items: items, TotalAmount: 280, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 7, reserved, "the variant line reserves against the product-level row")

	_, err = svc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: "shipped", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		q, r := readStock(t, ctx, warehouse, product)
		return q == 53 && r == 0
	}, 4*time.Second, 50*time.Millisecond, "and decrements it on ship")
}

// TestVariantStock_SameProductTwoVariantsOnOneOrder verifies two lines of the same
// product are tracked per variant rather than collapsed into one product total.
func TestVariantStock_SameProductTwoVariantsOnOneOrder(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	product := seedProduct(t, ctx, tenant, "Two Variant Widget", 25, false)
	small := seedVariant(t, ctx, tenant, product, "S")
	large := seedVariant(t, ctx, tenant, product, "L")
	seedVariantStock(t, ctx, tenant, warehouse, product, small, 12, 0)
	seedVariantStock(t, ctx, tenant, warehouse, product, large, 12, 0)

	items := json.RawMessage(fmt.Sprintf(
		`[{"product_id":"%s","variant_id":"%s","quantity":2,"price":25},
		  {"product_id":"%s","variant_id":"%s","quantity":5,"price":25}]`,
		product, small, product, large))
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Two Variant Buyer", Items: items, TotalAmount: 175, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, smallReserved := readVariantStock(t, ctx, warehouse, product, small)
	_, largeReserved := readVariantStock(t, ctx, warehouse, product, large)
	assert.Equal(t, 2, smallReserved)
	assert.Equal(t, 5, largeReserved)
}
