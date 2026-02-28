package erli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("tok-123")

	if c.apiToken != "tok-123" {
		t.Errorf("apiToken = %q, want %q", c.apiToken, "tok-123")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Offers == nil {
		t.Error("Offers service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("tok", WithSandbox())

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("tok", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("tok", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set to custom client")
	}
}

func TestDoSetsBearerToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("test-token",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoSetsContentTypeForBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
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
		if r.URL.Query().Get("status") != "purchased" {
			t.Errorf("status = %q, want purchased", r.URL.Query().Get("status"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OrdersResponse{
			Data: []Order{
				{
					ID:          "ORD-001",
					Status:      "purchased",
					BuyerName:   "Jan Kowalski",
					BuyerEmail:  "jan@example.com",
					TotalAmount: 199.99,
					Currency:    "PLN",
					Items: []OrderItem{
						{ID: "ITEM-1", Name: "Widget", SKU: "WDG-001", Quantity: 2, Price: 99.99},
					},
				},
			},
			Meta: struct {
				NextCursor string `json:"next_cursor"`
			}{NextCursor: "cursor-abc"},
		})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Orders.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].ID != "ORD-001" {
		t.Errorf("Data[0].ID = %q, want %q", resp.Data[0].ID, "ORD-001")
	}
	if resp.Data[0].BuyerName != "Jan Kowalski" {
		t.Errorf("Data[0].BuyerName = %q, want %q", resp.Data[0].BuyerName, "Jan Kowalski")
	}
	if resp.Data[0].TotalAmount != 199.99 {
		t.Errorf("Data[0].TotalAmount = %f, want 199.99", resp.Data[0].TotalAmount)
	}
	if len(resp.Data[0].Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(resp.Data[0].Items))
	}
	if resp.Data[0].Items[0].SKU != "WDG-001" {
		t.Errorf("Items[0].SKU = %q, want %q", resp.Data[0].Items[0].SKU, "WDG-001")
	}
	if resp.Meta.NextCursor != "cursor-abc" {
		t.Errorf("Meta.NextCursor = %q, want %q", resp.Meta.NextCursor, "cursor-abc")
	}
}

func TestOrdersListWithCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") != "page2" {
			t.Errorf("after = %q, want page2", r.URL.Query().Get("after"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OrdersResponse{Data: []Order{}})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Orders.List(context.Background(), "page2")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(resp.Data))
	}
}

func TestOrdersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders/ORD-123" {
			t.Errorf("path = %q, want /orders/ORD-123", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Order{
			ID:            "ORD-123",
			Status:        "purchased",
			BuyerName:     "Anna Nowak",
			BuyerEmail:    "anna@example.com",
			TotalAmount:   49.99,
			Currency:      "PLN",
			PaymentStatus: "paid",
			Address: Address{
				Name:     "Anna Nowak",
				Street:   "Kwiatowa 5",
				City:     "Krakow",
				PostCode: "30-001",
				Country:  "PL",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	order, err := c.Orders.Get(context.Background(), "ORD-123")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if order.ID != "ORD-123" {
		t.Errorf("ID = %q, want %q", order.ID, "ORD-123")
	}
	if order.BuyerName != "Anna Nowak" {
		t.Errorf("BuyerName = %q, want %q", order.BuyerName, "Anna Nowak")
	}
	if order.Address.City != "Krakow" {
		t.Errorf("Address.City = %q, want %q", order.Address.City, "Krakow")
	}
	if order.PaymentStatus != "paid" {
		t.Errorf("PaymentStatus = %q, want %q", order.PaymentStatus, "paid")
	}
}

func TestOrdersGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Order not found",
			"code":    "NOT_FOUND",
		})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.Get(context.Background(), "INVALID")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestOffersUpdateStock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/products/OFFER-42" {
			t.Errorf("path = %q, want /products/OFFER-42", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if stock, ok := body["stock"].(float64); !ok || int(stock) != 15 {
			t.Errorf("stock = %v, want 15", body["stock"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Offers.UpdateStock(context.Background(), "OFFER-42", 15)
	if err != nil {
		t.Fatalf("UpdateStock() error: %v", err)
	}
}

func TestOffersUpdatePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/products/OFFER-42" {
			t.Errorf("path = %q, want /products/OFFER-42", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if price, ok := body["price"].(float64); !ok || price != 29.99 {
			t.Errorf("price = %v, want 29.99", body["price"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Offers.UpdatePrice(context.Background(), "OFFER-42", 29.99)
	if err != nil {
		t.Fatalf("UpdatePrice() error: %v", err)
	}
}

func TestOffersCreate(t *testing.T) {
	const testSKU = "SKU-001"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/products/"+testSKU {
			t.Errorf("path = %q, want /products/%s", r.URL.Path, testSKU)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateOfferResponse{ID: "ERLI-INTERNAL-42"})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	req := CreateOfferRequest{
		Title: "Test Product",
		Price: 19.99,
		Stock: 100,
	}
	offerID, err := c.Offers.Create(context.Background(), testSKU, req)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if offerID != "ERLI-INTERNAL-42" {
		t.Errorf("Create() offerID = %q, want %q", offerID, "ERLI-INTERNAL-42")
	}
}

func TestOffersCreate_202Accepted(t *testing.T) {
	const testSKU = "SKU-ASYNC-001"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/products/"+testSKU {
			t.Errorf("path = %q, want /products/%s", r.URL.Path, testSKU)
		}
		// Erli responds 202 when product is queued for async validation.
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	req := CreateOfferRequest{
		Title: "Async Product",
		Price: 9.99,
		Stock: 5,
	}
	offerID, err := c.Offers.Create(context.Background(), testSKU, req)
	if !errors.Is(err, ErrProductPendingValidation) {
		t.Errorf("Create() error = %v, want ErrProductPendingValidation", err)
	}
	if offerID != testSKU {
		t.Errorf("Create() offerID = %q, want %q (externalID returned on 202)", offerID, testSKU)
	}
}

func TestOffersUpdateStockError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid stock value",
			"code":    "VALIDATION_ERROR",
		})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Offers.UpdateStock(context.Background(), "OFFER-42", -1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
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

func TestInvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer srv.Close()

	c := NewClient("tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.Get(context.Background(), "ORD-123")
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
			want: "erli: api error 400: Bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "erli: api error 500",
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
		erli   string
		oms    string
		wantOK bool
	}{
		{"pending", "new", true},
		{"purchased", "confirmed", true},
		{"cancelled", "cancelled", true},
		{"NONEXISTENT", "", false},
		{"", "", false},
		// Old statuses must no longer be in the map.
		{"new", "", false},
		{"paid", "", false},
		{"shipped", "", false},
		{"delivered", "", false},
		{"returned", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.erli)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.erli, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.erli, oms, tc.oms)
		}
	}
}
