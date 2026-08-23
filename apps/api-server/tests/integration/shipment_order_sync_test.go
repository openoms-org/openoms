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

// newSyncShipmentService builds a ShipmentService whose carrier-driven order status
// changes fan out through the given OrderService, mirroring the production wiring.
func newSyncShipmentService(pool *pgxpool.Pool, orders *service.OrderService) *service.ShipmentService {
	svc := service.NewShipmentService(
		repository.NewShipmentRepository(),
		repository.NewOrderRepository(),
		repository.NewProductRepository(),
		repository.NewAuditRepository(),
		repository.NewTenantRepository(pool),
		pool, nil, nil,
	)
	svc.SetOrderStatusSideEffects(orders)
	return svc
}

func readOrderRow(t *testing.T, ctx context.Context, orderID uuid.UUID) (status string, shippedAt, deliveredAt *time.Time) {
	t.Helper()
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status, shipped_at, delivered_at FROM orders WHERE id = $1`, orderID).
		Scan(&status, &shippedAt, &deliveredAt))
	return status, shippedAt, deliveredAt
}

// advanceShipment walks a shipment through the carrier states leading to target.
func advanceShipment(t *testing.T, ctx context.Context, svc *service.ShipmentService, tenant, shipmentID uuid.UUID, statuses ...string) {
	t.Helper()
	for _, s := range statuses {
		_, err := svc.TransitionStatus(ctx, tenant, shipmentID,
			model.ShipmentStatusTransitionRequest{Status: s}, uuid.Nil, "127.0.0.1")
		require.NoErrorf(t, err, "shipment transition to %q", s)
	}
}

// TestShipmentSync_PickedUp_ShipsOrderAndDecrementsStock is the CORR-01 regression: a
// carrier picking the package up marks the order shipped, and that ship now runs the
// same side effects an operator-driven transition would — most importantly the one-time
// warehouse stock decrement, which the direct orderRepo.UpdateStatus write skipped.
func TestShipmentSync_PickedUp_ShipsOrderAndDecrementsStock(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)
	shipmentSvc := newSyncShipmentService(appPool, orderSvc)

	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Carrier Widget", 20, false)
	seedStock(t, ctx, tenant, warehouse, product, 100, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":5,"price":20}]`, product))
	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Carrier Buyer", Items: items, TotalAmount: 100, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	_, reserved := readStock(t, ctx, warehouse, product)
	require.Equal(t, 5, reserved, "order create reserves the ordered quantity")

	shipment, err := shipmentSvc.Create(ctx, tenant, model.CreateShipmentRequest{
		OrderID: order.ID, Provider: "inpost",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	advanceShipment(t, ctx, shipmentSvc, tenant, shipment.ID, "label_ready", "picked_up")

	status, shippedAt, _ := readOrderRow(t, ctx, order.ID)
	assert.Equal(t, model.OrderStatusShipped, status, "carrier pickup ships the order")
	require.NotNil(t, shippedAt, "shipped_at is stamped")

	require.Eventually(t, func() bool {
		qty, res := readStock(t, ctx, warehouse, product)
		return qty == 95 && res == 0
	}, 4*time.Second, 50*time.Millisecond, "the carrier-driven ship decrements stock and clears the reservation exactly once")

	// The synced status change is audited like an operator's, and names the shipment.
	var raw []byte
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT changes FROM audit_log WHERE entity_id = $1 AND action = 'order.status_changed'`, order.ID).Scan(&raw))
	var changes map[string]string
	require.NoError(t, json.Unmarshal(raw, &changes))
	assert.Equal(t, model.OrderStatusShipped, changes["to"])
	assert.Equal(t, shipment.ID.String(), changes["shipment_id"])

	// A further carrier update must not decrement a second time (firstShip gate).
	advanceShipment(t, ctx, shipmentSvc, tenant, shipment.ID, "in_transit")
	time.Sleep(500 * time.Millisecond)
	qty, _ := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 95, qty, "in_transit after shipped does not double-decrement")
}

// TestShipmentSync_Delivered_WaitsForAllPackages verifies the multi-package rule
// survives the refactor: the order is delivered only once every shipment is.
func TestShipmentSync_Delivered_WaitsForAllPackages(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)
	shipmentSvc := newSyncShipmentService(appPool, orderSvc)

	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Two Package Buyer", TotalAmount: 10, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	first, err := shipmentSvc.Create(ctx, tenant, model.CreateShipmentRequest{OrderID: order.ID, Provider: "inpost"}, uuid.Nil, "")
	require.NoError(t, err)
	second, err := shipmentSvc.Create(ctx, tenant, model.CreateShipmentRequest{OrderID: order.ID, Provider: "inpost"}, uuid.Nil, "")
	require.NoError(t, err)

	advanceShipment(t, ctx, shipmentSvc, tenant, first.ID, "label_ready", "picked_up", "in_transit", "delivered")
	status, _, deliveredAt := readOrderRow(t, ctx, order.ID)
	assert.Equal(t, model.OrderStatusShipped, status, "one package delivered is not the whole order")
	assert.Nil(t, deliveredAt)

	advanceShipment(t, ctx, shipmentSvc, tenant, second.ID, "label_ready", "picked_up", "in_transit", "delivered")
	status, _, deliveredAt = readOrderRow(t, ctx, order.ID)
	assert.Equal(t, model.OrderStatusDelivered, status, "all packages delivered -> order delivered")
	assert.NotNil(t, deliveredAt, "delivered_at is stamped")
}
