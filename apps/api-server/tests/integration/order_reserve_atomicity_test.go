//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// countOrders returns how many orders exist for a tenant.
func countOrders(t *testing.T, ctx context.Context, tenant uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE tenant_id = $1`, tenant).Scan(&n))
	return n
}

// blockStockWrites revokes UPDATE on warehouse_stock from the app role for the rest of
// the test, making every stock reservation fail with a permission error. It is the
// deterministic stand-in for "the reserve failed after the order row was written".
func blockStockWrites(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := superPool.Exec(ctx, `REVOKE UPDATE ON warehouse_stock FROM openoms_app`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), `GRANT UPDATE ON warehouse_stock TO openoms_app`)
	})
}

// TestOrderReserve_FailedReserveRollsBackCreate is the CORR-02 regression. The reserve
// used to run in its own transaction after the order was committed, so a reserve that
// failed left a live order holding no stock — overselling that nothing reports. The
// reserve now shares the create transaction: if it cannot be applied, no order exists.
func TestOrderReserve_FailedReserveRollsBackCreate(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Atomic Widget", 20, false)
	seedStock(t, ctx, tenant, warehouse, product, 100, 0)

	blockStockWrites(t, ctx)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":5,"price":20}]`, product))
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Atomic Buyer", Items: items, TotalAmount: 100, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")

	require.Error(t, err, "a reserve that cannot be applied must fail the create")
	assert.Equal(t, 0, countOrders(t, ctx, tenant), "no order is left behind without its reservation")

	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 100, qty)
	assert.Equal(t, 0, reserved, "stock is untouched")
}

// TestOrderReserve_CreateWithoutStockRowsStillSucceeds guards the opposite direction:
// tying the reserve to the create transaction must not make orders for products that
// have no warehouse rows fail. Nothing to reserve is not an error.
func TestOrderReserve_CreateWithoutStockRowsStillSucceeds(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	product := seedProduct(t, ctx, tenant, "Stockless Widget", 20, false)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":2,"price":20}]`, product))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Stockless Buyer", Items: items, TotalAmount: 40, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "new", order.Status)
}

// TestOrderReserve_Overselling_ReservesWhatIsAvailable pins the existing tolerance:
// ordering more than is on hand reserves what exists rather than failing the order.
func TestOrderReserve_Overselling_ReservesWhatIsAvailable(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Scarce Widget", 20, false)
	seedStock(t, ctx, tenant, warehouse, product, 3, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":10,"price":20}]`, product))
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Greedy Buyer", Items: items, TotalAmount: 200, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 3, qty)
	assert.Equal(t, 3, reserved, "reserves the available quantity, order still placed")
}

// TestOrderReserve_DuplicateReservesStock is the CORR-04 regression: Duplicate minted a
// real, fulfillable order but never reserved anything, so its lines were invisible to
// every availability calculation until someone shipped it.
func TestOrderReserve_DuplicateReservesStock(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Dup Widget", 15, false)
	seedStock(t, ctx, tenant, warehouse, product, 50, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":6,"price":15}]`, product))
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Dup Buyer", Items: items, TotalAmount: 90, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, reserved := readStock(t, ctx, warehouse, product)
	require.Equal(t, 6, reserved, "source order reserved")

	_, err = svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 50, qty, "duplicating does not consume stock")
	assert.Equal(t, 12, reserved, "the duplicate reserves its own lines")
}

// TestOrderReserve_DuplicateBundleExpandsComponents verifies the duplicate takes the
// same reserve path as create, bundle expansion included.
func TestOrderReserve_DuplicateBundleExpandsComponents(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	bundle := seedProduct(t, ctx, tenant, "DupBundle", 0, true)
	component := seedProduct(t, ctx, tenant, "DupComp", 10, false)
	_, err := superPool.Exec(ctx,
		`INSERT INTO product_bundles (tenant_id, bundle_product_id, component_product_id, quantity, position) VALUES ($1,$2,$3,2,0)`,
		tenant, bundle, component)
	require.NoError(t, err)
	seedStock(t, ctx, tenant, warehouse, component, 100, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":3,"price":0}]`, bundle))
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Dup Bundle Buyer", Items: items, TotalAmount: 30, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, reserved := readStock(t, ctx, warehouse, component)
	require.Equal(t, 6, reserved, "3 bundles x 2 components")

	_, err = svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, reserved = readStock(t, ctx, warehouse, component)
	assert.Equal(t, 12, reserved, "the duplicate expands the bundle and reserves components too")

	var bundleRows int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM warehouse_stock WHERE product_id = $1`, bundle).Scan(&bundleRows))
	assert.Equal(t, 0, bundleRows, "the virtual bundle never gets a stock row")
}

// TestOrderReserve_FailedReserveRollsBackDuplicate mirrors the create atomicity check
// on the duplicate path.
func TestOrderReserve_FailedReserveRollsBackDuplicate(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Dup Atomic Widget", 20, false)
	seedStock(t, ctx, tenant, warehouse, product, 100, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":4,"price":20}]`, product))
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Dup Atomic Buyer", Items: items, TotalAmount: 80, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, 1, countOrders(t, ctx, tenant))

	blockStockWrites(t, ctx)

	_, err = svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.Error(t, err, "a reserve that cannot be applied must fail the duplicate")
	assert.Equal(t, 1, countOrders(t, ctx, tenant), "no duplicate order is left behind")
}
