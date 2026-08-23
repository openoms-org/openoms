//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// failingStockRepo is a WarehouseStockRepo whose reads always fail, standing in for a
// warehouse that is unreachable at the moment an order is accepted.
type failingStockRepo struct{ err error }

func (f failingStockRepo) ListByWarehouse(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.WarehouseStockListFilter) ([]model.WarehouseStock, int, error) {
	return nil, 0, f.err
}

func (f failingStockRepo) ListByProduct(_ context.Context, _ pgx.Tx, _ uuid.UUID) ([]model.WarehouseStock, error) {
	return nil, f.err
}

func (f failingStockRepo) Upsert(_ context.Context, _ pgx.Tx, _ *model.WarehouseStock) error {
	return f.err
}

func (f failingStockRepo) AdjustQuantity(_ context.Context, _ pgx.Tx, _, _ uuid.UUID, _ *uuid.UUID, _ int) error {
	return f.err
}

func countOrders(t *testing.T, ctx context.Context, tenant uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, superPool.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE tenant_id = $1`, tenant).Scan(&n))
	return n
}

// TestOrderStock_CreateReservesInSameTransaction is the CORR-02 regression: the
// reservation used to run in a second transaction after the order had committed, so a
// failure there left an order with no claim on stock. Now a failing reserve takes the
// order down with it.
func TestOrderStock_CreateReservesInSameTransaction(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Atomic Widget", 15, false)
	seedStock(t, ctx, tenant, warehouse, product, 40, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":6,"price":15}]`, product))

	// Happy path: the order and its reservation are both there.
	svc := newLifecycleOrderService(appPool)
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Atomic Buyer", Items: items, TotalAmount: 90, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 40, qty, "reserving never moves quantity")
	assert.Equal(t, 6, reserved)
	require.Equal(t, 1, countOrders(t, ctx, tenant))

	// Failing reserve: no order row survives, and the reservation is unchanged.
	broken := newLifecycleOrderService(appPool)
	broken.SetWarehouseStockRepo(failingStockRepo{err: errors.New("warehouse unavailable")})
	_, err = broken.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Rolled Back Buyer", Items: items, TotalAmount: 90, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.Error(t, err, "a reserve that cannot be completed fails the order")

	assert.Equal(t, 1, countOrders(t, ctx, tenant), "the rolled-back order left no row")
	qty, reserved = readStock(t, ctx, warehouse, product)
	assert.Equal(t, 40, qty)
	assert.Equal(t, 6, reserved, "reservations unchanged by the failed create")
}

// TestOrderStock_DuplicateReserves is the CORR-04 regression: a duplicated order carries
// the source order's lines, so it must claim stock like any other order. It used to
// claim none, which made every duplicate a free oversell.
func TestOrderStock_DuplicateReserves(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Dup Widget", 25, false)
	seedStock(t, ctx, tenant, warehouse, product, 30, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":4,"price":25}]`, product))
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Dup Stock Buyer", Items: items, TotalAmount: 100, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, reserved := readStock(t, ctx, warehouse, product)
	require.Equal(t, 4, reserved)

	dup, err := svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.NotEqual(t, src.ID, dup.ID)

	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 30, qty)
	assert.Equal(t, 8, reserved, "the duplicate reserves its own copy of the lines")
}

// TestOrderStock_DuplicateBundleExpandsComponents verifies the duplicate reserves the
// components of a bundle line, since bundles hold no warehouse stock themselves.
func TestOrderStock_DuplicateBundleExpandsComponents(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	bundle := seedProduct(t, ctx, tenant, "Dup Bundle", 0, true)
	component := seedProduct(t, ctx, tenant, "Dup Component", 10, false)
	_, err := superPool.Exec(ctx,
		`INSERT INTO product_bundles (tenant_id, bundle_product_id, component_product_id, quantity, position) VALUES ($1,$2,$3,3,0)`,
		tenant, bundle, component)
	require.NoError(t, err)
	seedStock(t, ctx, tenant, warehouse, component, 100, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":2,"price":0}]`, bundle))
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Dup Bundle Buyer", Items: items, TotalAmount: 0, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, reserved := readStock(t, ctx, warehouse, component)
	require.Equal(t, 6, reserved, "2 bundles x 3 components")

	_, err = svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, reserved = readStock(t, ctx, warehouse, component)
	assert.Equal(t, 12, reserved, "the duplicate reserves the components too")

	// The bundle itself still holds no stock row.
	var bundleRows int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM warehouse_stock WHERE product_id = $1`, bundle).Scan(&bundleRows))
	assert.Zero(t, bundleRows)
}

// TestOrderStock_ReserveClampedToAvailable verifies an over-ordered line reserves only
// what is actually available (order acceptance is unchanged) and never pushes reserved
// past quantity.
func TestOrderStock_ReserveClampedToAvailable(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Scarce Widget", 5, false)
	seedStock(t, ctx, tenant, warehouse, product, 3, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":10,"price":5}]`, product))
	_, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Over Buyer", Items: items, TotalAmount: 50, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err, "an oversold line is still accepted")

	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 3, qty)
	assert.Equal(t, 3, reserved, "reserved never exceeds what the row has")
}

// TestOrderStock_ReserveIsTenantScoped guards the RLS boundary: reserving for one
// tenant's order cannot touch another tenant's stock row for the same product id.
func TestOrderStock_ReserveIsTenantScoped(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)
	warehouseA := seedWarehouse(t, ctx, tenantA)
	warehouseB := seedWarehouse(t, ctx, tenantB)
	productA := seedProduct(t, ctx, tenantA, "Shared Name", 10, false)
	productB := seedProduct(t, ctx, tenantB, "Shared Name", 10, false)
	seedStock(t, ctx, tenantA, warehouseA, productA, 20, 0)
	seedStock(t, ctx, tenantB, warehouseB, productB, 20, 0)

	svc := newLifecycleOrderService(appPool)
	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":5,"price":10}]`, productA))
	_, err := svc.Create(ctx, tenantA, model.CreateOrderRequest{
		CustomerName: "Tenant A", Items: items, TotalAmount: 50, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, reservedA := readStock(t, ctx, warehouseA, productA)
	_, reservedB := readStock(t, ctx, warehouseB, productB)
	assert.Equal(t, 5, reservedA)
	assert.Zero(t, reservedB, "the other tenant's stock is untouched")
}
