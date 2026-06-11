//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// newLifecycleOrderService builds a fully-wired OrderService (customer link/stats, B2B
// pricing, loyalty accrual, bundle stock) against the RLS-scoped app pool. fulfillment is
// the disabled no-op service.
func newLifecycleOrderService(pool *pgxpool.Pool) *service.OrderService {
	auditRepo := repository.NewAuditRepository()
	productRepo := repository.NewProductRepository()
	fulfillment := service.NewFulfillmentService(false,
		repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())

	svc := service.NewOrderService(
		repository.NewOrderRepository(), auditRepo, repository.NewTenantRepository(pool),
		pool, nil, nil, fulfillment)
	svc.SetCustomerRepo(repository.NewCustomerRepository())
	svc.SetWarehouseStockRepo(repository.NewWarehouseStockRepository())
	svc.SetBundleService(service.NewBundleService(repository.NewBundleRepository(), productRepo, auditRepo, pool))
	svc.SetPriceListService(service.NewPriceListService(repository.NewPriceListRepository(), productRepo, auditRepo, pool))
	svc.SetLoyaltyService(service.NewLoyaltyService(repository.NewLoyaltyRepository(), auditRepo, pool, slog.Default()))
	return svc
}

func seedProduct(t *testing.T, ctx context.Context, tenant uuid.UUID, name string, price float64, isBundle bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO products (id, tenant_id, name, price, is_bundle) VALUES ($1,$2,$3,$4,$5)`,
		id, tenant, name, price, isBundle)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id) })
	return id
}

func seedWarehouse(t *testing.T, ctx context.Context, tenant uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO warehouses (id, tenant_id, name, is_default) VALUES ($1,$2,$3,true)`,
		id, tenant, "WH "+id.String()[:8])
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM warehouses WHERE id = $1", id) })
	return id
}

func seedStock(t *testing.T, ctx context.Context, tenant, warehouse, product uuid.UUID, qty, reserved int) {
	t.Helper()
	_, err := superPool.Exec(ctx,
		`INSERT INTO warehouse_stock (tenant_id, warehouse_id, product_id, quantity, reserved) VALUES ($1,$2,$3,$4,$5)`,
		tenant, warehouse, product, qty, reserved)
	require.NoError(t, err)
}

func readStock(t *testing.T, ctx context.Context, warehouse, product uuid.UUID) (qty, reserved int) {
	t.Helper()
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT quantity, reserved FROM warehouse_stock WHERE warehouse_id = $1 AND product_id = $2`,
		warehouse, product).Scan(&qty, &reserved))
	return qty, reserved
}

// --- Order -> customer linkage + stats ---

func TestOrderLifecycle_Create_LinksCustomerAndIncrementsStats(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	email := "buyer-" + uuid.New().String()[:8] + "@example.com"
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName:  "Jan Kowalski",
		CustomerEmail: &email,
		TotalAmount:   250,
		Currency:      "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, order.CustomerID, "order is linked to a freshly-created customer")

	var name string
	var totalOrders int
	var totalSpent float64
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT name, total_orders, total_spent FROM customers WHERE id = $1`, *order.CustomerID).
		Scan(&name, &totalOrders, &totalSpent))
	assert.Equal(t, "Jan Kowalski", name)
	assert.Equal(t, 1, totalOrders)
	assert.InDelta(t, 250.0, totalSpent, 0.001)

	// A second order with the SAME email reuses the customer and bumps stats to 2 / 450.
	_, err = svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Jan Kowalski", CustomerEmail: &email, TotalAmount: 200, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT total_orders, total_spent FROM customers WHERE id = $1`, *order.CustomerID).
		Scan(&totalOrders, &totalSpent))
	assert.Equal(t, 2, totalOrders, "same email reused, not duplicated")
	assert.InDelta(t, 450.0, totalSpent, 0.001)
}

func TestOrderLifecycle_Create_NoEmail_NoCustomerLink(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "No Email", TotalAmount: 10, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	assert.Nil(t, order.CustomerID, "no email -> no customer link")
}

