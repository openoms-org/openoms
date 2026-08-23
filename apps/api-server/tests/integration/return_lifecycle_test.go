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

// newReturnServiceWithRestock builds a ReturnService wired for restocking, mirroring
// production: the tenant repo supplies the policy and the OrderService performs the
// warehouse write.
func newReturnServiceWithRestock(pool *pgxpool.Pool) (*service.ReturnService, *service.OrderService) {
	orders := newLifecycleOrderService(pool)
	// Wired so a restock actually reaches the stock owner, which is what proves the
	// restock trigger type is one it accepts.
	orders.SetStockSyncService(newStockSyncService(pool))
	returns := service.NewReturnService(
		repository.NewReturnRepository(),
		repository.NewOrderRepository(),
		repository.NewAuditRepository(),
		pool, nil,
	)
	returns.SetRestockPolicy(repository.NewTenantRepository(pool), orders)
	return returns, orders
}

func setTenantSettings(t *testing.T, ctx context.Context, tenant uuid.UUID, settings string) {
	t.Helper()
	_, err := superPool.Exec(ctx, `UPDATE tenants SET settings = $2 WHERE id = $1`, tenant, settings)
	require.NoError(t, err)
}

func countAudit(t *testing.T, ctx context.Context, entityID uuid.UUID, action string) int {
	t.Helper()
	var n int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE entity_id = $1 AND action = $2`, entityID, action).Scan(&n))
	return n
}

// TestReturnLifecycle_CreatePublic_IsAudited is the CORR-05 regression: a return
// submitted through the public self-service form now goes through ReturnService, so it
// lands in the audit log with the order it belongs to and is marked as coming from the
// public form. The public endpoint used to insert the row itself and audit nothing.
func TestReturnLifecycle_CreatePublic_IsAudited(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	returns, orders := newReturnServiceWithRestock(appPool)

	email := "public-return-" + uuid.New().String()[:8] + "@example.com"
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Public Returner", CustomerEmail: &email, TotalAmount: 60, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	notes := "arrived scratched"
	ret, err := returns.CreatePublic(ctx, tenant, service.PublicReturnInput{
		OrderID:       order.ID,
		Reason:        "damaged",
		Notes:         &notes,
		CustomerEmail: email,
		ReturnToken:   "tok-" + uuid.New().String()[:12],
		IP:            "203.0.113.7",
	})
	require.NoError(t, err)
	assert.Equal(t, "requested", ret.Status)
	require.NotNil(t, ret.ReturnToken)
	require.NotNil(t, ret.CustomerEmail)
	assert.Equal(t, email, *ret.CustomerEmail)
	require.NotNil(t, ret.CustomerNotes)
	assert.Equal(t, notes, *ret.CustomerNotes)

	var raw []byte
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT changes FROM audit_log WHERE entity_id = $1 AND action = 'return.created'`, ret.ID).Scan(&raw))
	var changes map[string]string
	require.NoError(t, json.Unmarshal(raw, &changes))
	assert.Equal(t, order.ID.String(), changes["order_id"])
	assert.Equal(t, "damaged", changes["reason"])
	assert.Equal(t, "public_form", changes["source"], "the audit trail names the public form")
}

