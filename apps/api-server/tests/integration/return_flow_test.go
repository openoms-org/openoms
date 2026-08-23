//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func newReturnService(pool *pgxpool.Pool) *service.ReturnService {
	svc := service.NewReturnService(
		repository.NewReturnRepository(), repository.NewOrderRepository(),
		repository.NewAuditRepository(), pool, nil)
	return svc
}

// advanceReturn walks a return through the status graph to the target status.
func advanceReturn(t *testing.T, ctx context.Context, svc *service.ReturnService, tenant, returnID uuid.UUID, statuses ...string) {
	t.Helper()
	for _, status := range statuses {
		_, err := svc.TransitionStatus(ctx, tenant, returnID,
			model.ReturnStatusRequest{Status: status}, uuid.Nil, "127.0.0.1")
		require.NoError(t, err, "transition to %s", status)
	}
}

// TestPublicReturn_CreateGoesThroughService is the CORR-05 regression: the public
// endpoint inserted the row with the repository directly, so a customer-submitted
// return left no audit trail and fired no webhook or automation event — it was
// invisible to every rule a tenant had configured for returns.
func TestPublicReturn_CreateGoesThroughService(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)
	returnSvc := newReturnService(appPool)

	email := "public-buyer-" + uuid.New().String()[:8] + "@example.com"
	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Public Buyer", CustomerEmail: &email, TotalAmount: 99, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	h := handler.NewPublicReturnHandler(appPool, returnSvc)
	body := fmt.Sprintf(`{"order_id":%q,"email":%q,"reason":"damaged in transit","notes":"box was wet"}`,
		order.ID, strings.ToUpper(email))
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.CreatePublicReturn(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp struct {
		ID          uuid.UUID `json:"id"`
		Status      string    `json:"status"`
		ReturnToken string    `json:"return_token"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "requested", resp.Status)
	assert.NotEmpty(t, resp.ReturnToken, "the customer gets a token to track the return")

	var storedEmail, storedNotes, storedToken *string
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT customer_email, customer_notes, return_token FROM returns WHERE id = $1`, resp.ID).
		Scan(&storedEmail, &storedNotes, &storedToken))
	require.NotNil(t, storedEmail)
	assert.Equal(t, email, *storedEmail, "the email is normalised to lower case")
	require.NotNil(t, storedNotes)
	assert.Equal(t, "box was wet", *storedNotes)
	require.NotNil(t, storedToken)
	assert.Equal(t, resp.ReturnToken, *storedToken)

	assert.Equal(t, 1, countAudit(t, ctx, resp.ID, "return.created"),
		"a customer-submitted return is audited like an internally created one")
}

// TestPublicReturn_RejectsMismatchedEmail keeps the ownership check on the public path:
// knowing an order id must not be enough to open a return against it.
func TestPublicReturn_RejectsMismatchedEmail(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)

	email := "owner-" + uuid.New().String()[:8] + "@example.com"
	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Owner", CustomerEmail: &email, TotalAmount: 10, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	h := handler.NewPublicReturnHandler(appPool, newReturnService(appPool))
	body := fmt.Sprintf(`{"order_id":%q,"email":"stranger@example.com","reason":"mine now"}`, order.ID)
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var count int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM returns WHERE order_id = $1`, order.ID).Scan(&count))
	assert.Equal(t, 0, count, "no return row is created")
}

// TestReturnRestock_ReceivedCreditsStock is the CORR-06 regression: no return status
// ever put stock back, so goods physically returned to the warehouse stayed invisible
// and the product looked out of stock until someone corrected it by hand.
func TestReturnRestock_ReceivedCreditsStock(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)
	returnSvc := newReturnService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Returnable Widget", 60, false)
	seedStock(t, ctx, tenant, warehouse, product, 20, 0)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":%q,"quantity":3,"price":60}]`, product))
	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Return Buyer", Items: items, TotalAmount: 180, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, err = orderSvc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: "shipped", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		q, r := readStock(t, ctx, warehouse, product)
		return q == 17 && r == 0
	}, 4*time.Second, 50*time.Millisecond, "the shipment leaves 17 on hand")

	ret, err := returnSvc.Create(ctx, tenant, model.CreateReturnRequest{
		OrderID: order.ID, Reason: "changed mind", Items: items, RefundAmount: 180,
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	advanceReturn(t, ctx, returnSvc, tenant, ret.ID, "approved")
	qty, _ := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 17, qty, "approving a return moves no goods — nothing has arrived yet")

	advanceReturn(t, ctx, returnSvc, tenant, ret.ID, "received")
	qty, reserved := readStock(t, ctx, warehouse, product)
	assert.Equal(t, 20, qty, "receiving the parcel puts the goods back on hand")
	assert.Equal(t, 0, reserved, "restocking does not reserve anything")

	// Refunding is the money event; the goods are already back.
	advanceReturn(t, ctx, returnSvc, tenant, ret.ID, "refunded")
	qty, _ = readStock(t, ctx, warehouse, product)
	assert.Equal(t, 20, qty, "the refund must not credit the stock a second time")
}

