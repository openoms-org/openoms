//go:build integration

package erli_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	erli "github.com/openoms-org/openoms/packages/erli-go-sdk"
)

var (
	erliToken      string
	sandboxOfferID string
	sandboxOrderID string
	createdOfferID string
)

func TestMain(m *testing.M) {
	erliToken = os.Getenv("ERLI_SANDBOX_TOKEN")
	if erliToken == "" {
		fmt.Println("ERLI_SANDBOX_TOKEN not set, skipping Erli integration tests")
		os.Exit(0)
	}
	sandboxOfferID = os.Getenv("ERLI_SANDBOX_OFFER_ID")
	sandboxOrderID = os.Getenv("ERLI_SANDBOX_ORDER_ID")
	os.Exit(m.Run())
}

func newSandboxClient() *erli.Client {
	return erli.NewClient(erliToken, erli.WithSandbox())
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// effectiveOfferID returns sandboxOfferID (env) if set, otherwise createdOfferID
// (created by TestIntegration_OffersCreate earlier in this run).
func effectiveOfferID() string {
	if sandboxOfferID != "" {
		return sandboxOfferID
	}
	return createdOfferID
}

// resolveOrderID returns sandboxOrderID if set; otherwise fetches the first
// available order from the sandbox. Returns ("", false) when no order exists.
func resolveOrderID(t *testing.T, c *erli.Client) (string, bool) {
	t.Helper()
	if sandboxOrderID != "" {
		return sandboxOrderID, true
	}
	resp, err := c.Orders.List(testCtx(t), "")
	if err != nil {
		t.Fatalf("Orders.List() error while resolving order ID: %v", err)
	}
	if len(resp.Data) == 0 {
		return "", false
	}
	return resp.Data[0].ID, true
}

// --- Group 1: Auth / Connectivity ---

func TestIntegration_InvalidToken(t *testing.T) {
	c := erli.NewClient("definitely-invalid-token", erli.WithSandbox())
	_, err := c.Orders.List(testCtx(t), "")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
	var apiErr *erli.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *erli.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 && apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 401 or 403", apiErr.StatusCode)
	}
}

// --- Group 2: Orders ---

func TestIntegration_OrdersList(t *testing.T) {
	c := newSandboxClient()
	resp, err := c.Orders.List(testCtx(t), "")
	if err != nil {
		t.Fatalf("Orders.List() error: %v", err)
	}
	if resp == nil {
		t.Fatal("Orders.List() returned nil response")
	}
	// Data must be a non-nil slice (empty is acceptable for a fresh sandbox).
	if resp.Data == nil {
		t.Error("Orders.List() Data is nil, want non-nil slice")
	}
}

func TestIntegration_OrdersListPagination(t *testing.T) {
	c := newSandboxClient()
	resp, err := c.Orders.List(testCtx(t), "")
	if err != nil {
		t.Fatalf("Orders.List() first page error: %v", err)
	}
	if resp.Meta.NextCursor == "" {
		t.Skip("no next_cursor in first page response, skipping pagination test")
	}
	resp2, err := c.Orders.List(testCtx(t), resp.Meta.NextCursor)
	if err != nil {
		t.Fatalf("Orders.List() second page error: %v", err)
	}
	if resp2 == nil {
		t.Fatal("Orders.List() second page returned nil response")
	}
	if resp2.Data == nil {
		t.Error("Orders.List() second page Data is nil")
	}
}

func TestIntegration_OrdersGetExisting(t *testing.T) {
	c := newSandboxClient()
	orderID, ok := resolveOrderID(t, c)
	if !ok {
		t.Skip("no orders in sandbox and ERLI_SANDBOX_ORDER_ID not set, skipping")
	}
	order, err := c.Orders.Get(testCtx(t), orderID)
	if err != nil {
		t.Fatalf("Orders.Get(%q) error: %v", orderID, err)
	}
	if order.ID == "" {
		t.Error("order.ID is empty")
	}
	if order.Status == "" {
		t.Error("order.Status is empty")
	}
}