// TestReturnLifecycle_CreatePublic_UnknownOrderRejected verifies the public path still
// refuses a return for an order the tenant does not have.
func TestReturnLifecycle_CreatePublic_UnknownOrderRejected(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	returns, _ := newReturnServiceWithRestock(appPool)

	_, err := returns.CreatePublic(ctx, tenant, service.PublicReturnInput{
		OrderID:       uuid.New(),
		Reason:        "damaged",
		CustomerEmail: "nobody@example.com",
		ReturnToken:   "tok-" + uuid.New().String()[:12],
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order not found")
	assert.Zero(t, countAudit(t, ctx, tenant, "return.created"))
}

// seedReturnForOrder inserts a return in the given status with the given items.
func seedReturnForOrder(t *testing.T, ctx context.Context, tenant, order uuid.UUID, status string, items json.RawMessage) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO returns (id, tenant_id, order_id, status, reason, items, refund_amount)
		 VALUES ($1,$2,$3,$4,'damaged',$5,0)`,
		id, tenant, order, status, items)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM returns WHERE id = $1", id) })
	return id
}

// TestReturnLifecycle_ReceivedRestocks is the CORR-06 regression: goods coming back
// used to be lost to inventory forever. Marking the return received now puts the
// returned quantities back into warehouse stock.
func TestReturnLifecycle_ReceivedRestocks(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	returns, orders := newReturnServiceWithRestock(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Returnable Widget", 40, false)
	seedStock(t, ctx, tenant, warehouse, product, 10, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":2,"price":40}]`, product))
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Returner", Items: items, TotalAmount: 80, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, orders.RestockItems(ctx, tenant, nil), "no items is a no-op")

	// Ship it so the stock actually leaves: 10 -> 8.
	_, err = orders.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: model.OrderStatusShipped, Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		q, r := readStock(t, ctx, warehouse, product)
		return q == 8 && r == 0
	}, 4*time.Second, 50*time.Millisecond)

	ret := seedReturnForOrder(t, ctx, tenant, order.ID, "approved", items)
	_, err = returns.TransitionStatus(ctx, tenant, ret,
		model.ReturnStatusRequest{Status: "received"}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 10, qty, "the 2 returned units are sellable again")
	assert.Zero(t, reserved, "restocking does not create reservations")

	// The new availability is pushed on: the trigger type must be one the stock owner
	// accepts, or the propagation is dropped with a warning.
	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, superPool.QueryRow(ctx,
			`SELECT COUNT(*) FROM stock_sync_events WHERE product_id = $1 AND trigger_type = 'return_restocked'`,
			product).Scan(&n))
		return n == 1
	}, 4*time.Second, 50*time.Millisecond, "the restock is recorded as a stock change")
}

// TestReturnLifecycle_RestockPolicyOff verifies a tenant that reconciles returned goods
// by hand can turn restocking off.
func TestReturnLifecycle_RestockPolicyOff(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	setTenantSettings(t, ctx, tenant, `{"returns":{"restock_on":"off"}}`)
	returns, orders := newReturnServiceWithRestock(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "No Restock Widget", 40, false)
	seedStock(t, ctx, tenant, warehouse, product, 6, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":2,"price":40}]`, product))
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "No Restock", Items: items, TotalAmount: 80, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	ret := seedReturnForOrder(t, ctx, tenant, order.ID, "approved", items)
	_, err = returns.TransitionStatus(ctx, tenant, ret,
		model.ReturnStatusRequest{Status: "received"}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	qty, _ := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 6, qty, "policy off -> quantity untouched")
}

// TestReturnLifecycle_RestockPolicyOnRefunded verifies the restock point is
// configurable: with restock_on=refunded, entering "received" must not restock and
// entering "refunded" must.
func TestReturnLifecycle_RestockPolicyOnRefunded(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	setTenantSettings(t, ctx, tenant, `{"returns":{"restock_on":"refunded"}}`)
	returns, orders := newReturnServiceWithRestock(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Refund Restock Widget", 40, false)
	seedStock(t, ctx, tenant, warehouse, product, 5, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":1,"price":40}]`, product))
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Refund Restock", Items: items, TotalAmount: 40, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	ret := seedReturnForOrder(t, ctx, tenant, order.ID, "approved", items)
	_, err = returns.TransitionStatus(ctx, tenant, ret, model.ReturnStatusRequest{Status: "received"}, uuid.Nil, "")
	require.NoError(t, err)
	qty, _ := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 5, qty, "received is not the configured restock point")

	_, err = returns.TransitionStatus(ctx, tenant, ret, model.ReturnStatusRequest{Status: "refunded"}, uuid.Nil, "")
	require.NoError(t, err)
	qty, _ = readStock(t, ctx, warehouse, product)
	assert.Equal(t, 6, qty, "refunded restocks under this policy")
}

// TestReturnLifecycle_ReturnWithoutItemsRestocksNothing verifies the conservative rule:
// what physically came back is only knowable from the return's own lines, so a return
// that lists none restocks none (rather than guessing the whole order).
func TestReturnLifecycle_ReturnWithoutItemsRestocksNothing(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	returns, orders := newReturnServiceWithRestock(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Unlisted Widget", 40, false)
	seedStock(t, ctx, tenant, warehouse, product, 7, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":3,"price":40}]`, product))
	order, err := orders.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Unlisted", Items: items, TotalAmount: 120, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	ret := seedReturnForOrder(t, ctx, tenant, order.ID, "approved", json.RawMessage(`[]`))
	_, err = returns.TransitionStatus(ctx, tenant, ret, model.ReturnStatusRequest{Status: "received"}, uuid.Nil, "")
	require.NoError(t, err)

	qty, _ := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 7, qty, "a return listing no items restocks nothing")
}
