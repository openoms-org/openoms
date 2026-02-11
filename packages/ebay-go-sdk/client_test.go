package ebay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("app123", "cert123", "dev123", "refresh_tok")

	if c.appID != "app123" {
		t.Errorf("appID = %q, want %q", c.appID, "app123")
	}
	if c.certID != "cert123" {
		t.Errorf("certID = %q, want %q", c.certID, "cert123")
	}
	if c.devID != "dev123" {
		t.Errorf("devID = %q, want %q", c.devID, "dev123")
	}
	if c.refreshToken != "refresh_tok" {
		t.Errorf("refreshToken = %q, want %q", c.refreshToken, "refresh_tok")
	}
	if c.apiURL != productionAPIURL {
		t.Errorf("apiURL = %q, want %q", c.apiURL, productionAPIURL)
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
}

func TestNewClientSandbox(t *testing.T) {
	c := NewClient("app", "cert", "dev", "tok", WithSandbox())

	if c.apiURL != sandboxAPIURL {
		t.Errorf("apiURL = %q, want %q", c.apiURL, sandboxAPIURL)
	}
	if c.authURL != sandboxAuthURL {
		t.Errorf("authURL = %q, want %q", c.authURL, sandboxAuthURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("app", "cert", "dev", "tok", WithBaseURL("https://custom.api"))

	if c.apiURL != "https://custom.api" {
		t.Errorf("apiURL = %q, want %q", c.apiURL, "https://custom.api")
	}
}

func TestWithAccessToken(t *testing.T) {
	c := NewClient("app", "cert", "dev", "tok", WithAccessToken("test_token"))

	if c.accessToken != "test_token" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "test_token")
	}
}

func TestDoSetsAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test_token")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoHandlesErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"errorId":11001,"domain":"fulfillment","category":"REQUEST","message":"Order not found"}]}`))
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/missing", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if len(apiErr.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(apiErr.Errors))
	}
	if apiErr.Errors[0].ErrorID != 11001 {
		t.Errorf("ErrorID = %d, want 11001", apiErr.Errors[0].ErrorID)
	}
}

func TestGetOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sell/fulfillment/v1/order" {
			t.Errorf("path = %q, want /sell/fulfillment/v1/order", r.URL.Path)
		}
		if f := r.URL.Query().Get("filter"); f != "creationdate:[2024-01-01T00:00:00Z..]" {
			t.Errorf("filter = %q, want creationdate filter", f)
		}
		if l := r.URL.Query().Get("limit"); l != "50" {
			t.Errorf("limit = %q, want 50", l)
		}

		resp := OrderSearchResponse{
			Total: 1,
			Limit: 50,
			Orders: []Order{
				{
					OrderID:          "04-12345-67890",
					CreationDate:     "2024-01-15T10:30:00.000Z",
					LastModifiedDate: "2024-01-15T11:00:00.000Z",
					OrderFulfStatus:  "NOT_STARTED",
					OrderPaymentStat: "PAID",
					PricingSummary: PricingSummary{
						Total: Amount{Value: "199.99", Currency: "PLN"},
					},
					Buyer: BuyerInfo{Username: "jan_kowalski"},
					LineItems: []LineItem{
						{
							LineItemID: "1001",
							Title:      "Widget Pro",
							SKU:        "WP-001",
							Quantity:   2,
							Total:      Amount{Value: "199.99", Currency: "PLN"},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Orders.GetOrders(context.Background(), OrderSearchParams{
		Filter: "creationdate:[2024-01-01T00:00:00Z..]",
		Limit:  50,
	})
	if err != nil {
		t.Fatalf("GetOrders error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Orders) != 1 {
		t.Fatalf("len(Orders) = %d, want 1", len(result.Orders))
	}
	if result.Orders[0].OrderID != "04-12345-67890" {
		t.Errorf("OrderID = %q, want %q", result.Orders[0].OrderID, "04-12345-67890")
	}
	if result.Orders[0].LineItems[0].SKU != "WP-001" {
		t.Errorf("SKU = %q, want %q", result.Orders[0].LineItems[0].SKU, "WP-001")
	}
}

func TestGetOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sell/fulfillment/v1/order/04-12345-67890" {
			t.Errorf("path = %q, want /sell/fulfillment/v1/order/04-12345-67890", r.URL.Path)
		}
		resp := Order{
			OrderID:          "04-12345-67890",
			OrderFulfStatus:  "NOT_STARTED",
			OrderPaymentStat: "PAID",
			PricingSummary: PricingSummary{
				Total: Amount{Value: "99.00", Currency: "EUR"},
			},
			Buyer: BuyerInfo{Username: "buyer123"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	order, err := c.Orders.GetOrder(context.Background(), "04-12345-67890")
	if err != nil {
		t.Fatalf("GetOrder error: %v", err)
	}
	if order.OrderID != "04-12345-67890" {
		t.Errorf("OrderID = %q, want %q", order.OrderID, "04-12345-67890")
	}
	if order.Buyer.Username != "buyer123" {
		t.Errorf("Buyer.Username = %q, want %q", order.Buyer.Username, "buyer123")
	}
}

func TestAPIErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with errors array",
			err: APIError{
				StatusCode: 404,
				Errors:     []EbErr{{ErrorID: 11001, Message: "Order not found"}},
			},
			want: "ebay: HTTP 404: Order not found",
		},
		{
			name: "with message fallback",
			err:  APIError{StatusCode: 500, Message: "Internal Server Error"},
			want: "ebay: HTTP 500: Internal Server Error",
		},
		{
			name: "status only",
			err:  APIError{StatusCode: 400},
			want: "ebay: HTTP 400",
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

func TestAPIErrorUnwrap(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{401, ErrUnauthorized},
		{403, ErrForbidden},
		{404, ErrNotFound},
		{429, ErrRateLimited},
		{500, ErrServerError},
		{503, ErrServerError},
		{400, nil},
	}

	for _, tc := range tests {
		apiErr := &APIError{StatusCode: tc.status}
		got := apiErr.Unwrap()
		if got != tc.want {
			t.Errorf("Unwrap() for status %d = %v, want %v", tc.status, got, tc.want)
		}
	}
}