func TestIntegration_OrdersGetNotFound(t *testing.T) {
	c := newSandboxClient()
	_, err := c.Orders.Get(testCtx(t), "nonexistent-order-id-000")
	if err == nil {
		t.Fatal("expected error for non-existent order, got nil")
	}
	var apiErr *erli.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *erli.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// --- Group 3: Offers ---

// TestIntegration_OffersCreate creates a minimal offer in sandbox and stores
// the returned ID in createdOfferID for subsequent update tests.
func TestIntegration_OffersCreate(t *testing.T) {
	if sandboxOfferID != "" {
		t.Skip("ERLI_SANDBOX_OFFER_ID already set via env, skipping create")
	}
	c := newSandboxClient()
	req := erli.CreateOfferRequest{
		Title: "OpenOMS Integration Test Offer",
		Price: 9.99,
		Stock: 1,
	}
	offerID, err := c.Offers.Create(testCtx(t), req)
	if err != nil {
		// Log raw error — helps diagnose schema mismatches between models.go and real API.
		t.Fatalf("Offers.Create() error: %v\nhint: check if CreateOfferRequest is missing required fields per sandbox docs", err)
	}
	if offerID == "" {
		t.Fatal("Offers.Create() returned empty offer ID")
	}
	createdOfferID = offerID
	t.Logf("created sandbox offer ID: %s", offerID)
}

func TestIntegration_OffersUpdateStock(t *testing.T) {
	offerID := effectiveOfferID()
	if offerID == "" {
		t.Skip("no offer ID available — set ERLI_SANDBOX_OFFER_ID or run OffersCreate first")
	}
	c := newSandboxClient()
	if err := c.Offers.UpdateStock(testCtx(t), offerID, 10); err != nil {
		t.Fatalf("Offers.UpdateStock(%q, 10) error: %v", offerID, err)
	}
}

func TestIntegration_OffersUpdatePrice(t *testing.T) {
	offerID := effectiveOfferID()
	if offerID == "" {
		t.Skip("no offer ID available — set ERLI_SANDBOX_OFFER_ID or run OffersCreate first")
	}
	c := newSandboxClient()
	if err := c.Offers.UpdatePrice(testCtx(t), offerID, 19.99); err != nil {
		t.Fatalf("Offers.UpdatePrice(%q, 19.99) error: %v", offerID, err)
	}
}

func TestIntegration_OffersUpdateStockInvalid(t *testing.T) {
	offerID := effectiveOfferID()
	if offerID == "" {
		t.Skip("no offer ID available — set ERLI_SANDBOX_OFFER_ID or run OffersCreate first")
	}
	c := newSandboxClient()
	err := c.Offers.UpdateStock(testCtx(t), offerID, -1)
	if err == nil {
		t.Fatal("expected error for negative stock value, got nil")
	}
	var apiErr *erli.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *erli.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 && apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 400 or 422", apiErr.StatusCode)
	}
}

// --- Group 4: Status mapping verification ---

// TestIntegration_OrderStatusValues iterates sandbox orders and warns about
// statuses not present in statusmap.go. Reports but does not fail — unknown
// statuses indicate statusmap.go needs updating.
func TestIntegration_OrderStatusValues(t *testing.T) {
	c := newSandboxClient()
	resp, err := c.Orders.List(testCtx(t), "")
	if err != nil {
		t.Fatalf("Orders.List() error: %v", err)
	}
	unknown := map[string]struct{}{}
	for _, order := range resp.Data {
		if _, ok := erli.MapStatus(order.Status); !ok {
			unknown[order.Status] = struct{}{}
		}
	}
	for status := range unknown {
		t.Logf("WARNING: unknown Erli status %q not in statusmap.go — add it if this is a real status", status)
	}
}

// --- Group 5: Response structure validation ---

func TestIntegration_OrderFieldsNotEmpty(t *testing.T) {
	c := newSandboxClient()
	orderID, ok := resolveOrderID(t, c)
	if !ok {
		t.Skip("no orders in sandbox, skipping field validation")
	}
	order, err := c.Orders.Get(testCtx(t), orderID)
	if err != nil {
		t.Fatalf("Orders.Get(%q) error: %v", orderID, err)
	}
	if order.ID == "" {
		t.Error("order.ID is empty")
	}
	if order.Currency == "" {
		t.Logf("WARNING: order.Currency is empty — check JSON tag (currency)")
	}
	if order.TotalAmount <= 0 {
		t.Logf("WARNING: order.TotalAmount = %f — check JSON tag (total_amount) or sandbox data", order.TotalAmount)
	}
	if order.BuyerEmail == "" {
		t.Logf("WARNING: order.BuyerEmail is empty — check JSON tag (buyer_email)")
	}
}

func TestIntegration_OrderAddressStructure(t *testing.T) {
	c := newSandboxClient()
	orderID, ok := resolveOrderID(t, c)
	if !ok {
		t.Skip("no orders in sandbox, skipping address validation")
	}
	order, err := c.Orders.Get(testCtx(t), orderID)
	if err != nil {
		t.Fatalf("Orders.Get(%q) error: %v", orderID, err)
	}
	if order.Address.City == "" {
		t.Logf("WARNING: Address.City is empty — check JSON tag (delivery_address.city)")
	}
	if order.Address.PostCode == "" {
		t.Logf("WARNING: Address.PostCode is empty — check JSON tag (delivery_address.post_code)")
	}
	if order.Address.Country == "" {
		t.Logf("WARNING: Address.Country is empty — check JSON tag (delivery_address.country)")
	}
}

func TestIntegration_OrderItemsStructure(t *testing.T) {
	c := newSandboxClient()
	orderID, ok := resolveOrderID(t, c)
	if !ok {
		t.Skip("no orders in sandbox, skipping items validation")
	}
	order, err := c.Orders.Get(testCtx(t), orderID)
	if err != nil {
		t.Fatalf("Orders.Get(%q) error: %v", orderID, err)
	}
	if len(order.Items) == 0 {
		t.Logf("WARNING: order has no items — check JSON tag (items) or sandbox data")
		return
	}
	for i, item := range order.Items {
		if item.ID == "" {
			t.Errorf("item[%d].ID is empty", i)
		}
		if item.Name == "" {
			t.Logf("WARNING: item[%d].Name is empty — check JSON tag (name)", i)
		}
		if item.Quantity <= 0 {
			t.Logf("WARNING: item[%d].Quantity = %d — check JSON tag (quantity)", i, item.Quantity)
		}
		if item.Price <= 0 {
			t.Logf("WARNING: item[%d].Price = %f — check JSON tag (unit_price)", i, item.Price)
		}
	}
}