// TestReturnRestock_CreditsMatchingVariantRow checks the restock follows the same
// variant preference as the reserve/ship path.
func TestReturnRestock_CreditsMatchingVariantRow(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)
	returnSvc := newReturnService(appPool)
	warehouse := seedWarehouse(t, ctx, tenant)
	product := seedProduct(t, ctx, tenant, "Variant Returnable", 45, false)
	red := seedVariant(t, ctx, tenant, product, "Red")
	blue := seedVariant(t, ctx, tenant, product, "Blue")
	seedVariantStock(t, ctx, tenant, warehouse, product, red, 8, 0)
	seedVariantStock(t, ctx, tenant, warehouse, product, blue, 8, 0)

	items := json.RawMessage(fmt.Sprintf(
		`[{"product_id":%q,"variant_id":%q,"quantity":2,"price":45}]`, product, red))
	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Variant Returner", Items: items, TotalAmount: 90, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	_, err = orderSvc.TransitionStatus(ctx, tenant, order.ID,
		model.StatusTransitionRequest{Status: "shipped", Force: true}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		q, _ := readVariantStock(t, ctx, warehouse, product, red)
		return q == 6
	}, 4*time.Second, 50*time.Millisecond)

	ret, err := returnSvc.Create(ctx, tenant, model.CreateReturnRequest{
		OrderID: order.ID, Reason: "wrong size", Items: items,
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	advanceReturn(t, ctx, returnSvc, tenant, ret.ID, "approved", "received")

	redQty, _ := readVariantStock(t, ctx, warehouse, product, red)
	blueQty, _ := readVariantStock(t, ctx, warehouse, product, blue)
	assert.Equal(t, 8, redQty, "the returned variant is credited")
	assert.Equal(t, 8, blueQty, "the sibling variant is untouched")
}

// TestReturnRestock_UntrackedProductIsSkipped covers a return for a product the tenant
// keeps no warehouse rows for: there is nothing to credit, and that must not fail the
// status change or block the refund.
func TestReturnRestock_UntrackedProductIsSkipped(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderSvc := newLifecycleOrderService(appPool)
	returnSvc := newReturnService(appPool)
	product := seedProduct(t, ctx, tenant, "Untracked Service", 500, false)

	items := json.RawMessage(fmt.Sprintf(`[{"product_id":%q,"quantity":1,"price":500}]`, product))
	order, err := orderSvc.Create(ctx, tenant, model.CreateOrderRequest{
		CustomerName: "Service Buyer", Items: items, TotalAmount: 500, Currency: "PLN",
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)

	ret, err := returnSvc.Create(ctx, tenant, model.CreateReturnRequest{
		OrderID: order.ID, Reason: "not needed", Items: items,
	}, uuid.Nil, "127.0.0.1")
	require.NoError(t, err)
	advanceReturn(t, ctx, returnSvc, tenant, ret.ID, "approved", "received", "refunded")

	var status, paymentStatus string
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT r.status, o.payment_status FROM returns r JOIN orders o ON o.id = r.order_id WHERE r.id = $1`,
		ret.ID).Scan(&status, &paymentStatus))
	assert.Equal(t, "refunded", status)
	assert.Equal(t, "refunded", paymentStatus, "the refund still syncs to the order")
}
