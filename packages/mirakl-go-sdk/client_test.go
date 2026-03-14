package mirakl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("https://marketplace.mirakl.net/api", "api-key-123")

	if c.baseURL != "https://marketplace.mirakl.net/api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://marketplace.mirakl.net/api")
	}
	if c.apiKey != "api-key-123" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "api-key-123")
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Offers == nil {
		t.Error("Offers service is nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("https://original.url", "key", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("https://url", "key", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set to custom client")
	}
}

func TestDoSetsAuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "test-api-key" {
			t.Errorf("Authorization = %q, want %q", auth, "test-api-key")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-api-key",
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
}

func TestDoSetsContentTypeForBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	body := map[string]string{"key": "value"}
	var result map[string]any
	err := c.do(context.Background(), "POST", "/test", body, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
}

func TestOrdersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders" {
			t.Errorf("path = %q, want /orders", r.URL.Path)
		}
		if r.URL.Query().Get("order_state_codes") != "SHIPPING" {
			t.Errorf("order_state_codes = %q, want SHIPPING", r.URL.Query().Get("order_state_codes"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OrdersResponse{
			Orders: []Order{
				{
					ID:           "MKL-001",
					Status:       "SHIPPING",
					TotalPrice:   299.99,
					CurrencyCode: "PLN",
					Customer: Customer{
						FirstName: "Jan",
						LastName:  "Kowalski",
						Email:     "jan@example.com",
					},
					OrderLines: []OrderLine{
						{ID: "LINE-1", OfferSKU: "SKU-001", ProductTitle: "Widget", Quantity: 2, Price: 149.99},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Orders.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(resp.Orders) != 1 {
		t.Fatalf("len(Orders) = %d, want 1", len(resp.Orders))
	}
	if resp.Orders[0].ID != "MKL-001" {
		t.Errorf("Orders[0].ID = %q, want %q", resp.Orders[0].ID, "MKL-001")
	}
	if resp.Orders[0].Customer.FirstName != "Jan" {
		t.Errorf("Customer.FirstName = %q, want %q", resp.Orders[0].Customer.FirstName, "Jan")
	}
	if resp.Orders[0].TotalPrice != 299.99 {
		t.Errorf("TotalPrice = %f, want 299.99", resp.Orders[0].TotalPrice)
	}
	if len(resp.Orders[0].OrderLines) != 1 {
		t.Fatalf("len(OrderLines) = %d, want 1", len(resp.Orders[0].OrderLines))
	}
	if resp.Orders[0].OrderLines[0].OfferSKU != "SKU-001" {
		t.Errorf("OrderLines[0].OfferSKU = %q, want %q", resp.Orders[0].OrderLines[0].OfferSKU, "SKU-001")
	}
}

func TestOrdersListWithLastUpdated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("start_update_date") != "2024-01-15T00:00:00Z" {
			t.Errorf("start_update_date = %q, want 2024-01-15T00:00:00Z", r.URL.Query().Get("start_update_date"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OrdersResponse{Orders: []Order{}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Orders.List(context.Background(), "2024-01-15T00:00:00Z")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(resp.Orders) != 0 {
		t.Errorf("len(Orders) = %d, want 0", len(resp.Orders))
	}
}

func TestOrdersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders/MKL-123" {
			t.Errorf("path = %q, want /orders/MKL-123", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OrdersResponse{
			Orders: []Order{
				{
					ID:     "MKL-123",
					Status: "SHIPPED",
					ShippingAddress: Address{
						FirstName: "Anna",
						LastName:  "Nowak",
						Street1:   "Kwiatowa 5",
						City:      "Krakow",
						ZipCode:   "30-001",
						Country:   "PL",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	order, err := c.Orders.Get(context.Background(), "MKL-123")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if order.ID != "MKL-123" {
		t.Errorf("ID = %q, want %q", order.ID, "MKL-123")
	}
	if order.Status != "SHIPPED" {
		t.Errorf("Status = %q, want %q", order.Status, "SHIPPED")
	}
	if order.ShippingAddress.City != "Krakow" {
		t.Errorf("ShippingAddress.City = %q, want %q", order.ShippingAddress.City, "Krakow")
	}
}

func TestOrdersGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OrdersResponse{Orders: []Order{}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.Get(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "mirakl: order NONEXISTENT not found"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestOrdersAcceptOrderLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/orders/LINE-001/accept" {
			t.Errorf("path = %q, want /orders/LINE-001/accept", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	err := c.Orders.AcceptOrderLine(context.Background(), "LINE-001")
	if err != nil {
		t.Fatalf("AcceptOrderLine() error: %v", err)
	}
}

func TestOffersUpdateOffers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/offers" {
			t.Errorf("path = %q, want /offers", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var body OfferUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if len(body.Offers) != 2 {
			t.Fatalf("len(Offers) = %d, want 2", len(body.Offers))
		}
		if body.Offers[0].SKU != "SKU-001" {
			t.Errorf("Offers[0].SKU = %q, want %q", body.Offers[0].SKU, "SKU-001")
		}
		if body.Offers[0].Quantity != 10 {
			t.Errorf("Offers[0].Quantity = %d, want 10", body.Offers[0].Quantity)
		}
		if body.Offers[1].SKU != "SKU-002" {
			t.Errorf("Offers[1].SKU = %q, want %q", body.Offers[1].SKU, "SKU-002")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	err := c.Offers.UpdateOffers(context.Background(), []OfferUpdate{
		{SKU: "SKU-001", Quantity: 10, Price: 29.99},
		{SKU: "SKU-002", Quantity: 5, Price: 49.99},
	})
	if err != nil {
		t.Fatalf("UpdateOffers() error: %v", err)
	}
}

func TestOffersGetOffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/offers" {
			t.Errorf("path = %q, want /offers", r.URL.Path)
		}
		if r.URL.Query().Get("shop_skus") != "SKU-001" {
			t.Errorf("shop_skus = %q, want SKU-001", r.URL.Query().Get("shop_skus"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OfferResponse{
			Offers: []OfferDetail{
				{SKU: "SKU-001", Price: 29.99, Quantity: 15, Active: true},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	offer, err := c.Offers.GetOffer(context.Background(), "SKU-001")
	if err != nil {
		t.Fatalf("GetOffer() error: %v", err)
	}
	if offer.SKU != "SKU-001" {
		t.Errorf("SKU = %q, want %q", offer.SKU, "SKU-001")
	}
	if offer.Price != 29.99 {
		t.Errorf("Price = %f, want 29.99", offer.Price)
	}
	if offer.Quantity != 15 {
		t.Errorf("Quantity = %d, want 15", offer.Quantity)
	}
	if !offer.Active {
		t.Error("Active = false, want true")
	}
}

func TestOffersGetOfferNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OfferResponse{Offers: []OfferDetail{}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Offers.GetOffer(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "mirakl: offer NONEXISTENT not found"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), "")
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
	if apiErr.Message != "Internal server error" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Internal server error")
	}
}

func TestBadRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid request",
			"code":    "VALIDATION_ERROR",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	err := c.Offers.UpdateOffers(context.Background(), []OfferUpdate{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key",
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with message",
			err:  APIError{StatusCode: 400, Message: "Bad request"},
			want: "mirakl: api error 400: Bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 503},
			want: "mirakl: api error 503",
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
		mirakl string
		oms    string
		wantOK bool
	}{
		{"STAGING", "pending", true},
		{"WAITING_ACCEPTANCE", "confirmed", true},
		{"SHIPPING", "confirmed", true},
		{"SHIPPED", "shipped", true},
		{"RECEIVED", "delivered", true},
		{"CLOSED", "delivered", true},
		{"REFUSED", "cancelled", true},
		{"CANCELED", "cancelled", true},
		{"NONEXISTENT", "", false},
		{"", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.mirakl)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.mirakl, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.mirakl, oms, tc.oms)
		}
	}
}