// --- B2B price list applied on create ---

func TestOrderLifecycle_Create_AppliesB2BPriceList(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	product := seedProduct(t, ctx, tenant, "Widget", 100, false)

	// Price list: 10% percentage discount, with an item for the product.
	plID := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO price_lists (id, tenant_id, name, currency, discount_type, active) VALUES ($1,$2,'B2B','PLN','percentage',true)`,
		plID, tenant)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM price_lists WHERE id = $1", plID) })
	_, err = superPool.Exec(ctx,
		`INSERT INTO price_list_items (tenant_id, price_list_id, product_id, discount, min_quantity) VALUES ($1,$2,$3,10,1)`,
		tenant, plID, product)
	require.NoError(t, err)

	// Customer with that price list.
	custID := uuid.New()
	email := "b2b-" + uuid.New().String()[:8] + "@example.com"
	_, err = superPool.Exec(ctx,
		`INSERT INTO customers (id, tenant_id, name, email, price_list_id) VALUES ($1,$2,'ACME',$3,$4)`,
		custID, tenant, email, plID)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customers WHERE id = $1", custID) })

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","name":"Widget","quantity":2,"price":100}]`, product))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "ACME", CustomerEmail: &email, Items: items, TotalAmount: 200, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	// Effective price 90 (100 - 10%), total recomputed = 2 * 90 = 180.
	assert.InDelta(t, 180.0, order.TotalAmount, 0.001)
	var lines []map[string]any
	require.NoError(t, json.Unmarshal(order.Items, &lines))
	require.Len(t, lines, 1)
	assert.InDelta(t, 90.0, lines[0]["price"], 0.001)
	assert.Equal(t, "Widget", lines[0]["name"], "unknown line keys preserved")
	assert.Equal(t, custID.String(), order.CustomerID.String(), "links to the existing B2B customer")
}

func TestOrderLifecycle_Create_PriceListCurrencyMismatch_Skipped(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	product := seedProduct(t, ctx, tenant, "EurWidget", 100, false)
	plID := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO price_lists (id, tenant_id, name, currency, discount_type, active) VALUES ($1,$2,'EUR list','EUR','percentage',true)`,
		plID, tenant)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM price_lists WHERE id = $1", plID) })
	_, err = superPool.Exec(ctx,
		`INSERT INTO price_list_items (tenant_id, price_list_id, product_id, discount, min_quantity) VALUES ($1,$2,$3,50,1)`,
		tenant, plID, product)
	require.NoError(t, err)

	email := "eur-" + uuid.New().String()[:8] + "@example.com"
	custID := uuid.New()
	_, err = superPool.Exec(ctx,
		`INSERT INTO customers (id, tenant_id, name, email, price_list_id) VALUES ($1,$2,'EurCo',$3,$4)`,
		custID, tenant, email, plID)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customers WHERE id = $1", custID) })

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":"%s","quantity":1,"price":100}]`, product))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "EurCo", CustomerEmail: &email, Items: items, TotalAmount: 100, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	// Currency guard: PLN order is not priced with the EUR list, total unchanged.
	assert.InDelta(t, 100.0, order.TotalAmount, 0.001)
}

// --- Duplicate ---

