//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// newSyncShipmentService builds a ShipmentService wired to the OrderService that owns
// order status, which is what makes carrier-driven status changes run the same side
// effects as API-driven ones (CORR-01). objectStorage is nil: these tests only drive
// status transitions, never labels.
func newSyncShipmentService(pool *pgxpool.Pool, orders *service.OrderService) *service.ShipmentService {
	svc := service.NewShipmentService(
		repository.NewShipmentRepository(),
		repository.NewOrderRepository(),
		repository.NewProductRepository(),
		repository.NewAuditRepository(),
		repository.NewTenantRepository(pool),
		pool, nil, nil,
	)
	svc.SetOrderStatusSyncer(orders)
	return svc
}

// countAudit returns how many audit_log rows exist for an entity + action.
func countAudit(t *testing.T, ctx context.Context, entityID uuid.UUID, action string) int {
	t.Helper()
	var n int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_id = $1 AND action = $2`, entityID, action).Scan(&n))
	return n
}

// orderStatusOf reads an order's status and shipped_at directly.
func orderStatusOf(t *testing.T, ctx context.Context, orderID uuid.UUID) (string, *time.Time) {
	t.Helper()
	var status string
	var shippedAt *time.Time
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status, shipped_at FROM orders WHERE id = $1`, orderID).Scan(&status, &shippedAt))
	return status, shippedAt
}

// seedShipmentFor creates a shipment for an order and walks it to label_ready, the
// state the carrier graph requires before picked_up.
func seedShipmentFor(t *testing.T, ctx context.Context, tenant uuid.UUID, shipments *service.ShipmentService, orderID uuid.UUID) uuid.UUID {
	t.Helper()
	shipment, err := shipments.Create(ctx, tenant, model.CreateShipmentRequest{
		OrderID:  orderID,
		Provider: "inpost",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, err = shipments.TransitionStatus(ctx, tenant, shipment.ID,
		model.ShipmentStatusTransitionRequest{Status: "label_ready"}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	return shipment.ID
}

// TestShipmentSync_PickedUp_RunsOrderShipSideEffects is the CORR-01 regression: a
// carrier pickup marks the order shipped, and that has to carry the same side effects
// as an API-driven transition — the one-time stock decrement, shipped_at, and an
// order.status_changed audit row. Before the shared writer, ShipmentService wrote the
// status with a bare orderRepo.UpdateStatus and stock silently never moved.
func TestShipmentSync_PickedUp_RunsOrderShipSideEffects(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orders := newLifecycleOrderService(appPool)
	shipments := newSyncShipmentService(appPool, orders)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Carrier Widget", 25, false)
	seedStock(t, ctx, tenant, warehouse, product, 100, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":4,"price":25}]`, product))
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Carrier Buyer", Items: items, TotalAmount: 100, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, reserved := readStock(t, ctx, warehouse, product)
	require.Equal(t, 4, reserved, "create reserved the ordered quantity")

	shipmentID := seedShipmentFor(t, ctx, tenant, shipments, order.ID)

	_, err = shipments.TransitionStatus(ctx, tenant, shipmentID,
		model.ShipmentStatusTransitionRequest{Status: "picked_up"}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	status, shippedAt := orderStatusOf(t, ctx, order.ID)
	assert.Equal(t, "shipped", status, "carrier pickup marks the order shipped")
	assert.NotNil(t, shippedAt, "shipped_at is stamped, which is what gates the one-time decrement")

	require.Eventually(t, func() bool {
		qty, res := readStock(t, ctx, warehouse, product)
		return qty == 96 && res == 0
	}, 4*time.Second, 50*time.Millisecond, "carrier-driven ship decrements stock and releases the reservation")

	assert.Equal(t, 1, countAudit(t, ctx, order.ID, "order.status_changed"),
		"the order status change is audited, not written silently")
}

// TestShipmentSync_AllDelivered_RunsOrderSideEffects verifies the delivered branch
// goes through the same writer: the order only flips once every package is delivered,
// and the change is audited.
func TestShipmentSync_AllDelivered_RunsOrderSideEffects(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orders := newLifecycleOrderService(appPool)
	shipments := newSyncShipmentService(appPool, orders)

	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Two Parcel Buyer", TotalAmount: 50, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	first := seedShipmentFor(t, ctx, tenant, shipments, order.ID)
	second := seedShipmentFor(t, ctx, tenant, shipments, order.ID)

	deliver := func(shipmentID uuid.UUID) {
		for _, s := range []string{"picked_up", "in_transit", "delivered"} {
			_, err := shipments.TransitionStatus(ctx, tenant, shipmentID,
				model.ShipmentStatusTransitionRequest{Status: s}, uuid.Nil, "127.0.0.1")
			require.NoError(t, err)
		}
	}

	deliver(first)
	status, _ := orderStatusOf(t, ctx, order.ID)
	assert.Equal(t, "shipped", status, "one of two parcels delivered leaves the order shipped")

	deliver(second)
	status, _ = orderStatusOf(t, ctx, order.ID)
	assert.Equal(t, "delivered", status, "the order flips once every parcel is delivered")

	var deliveredAt *time.Time
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT delivered_at FROM orders WHERE id = $1`, order.ID).Scan(&deliveredAt))
	assert.NotNil(t, deliveredAt, "delivered_at is stamped by the shared writer")

	// shipped (first pickup) + delivered — both audited by the same writer.
	assert.Equal(t, 2, countAudit(t, ctx, order.ID, "order.status_changed"))
}

// TestShipmentSync_RepeatedPickup_DoesNotDoubleDecrement verifies the firstShip gate
// still holds through the shipment path: a second parcel picked up after the order is
// already shipped must not decrement stock again.
func TestShipmentSync_RepeatedPickup_DoesNotDoubleDecrement(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orders := newLifecycleOrderService(appPool)
	shipments := newSyncShipmentService(appPool, orders)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Split Widget", 10, false)
	seedStock(t, ctx, tenant, warehouse, product, 50, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":3,"price":10}]`, product))
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Split Buyer", Items: items, TotalAmount: 30, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	first := seedShipmentFor(t, ctx, tenant, shipments, order.ID)
	second := seedShipmentFor(t, ctx, tenant, shipments, order.ID)

	_, err = shipments.TransitionStatus(ctx, tenant, first,
		model.ShipmentStatusTransitionRequest{Status: "picked_up"}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		qty, _ := readStock(t, ctx, warehouse, product)
		return qty == 47
	}, 4*time.Second, 50*time.Millisecond, "first pickup decrements once")

	_, err = shipments.TransitionStatus(ctx, tenant, second,
		model.ShipmentStatusTransitionRequest{Status: "picked_up"}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	qty, _ := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 47, qty, "the second parcel does not decrement the order's stock again")
}
