package kaufland

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("api_key_123", "secret_key_456")

	if c.apiKey != "api_key_123" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "api_key_123")
	}
	if c.secretKey != "secret_key_456" {
		t.Errorf("secretKey = %q, want %q", c.secretKey, "secret_key_456")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.sandbox {
		t.Error("sandbox should be false by default")
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
}

func TestNewClientSandbox(t *testing.T) {
	c := NewClient("key", "secret", WithSandbox())

	if !c.sandbox {
		t.Error("expected sandbox = true")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("key", "secret", WithBaseURL("https://custom.api/v2/"))

	if c.baseURL != "https://custom.api/v2" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api/v2")
	}
}

func TestDoSetsSignatureHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Shop-Client-Key") != "api_key" {
			t.Errorf("Shop-Client-Key = %q, want %q", r.Header.Get("Shop-Client-Key"), "api_key")
		}
		if r.Header.Get("Shop-Timestamp") == "" {
			t.Error("Shop-Timestamp is empty")
		}
		if r.Header.Get("Shop-Signature") == "" {
			t.Error("Shop-Signature is empty")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("api_key", "secret_key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoSetsSandboxHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Shop-Storefront") != "sandbox" {
			t.Errorf("Shop-Storefront = %q, want sandbox", r.Header.Get("Shop-Storefront"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithSandbox(),
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
		w.Write([]byte(`{"message":"Order unit not found","code":404}`))
	}))
	defer srv.Close()

	c := NewClient("key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
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
	if apiErr.Message != "Order unit not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Order unit not found")
	}
}

func TestGetOrderUnits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order-units" {
			t.Errorf("path = %q, want /order-units", r.URL.Path)
		}
		if ca := r.URL.Query().Get("ts_created_from_iso"); ca != "2024-01-15T00:00:00Z" {
			t.Errorf("ts_created_from_iso = %q, want 2024-01-15T00:00:00Z", ca)
		}

		resp := OrderUnitListResponse{
			Data: []OrderUnit{
				{
					IDOrderUnit: 12345,
					IDOrder:     99001,
					Status:      "received",
					CreatedAt:   "2024-01-15T10:30:00Z",
					Item:        Item{IDItem: 555, Title: "Widget Pro", EAN: "5901234123457"},
					Buyer:       Buyer{Email: "jan@test.pl"},
					ShippingAddr: Address{
						FirstName: "Jan", LastName: "Kowalski",
						Street: "Marszalkowska", HouseNumber: "1",
						City: "Warszawa", PostalCode: "00-001", Country: "PL",
					},
					Price:    199.99,
					Quantity: 1,
					Currency: "PLN",
				},
			},
			Pagination: Pagination{Total: 1, Limit: 25, Offset: 0},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	result, err := c.Orders.GetOrderUnits(context.Background(), OrderUnitListParams{
		CreatedAfter: "2024-01-15T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("GetOrderUnits error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].IDOrderUnit != 12345 {
		t.Errorf("IDOrderUnit = %d, want 12345", result.Data[0].IDOrderUnit)
	}
	if result.Data[0].Item.Title != "Widget Pro" {
		t.Errorf("Item.Title = %q, want %q", result.Data[0].Item.Title, "Widget Pro")
	}
	if result.Data[0].Buyer.Email != "jan@test.pl" {
		t.Errorf("Buyer.Email = %q, want %q", result.Data[0].Buyer.Email, "jan@test.pl")
	}
}

func TestGetOrderUnit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order-units/12345" {
			t.Errorf("path = %q, want /order-units/12345", r.URL.Path)
		}
		resp := struct {
			Data OrderUnit `json:"data"`
		}{
			Data: OrderUnit{
				IDOrderUnit: 12345,
				IDOrder:     99001,
				Status:      "received",
				Item:        Item{IDItem: 555, Title: "Widget Pro"},
				Buyer:       Buyer{Email: "buyer@test.pl"},
				Price:       99.50,
				Currency:    "EUR",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	unit, err := c.Orders.GetOrderUnit(context.Background(), 12345)
	if err != nil {
		t.Fatalf("GetOrderUnit error: %v", err)
	}
	if unit.IDOrderUnit != 12345 {
		t.Errorf("IDOrderUnit = %d, want 12345", unit.IDOrderUnit)
	}
	if unit.Item.Title != "Widget Pro" {
		t.Errorf("Item.Title = %q, want %q", unit.Item.Title, "Widget Pro")
	}
}

func TestSignature(t *testing.T) {
	c := NewClient("key", "secret")
	sig1 := c.sign("GET", "https://example.com/v2/orders", nil, "2024-01-15T10:00:00Z")
	sig2 := c.sign("GET", "https://example.com/v2/orders", nil, "2024-01-15T10:00:00Z")

	if sig1 != sig2 {
		t.Errorf("same input should produce same signature, got %q and %q", sig1, sig2)
	}
	if len(sig1) != 64 { // HMAC-SHA256 hex is 64 chars
		t.Errorf("signature length = %d, want 64", len(sig1))
	}

	sig3 := c.sign("POST", "https://example.com/v2/orders", []byte(`{"foo":"bar"}`), "2024-01-15T10:00:00Z")
	if sig1 == sig3 {
		t.Error("different input should produce different signature")
	}
}

func TestAPIErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with message",
			err:  APIError{StatusCode: 404, Message: "Not found"},
			want: "kaufland: HTTP 404: Not found",
		},
		{
			name: "status only",
			err:  APIError{StatusCode: 500},
			want: "kaufland: HTTP 500",
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