func TestOrderLifecycle_Duplicate(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	ext := "EXT-123"
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		ExternalID: &ext, Source: "manual", CustomerName: "Dup Source",
		TotalAmount: 99, Currency: "PLN", Tags: []string{"vip"}, Priority: "high",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	dup, err := svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	assert.NotEqual(t, src.ID, dup.ID)
	assert.Nil(t, dup.ExternalID, "external id cleared on duplicate")
	assert.Equal(t, "new", dup.Status)
	assert.Equal(t, "Dup Source", dup.CustomerName)
	assert.InDelta(t, 99.0, dup.TotalAmount, 0.001)
	assert.Equal(t, []string{"vip"}, dup.Tags)
	assert.Equal(t, "high", dup.Priority)

	// Audit entry recorded with source_order_id.
	var changes map[string]string
	var raw []byte
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT changes FROM audit_log WHERE entity_id = $1 AND action = 'order.duplicated'`, dup.ID).Scan(&raw))
	require.NoError(t, json.Unmarshal(raw, &changes))
	assert.Equal(t, src.ID.String(), changes["source_order_id"])

	// Not found.
	_, err = svc.Duplicate(ctx, tenant, uuid.New(), 0, uuid.Nil, "127.0.0.1")
	assert.True(t, errors.Is(err, service.ErrOrderNotFound))
}

// TestOrderLifecycle_Duplicate_IncrementsLinkedCustomerStats verifies the duplicate
// (which persists the source order's customer_id) counts toward the linked customer's
// denormalized stats, keeping them equal to the RFM aggregate that counts every order row.
func TestOrderLifecycle_Duplicate_IncrementsLinkedCustomerStats(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)

	email := "dupstats-" + uuid.New().String()[:8] + "@example.com"
	src, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Dup Stats", CustomerEmail: &email, TotalAmount: 120, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, src.CustomerID, "source order links a customer")

	readStats := func() (int, float64) {
		var n int
		var spent float64
		require.NoError(t, superPool.QueryRow(ctx,
			`SELECT total_orders, total_spent FROM customers WHERE id = $1`, *src.CustomerID).Scan(&n, &spent))
		return n, spent
	}
	n, spent := readStats()
	assert.Equal(t, 1, n)
	assert.InDelta(t, 120.0, spent, 0.001)

	dup, err := svc.Duplicate(ctx, tenant, src.ID, 0, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, dup.CustomerID)
	assert.Equal(t, src.CustomerID.String(), dup.CustomerID.String(), "duplicate keeps the customer link")

	n, spent = readStats()
	assert.Equal(t, 2, n, "duplicate counts toward customer order count")
	assert.InDelta(t, 240.0, spent, 0.001, "duplicate counts toward customer spend")
}

// NOTE: the bulk-transition duplicate-id dedupe (BulkTransitionStatus) is covered by the
// pure unit test TestDedupeOrderIDs in internal/service. A DB-level bulk test is not
// feasible here because BulkTransitionStatus calls OrderRepository.FindByIDs, whose
// `id = ANY($1)` binds a []uuid.UUID; pgx cannot encode a uuid slice under the harness's
// forced simple-protocol mode (production uses the extended protocol where it works). The
// single-order firstShip gate is DB-proven by TestOrderLifecycle_BundleStock_*; dedupe
// guarantees the bulk path reads the firstShip snapshot once per unique order.

// --- Loyalty accrual: exactly-once ledger ---

func seedPointsProgram(t *testing.T, ctx context.Context, tenant uuid.UUID, perPLN int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	cfg := fmt.Sprintf(`{"points_per_pln":%d}`, perPLN)
	_, err := superPool.Exec(ctx,
		`INSERT INTO loyalty_programs (id, tenant_id, name, status, program_type, config) VALUES ($1,$2,'Pts','active','points',$3)`,
		id, tenant, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM loyalty_programs WHERE id = $1", id) })
	return id
}

func seedCustomer(t *testing.T, ctx context.Context, tenant uuid.UUID, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO customers (id, tenant_id, name, email) VALUES ($1,$2,'Loyal',$3)`, id, tenant, email)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customers WHERE id = $1", id) })
	return id
}

