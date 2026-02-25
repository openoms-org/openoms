package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("client-id", "client-secret")

	if c.clientID != "client-id" {
		t.Errorf("clientID = %q, want %q", c.clientID, "client-id")
	}
	if c.clientSecret != "client-secret" {
		t.Errorf("clientSecret = %q, want %q", c.clientSecret, "client-secret")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Catalog == nil {
		t.Error("Catalog service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("id", "secret", WithSandbox())

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("id", "secret", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithTokens(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	c := NewClient("id", "secret", WithTokens("access-tok", "refresh-tok", exp))

	if c.accessToken != "access-tok" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "access-tok")
	}
	if c.refreshToken != "refresh-tok" {
		t.Errorf("refreshToken = %q, want %q", c.refreshToken, "refresh-tok")
	}
}

func TestWithRefreshToken(t *testing.T) {
	c := NewClient("id", "secret", WithRefreshToken("rt-123"))

	if c.refreshToken != "rt-123" {
		t.Errorf("refreshToken = %q, want %q", c.refreshToken, "rt-123")
	}
}

// newTestClient creates a test client with a valid non-expired token,
// so API calls don't trigger token refresh.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithTokens("test-access-token", "test-refresh-token", time.Now().Add(time.Hour)),
	)
}

func TestDoSetsAccessTokenHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("x-amz-access-token")
		if tok != "test-access-token" {
			t.Errorf("x-amz-access-token = %q, want %q", tok, "test-access-token")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
}

func TestOrdersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders/v0/orders" {
			t.Errorf("path = %q, want /orders/v0/orders", r.URL.Path)
		}
		if r.URL.Query().Get("CreatedAfter") != "2024-01-01T00:00:00Z" {
			t.Errorf("CreatedAfter = %q, want 2024-01-01T00:00:00Z", r.URL.Query().Get("CreatedAfter"))
		}
		if r.URL.Query().Get("MarketplaceIds") != "A1RKKUPIHCS9HS" {
			t.Errorf("MarketplaceIds = %q, want A1RKKUPIHCS9HS", r.URL.Query().Get("MarketplaceIds"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrdersResponse{
			Payload: OrdersPayload{
				Orders: []Order{
					{
						AmazonOrderID:  "305-1234567-8901234",
						PurchaseDate:   "2024-01-15T10:30:00Z",
						OrderStatus:    "Unshipped",
						OrderTotal:     &Money{Amount: "49.99", CurrencyCode: "PLN"},
						MarketplaceID:  "A1RKKUPIHCS9HS",
						FulfillmentChannel: "MFN",
						NumberOfItemsUnshipped: 1,
					},
				},
				NextToken: "next-page-token",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	resp, err := c.Orders.List(context.Background(), "2024-01-01T00:00:00Z", []string{"A1RKKUPIHCS9HS"}, "")
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
	if len(resp.Payload.Orders) != 1 {
		t.Fatalf("len(Orders) = %d, want 1", len(resp.Payload.Orders))
	}

	order := resp.Payload.Orders[0]
	if order.AmazonOrderID != "305-1234567-8901234" {
		t.Errorf("AmazonOrderID = %q, want %q", order.AmazonOrderID, "305-1234567-8901234")
	}
	if order.OrderStatus != "Unshipped" {
		t.Errorf("OrderStatus = %q, want %q", order.OrderStatus, "Unshipped")
	}
	if order.OrderTotal.Amount != "49.99" {
		t.Errorf("OrderTotal.Amount = %q, want %q", order.OrderTotal.Amount, "49.99")
	}
	if resp.Payload.NextToken != "next-page-token" {
		t.Errorf("NextToken = %q, want %q", resp.Payload.NextToken, "next-page-token")
	}
}

func TestOrdersListWithNextToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("NextToken") != "page-2-token" {
			t.Errorf("NextToken = %q, want page-2-token", r.URL.Query().Get("NextToken"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrdersResponse{
			Payload: OrdersPayload{
				Orders: []Order{},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.List(context.Background(), "", nil, "page-2-token")
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
}

func TestOrdersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders/v0/orders/305-1234567-8901234" {
			t.Errorf("path = %q, want /orders/v0/orders/305-1234567-8901234", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetOrderResponse{
			Payload: Order{
				AmazonOrderID:  "305-1234567-8901234",
				OrderStatus:    "Shipped",
				PurchaseDate:   "2024-01-15T10:30:00Z",
				MarketplaceID:  "A1RKKUPIHCS9HS",
				FulfillmentChannel: "MFN",
				ShippingAddress: &Address{
					Name:       "Jan Kowalski",
					City:       "Warszawa",
					PostalCode: "00-001",
					CountryCode: "PL",
				},
				NumberOfItemsShipped: 2,
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	order, err := c.Orders.Get(context.Background(), "305-1234567-8901234")
	if err != nil {
		t.Fatalf("Orders.Get error: %v", err)
	}
	if order.AmazonOrderID != "305-1234567-8901234" {
		t.Errorf("AmazonOrderID = %q, want %q", order.AmazonOrderID, "305-1234567-8901234")
	}
	if order.OrderStatus != "Shipped" {
		t.Errorf("OrderStatus = %q, want %q", order.OrderStatus, "Shipped")
	}
	if order.ShippingAddress == nil {
		t.Fatal("ShippingAddress is nil")
	}
	if order.ShippingAddress.City != "Warszawa" {
		t.Errorf("ShippingAddress.City = %q, want %q", order.ShippingAddress.City, "Warszawa")
	}
	if order.NumberOfItemsShipped != 2 {
		t.Errorf("NumberOfItemsShipped = %d, want 2", order.NumberOfItemsShipped)
	}
}

func TestOrdersGetItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/v0/orders/305-1234567-8901234/orderItems" {
			t.Errorf("path = %q, want /orders/v0/orders/305-1234567-8901234/orderItems", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrderItemsResponse{
			Payload: OrderItemsPayload{
				OrderItems: []OrderItem{
					{
						ASIN:            "B08N5WRWNW",
						SellerSKU:       "SKU-001",
						OrderItemID:     "item-001",
						Title:           "Widget Pro",
						QuantityOrdered: 2,
						QuantityShipped: 0,
						ItemPrice:       &Money{Amount: "49.99", CurrencyCode: "PLN"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	resp, err := c.Orders.GetItems(context.Background(), "305-1234567-8901234", "")
	if err != nil {
		t.Fatalf("Orders.GetItems error: %v", err)
	}
	if len(resp.Payload.OrderItems) != 1 {
		t.Fatalf("len(OrderItems) = %d, want 1", len(resp.Payload.OrderItems))
	}

	item := resp.Payload.OrderItems[0]
	if item.ASIN != "B08N5WRWNW" {
		t.Errorf("ASIN = %q, want %q", item.ASIN, "B08N5WRWNW")
	}
	if item.SellerSKU != "SKU-001" {
		t.Errorf("SellerSKU = %q, want %q", item.SellerSKU, "SKU-001")
	}
	if item.QuantityOrdered != 2 {
		t.Errorf("QuantityOrdered = %d, want 2", item.QuantityOrdered)
	}
	if item.ItemPrice.Amount != "49.99" {
		t.Errorf("ItemPrice.Amount = %q, want %q", item.ItemPrice.Amount, "49.99")
	}
}

func TestCatalogGetItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog/2022-04-01/items/B08N5WRWNW" {
			t.Errorf("path = %q, want /catalog/2022-04-01/items/B08N5WRWNW", r.URL.Path)
		}
		if r.URL.Query().Get("marketplaceIds") != "A1RKKUPIHCS9HS" {
			t.Errorf("marketplaceIds = %q, want A1RKKUPIHCS9HS", r.URL.Query().Get("marketplaceIds"))
		}
		if r.URL.Query().Get("includedData") != "summaries" {
			t.Errorf("includedData = %q, want summaries", r.URL.Query().Get("includedData"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CatalogItemResponse{
			ASIN: "B08N5WRWNW",
			Summaries: []ItemSummary{
				{
					MarketplaceID: "A1RKKUPIHCS9HS",
					ItemName:      "Widget Pro",
					BrandName:     "WidgetCo",
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	item, err := c.Catalog.GetItem(context.Background(), "B08N5WRWNW", "A1RKKUPIHCS9HS")
	if err != nil {
		t.Fatalf("Catalog.GetItem error: %v", err)
	}
	if item.ASIN != "B08N5WRWNW" {
		t.Errorf("ASIN = %q, want %q", item.ASIN, "B08N5WRWNW")
	}
	if len(item.Summaries) != 1 {
		t.Fatalf("len(Summaries) = %d, want 1", len(item.Summaries))
	}
	if item.Summaries[0].ItemName != "Widget Pro" {
		t.Errorf("ItemName = %q, want %q", item.Summaries[0].ItemName, "Widget Pro")
	}
	if item.Summaries[0].BrandName != "WidgetCo" {
		t.Errorf("BrandName = %q, want %q", item.Summaries[0].BrandName, "WidgetCo")
	}
}

func TestOrdersListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(APIError{
			Errors: []SPError{
				{Code: "Unauthorized", Message: "Access denied"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.List(context.Background(), "", nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func TestOrdersGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Errors: []SPError{
				{Code: "InvalidInput", Message: "Order not found"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.Get(context.Background(), "invalid-order-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"code":"InternalFailure","message":"Internal error"}]}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Catalog.GetItem(context.Background(), "B123", "A1RKKUPIHCS9HS")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if len(apiErr.Errors) == 0 {
		t.Fatal("expected Errors to be non-empty")
	}
	if apiErr.Errors[0].Code != "InternalFailure" {
		t.Errorf("Errors[0].Code = %q, want %q", apiErr.Errors[0].Code, "InternalFailure")
	}
}

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with errors",
			err: APIError{
				StatusCode: 400,
				Errors:     []SPError{{Code: "InvalidInput", Message: "Bad request"}},
			},
			want: "amazon: api error InvalidInput: Bad request",
		},
		{
			name: "no errors",
			err:  APIError{StatusCode: 500},
			want: "amazon: api error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		amazon string
		oms    string
		wantOK bool
	}{
		{"Pending", "pending", true},
		{"Unshipped", "confirmed", true},
		{"PartiallyShipped", "confirmed", true},
		{"Shipped", "shipped", true},
		{"Canceled", "cancelled", true},
		{"Unfulfillable", "cancelled", true},
		{"InvoiceUnconfirmed", "pending", true},
		{"PendingAvailability", "pending", true},
		{"UnknownStatus", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.amazon)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.amazon, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.amazon, oms, tc.oms)
		}
	}
}

func TestEnsureTokenSkipsRefreshWhenValid(t *testing.T) {
	// Client with a valid (non-expired) token should NOT trigger refresh.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If token refresh is triggered, it would hit amazon.com, not our server.
		// Our server should only see the actual API call with the pre-set token.
		tok := r.Header.Get("x-amz-access-token")
		if tok != "valid-access-token" {
			t.Errorf("x-amz-access-token = %q, want valid-access-token", tok)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrdersResponse{
			Payload: OrdersPayload{Orders: []Order{}},
		})
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithTokens("valid-access-token", "rt", time.Now().Add(time.Hour)),
	)

	_, err := c.Orders.List(context.Background(), "", nil, "")
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
}