func seedOrder(t *testing.T, ctx context.Context, tenant uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,'Ledger Order')`, id, tenant)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id) })
	return id
}

// TestLoyalty_AwardPointsForOrder_Idempotent calls the accrual directly (synchronous) twice
// with the same order id and asserts the points/order_count are incremented exactly once and
// exactly one ledger row exists — the ON CONFLICT guard.
func TestLoyalty_AwardPointsForOrder_Idempotent(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	program := seedPointsProgram(t, ctx, tenant, 1)
	custID := seedCustomer(t, ctx, tenant, "loyal-"+uuid.New().String()[:8]+"@example.com")
	orderID := seedOrder(t, ctx, tenant)

	loyalty := service.NewLoyaltyService(repository.NewLoyaltyRepository(), repository.NewAuditRepository(), appPool, slog.Default())

	loyalty.AwardPointsForOrder(ctx, tenant, custID, orderID, 100)
	loyalty.AwardPointsForOrder(ctx, tenant, custID, orderID, 100) // replay — must be a no-op

	var points, orderCount int
	var totalSpent float64
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT points_balance, order_count, total_spent FROM customer_loyalty WHERE customer_id = $1 AND program_id = $2`,
		custID, program).Scan(&points, &orderCount, &totalSpent))
	assert.Equal(t, 100, points, "points awarded exactly once")
	assert.Equal(t, 1, orderCount)
	assert.InDelta(t, 100.0, totalSpent, 0.001)

	var ledgerRows int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM loyalty_transactions WHERE order_id = $1 AND program_id = $2`, orderID, program).Scan(&ledgerRows))
	assert.Equal(t, 1, ledgerRows, "exactly one accrual ledger row")
}

// TestRecordOrderAccrual_OnConflictDoNothing exercises the repository guard directly.
func TestRecordOrderAccrual_OnConflictDoNothing(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	program := seedPointsProgram(t, ctx, tenant, 1)
	custID := seedCustomer(t, ctx, tenant, "acc-"+uuid.New().String()[:8]+"@example.com")
	orderID := seedOrder(t, ctx, tenant)
	repo := repository.NewLoyaltyRepository()

	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		inserted, err := repo.RecordOrderAccrual(ctx, tx, &model.LoyaltyTransaction{
			ID: uuid.New(), TenantID: tenant, CustomerID: custID, ProgramID: program, OrderID: &orderID, Amount: 50,
		})
		require.NoError(t, err)
		assert.True(t, inserted, "first insert claims the slot")

		inserted, err = repo.RecordOrderAccrual(ctx, tx, &model.LoyaltyTransaction{
			ID: uuid.New(), TenantID: tenant, CustomerID: custID, ProgramID: program, OrderID: &orderID, Amount: 50,
		})
		require.NoError(t, err)
		assert.False(t, inserted, "second insert conflicts -> no-op")
		return nil
	}))
}

// TestOrderLifecycle_Completed_AccruesLoyaltyOnce drives the async path: forcing an order to
// completed accrues points once; a forced re-completion does not double-award.
func TestOrderLifecycle_Completed_AccruesLoyaltyOnce(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	program := seedPointsProgram(t, ctx, tenant, 1)
	svc := newLifecycleOrderService(appPool)

	email := "complete-" + uuid.New().String()[:8] + "@example.com"
	custID := seedCustomer(t, ctx, tenant, email)

	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Loyal", CustomerEmail: &email, TotalAmount: 100, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, order.CustomerID)
	assert.Equal(t, custID.String(), order.CustomerID.String())

	_, err = svc.TransitionStatus(ctx, tenant, order.ID, model.StatusTransitionRequest{Status: "completed", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	pointsOf := func() int {
		var p int
		_ = superPool.QueryRow(ctx,
			`SELECT COALESCE((SELECT points_balance FROM customer_loyalty WHERE customer_id=$1 AND program_id=$2),0)`,
			custID, program).Scan(&p)
		return p
	}
	require.Eventually(t, func() bool { return pointsOf() == 100 }, 4*time.Second, 50*time.Millisecond,
		"points accrued once on completion")

	// Force a re-completion — the ledger guard makes it a no-op.
	_, err = svc.TransitionStatus(ctx, tenant, order.ID, model.StatusTransitionRequest{Status: "completed", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 100, pointsOf(), "re-completion does not double-award")

	var ledgerRows int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM loyalty_transactions WHERE order_id = $1 AND program_id = $2`, order.ID, program).Scan(&ledgerRows))
	assert.Equal(t, 1, ledgerRows)
}

// --- Bundle stock decrement + firstShip idempotency ---

func TestOrderLifecycle_BundleStock_ShipDecrementsComponentsOnce(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newLifecycleOrderService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)

	bundle := seedProduct(t, ctx, tenant, "Bundle", 0, true)
	compA := seedProduct(t, ctx, tenant, "CompA", 10, false)
	compB := seedProduct(t, ctx, tenant, "CompB", 10, false)
	normal := seedProduct(t, ctx, tenant, "Normal", 10, false)

	// Bundle: 1x CompA + 3x CompB.
	_, err := superPool.Exec(ctx,
		`INSERT INTO product_bundles (tenant_id, bundle_product_id, component_product_id, quantity, position) VALUES ($1,$2,$3,1,0)`,
		tenant, bundle, compA)
	require.NoError(t, err)
	_, err = superPool.Exec(ctx,
		`INSERT INTO product_bundles (tenant_id, bundle_product_id, component_product_id, quantity, position) VALUES ($1,$2,$3,3,1)`,
		tenant, bundle, compB)
	require.NoError(t, err)

	seedStock(t, ctx, tenant, warehouse, compA, 100, 0)
	seedStock(t, ctx, tenant, warehouse, compB, 100, 0)
	seedStock(t, ctx, tenant, warehouse, normal, 50, 0)

	// Order: 2x bundle + 4x normal. Create reserves component + normal stock (synchronous).
	items := json.RawMessage(fmt.Sprintf(
		`[{"product_id":"%s","quantity":2,"price":0},{"product_id":"%s","quantity":4,"price":10}]`, bundle, normal))
	order, err := svc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Bundle Buyer", Items: items, TotalAmount: 40, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	// After create: reservations raised for components (1*2, 3*2) and normal (4).
	_, ra := readStock(t, ctx, warehouse, compA)
	_, rb := readStock(t, ctx, warehouse, compB)
	_, rn := readStock(t, ctx, warehouse, normal)
	assert.Equal(t, 2, ra, "compA reserved")
	assert.Equal(t, 6, rb, "compB reserved")
	assert.Equal(t, 4, rn, "normal reserved")

	// Ship (force new->shipped): decrement components + normal exactly once.
	_, err = svc.TransitionStatus(ctx, tenant, order.ID, model.StatusTransitionRequest{Status: "shipped", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		qa, _ := readStock(t, ctx, warehouse, compA)
		qb, _ := readStock(t, ctx, warehouse, compB)
		qn, _ := readStock(t, ctx, warehouse, normal)
		return qa == 98 && qb == 94 && qn == 46
	}, 4*time.Second, 50*time.Millisecond, "first ship decrements component + normal quantities once")

	qa, rA := readStock(t, ctx, warehouse, compA)
	qb, rB := readStock(t, ctx, warehouse, compB)
	qn, rN := readStock(t, ctx, warehouse, normal)
	assert.Equal(t, 0, rA)
	assert.Equal(t, 0, rB)
	assert.Equal(t, 0, rN)

	// The bundle itself never had a warehouse_stock row.
	var bundleStockRows int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM warehouse_stock WHERE product_id = $1`, bundle).Scan(&bundleStockRows))
	assert.Equal(t, 0, bundleStockRows)

	// Force re-ship: firstShip gate (shipped_at already set) -> NO second decrement.
	_, err = svc.TransitionStatus(ctx, tenant, order.ID, model.StatusTransitionRequest{Status: "shipped", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	qa2, _ := readStock(t, ctx, warehouse, compA)
	qb2, _ := readStock(t, ctx, warehouse, compB)
	qn2, _ := readStock(t, ctx, warehouse, normal)
	assert.Equal(t, qa, qa2, "re-ship does not double-decrement compA")
	assert.Equal(t, qb, qb2, "re-ship does not double-decrement compB")
	assert.Equal(t, qn, qn2, "re-ship does not double-decrement normal product")
}
