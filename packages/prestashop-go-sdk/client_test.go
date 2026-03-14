package prestashop

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("https://myshop.example.com", "APIKEY123")

	if c.apiKey != "APIKEY123" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "APIKEY123")
	}
	if c.baseURL != "https://myshop.example.com/api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://myshop.example.com/api")
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Products == nil {
		t.Error("Products service is nil")
	}
	if c.Stock == nil {
		t.Error("Stock service is nil")
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://myshop.example.com/", "key")

	if c.baseURL != "https://myshop.example.com/api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://myshop.example.com/api")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("https://shop.com", "key", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set by WithHTTPClient")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("https://shop.com", "key", WithBaseURL("https://custom.api/v1"))

	if c.baseURL != "https://custom.api/v1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api/v1")
	}
}

func TestDoSetsBasicAuthWithAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth header")
		}
		if user != "test-api-key" {
			t.Errorf("username = %q, want %q", user, "test-api-key")
		}
		if pass != "" {
			t.Errorf("password = %q, want empty string", pass)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "test-api-key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoAppendsOutputFormatJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("output_format") != "JSON" {
			t.Errorf("output_format = %q, want JSON", r.URL.Query().Get("output_format"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoWithRequestBody(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	body := map[string]string{"key": "value"}
	err := c.do(context.Background(), "POST", "/test", body, nil)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if len(gotBody) == 0 {
		t.Error("expected request body, got empty")
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
		if r.URL.Query().Get("display") != "full" {
			t.Error("expected display=full query param")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orders": []map[string]any{
				{"id": 1, "reference": "ORD-001", "total_paid": "99.99"},
				{"id": 2, "reference": "ORD-002", "total_paid": "49.99"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	orders, err := c.Orders.List(context.Background(), OrderListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("len(orders) = %d, want 2", len(orders))
	}
	if orders[0].Reference != "ORD-001" {
		t.Errorf("orders[0].Reference = %q, want %q", orders[0].Reference, "ORD-001")
	}
	if orders[1].Reference != "ORD-002" {
		t.Errorf("orders[1].Reference = %q, want %q", orders[1].Reference, "ORD-002")
	}
}

func TestOrdersListWithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "0,10" {
			t.Errorf("limit = %q, want 0,10", q.Get("limit"))
		}
		if q.Get("sort") != "[date_upd_DESC]" {
			t.Errorf("sort = %q, want [date_upd_DESC]", q.Get("sort"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"orders": []any{}})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), OrderListParams{
		Limit:  10,
		Offset: 0,
		SortBy: "date_upd_DESC",
	})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
}

func TestOrdersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders/42" {
			t.Errorf("path = %q, want /orders/42", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{
				"id":          42,
				"reference":   "ORD-042",
				"total_paid":  "199.99",
				"id_customer": 7,
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	order, err := c.Orders.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if order.ID != 42 {
		t.Errorf("ID = %d, want 42", order.ID)
	}
	if order.Reference != "ORD-042" {
		t.Errorf("Reference = %q, want %q", order.Reference, "ORD-042")
	}
	if order.IDCustomer != 7 {
		t.Errorf("IDCustomer = %d, want 7", order.IDCustomer)
	}
}

func TestOrdersUpdateState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/order_histories" {
			t.Errorf("path = %q, want /order_histories", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		oh, ok := body["order_history"].(map[string]any)
		if !ok {
			t.Fatal("missing order_history key in body")
		}
		if oh["id_order"] != float64(42) {
			t.Errorf("id_order = %v, want 42", oh["id_order"])
		}
		if oh["id_order_state"] != float64(3) {
			t.Errorf("id_order_state = %v, want 3", oh["id_order_state"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Orders.UpdateState(context.Background(), 42, 3)
	if err != nil {
		t.Fatalf("UpdateState() error: %v", err)
	}
}

func TestOrdersGetAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/addresses/10" {
			t.Errorf("path = %q, want /addresses/10", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"address": map[string]any{
				"id":        10,
				"firstname": "Jan",
				"lastname":  "Kowalski",
				"city":      "Warszawa",
				"postcode":  "00-001",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	addr, err := c.Orders.GetAddress(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetAddress() error: %v", err)
	}
	if addr.FirstName != "Jan" {
		t.Errorf("FirstName = %q, want %q", addr.FirstName, "Jan")
	}
	if addr.City != "Warszawa" {
		t.Errorf("City = %q, want %q", addr.City, "Warszawa")
	}
}

func TestOrdersGetCustomer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customers/5" {
			t.Errorf("path = %q, want /customers/5", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"customer": map[string]any{
				"id":        5,
				"firstname": "Anna",
				"lastname":  "Nowak",
				"email":     "anna@example.com",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	cust, err := c.Orders.GetCustomer(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetCustomer() error: %v", err)
	}
	if cust.Email != "anna@example.com" {
		t.Errorf("Email = %q, want %q", cust.Email, "anna@example.com")
	}
	if cust.FirstName != "Anna" {
		t.Errorf("FirstName = %q, want %q", cust.FirstName, "Anna")
	}
}

func TestProductsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/products" {
			t.Errorf("path = %q, want /products", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{
				{"id": 1, "reference": "SKU-001", "price": "29.99"},
				{"id": 2, "reference": "SKU-002", "price": "59.99"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	products, err := c.Products.List(context.Background(), ProductListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
	if products[0].Reference != "SKU-001" {
		t.Errorf("products[0].Reference = %q, want %q", products[0].Reference, "SKU-001")
	}
}

func TestProductsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/7" {
			t.Errorf("path = %q, want /products/7", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{
				"id":        7,
				"reference": "SKU-007",
				"ean13":     "5901234123457",
				"price":     "149.99",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	product, err := c.Products.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if product.ID != 7 {
		t.Errorf("ID = %d, want 7", product.ID)
	}
	if product.EAN13 != "5901234123457" {
		t.Errorf("EAN13 = %q, want %q", product.EAN13, "5901234123457")
	}
}

func TestProductsUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/products/7" {
			t.Errorf("path = %q, want /products/7", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		prod, ok := body["product"].(map[string]any)
		if !ok {
			t.Fatal("missing product key in body")
		}
		if prod["active"] != "1" {
			t.Errorf("active = %v, want 1", prod["active"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Products.Update(context.Background(), 7, map[string]any{"active": "1"})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestProductsUpdatePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		prod, ok := body["product"].(map[string]any)
		if !ok {
			t.Fatal("missing product key in body")
		}
		if prod["price"] != "199.99" {
			t.Errorf("price = %v, want 199.99", prod["price"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Products.UpdatePrice(context.Background(), 7, "199.99")
	if err != nil {
		t.Fatalf("UpdatePrice() error: %v", err)
	}
}

func TestProductsListOrderDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/order_details" {
			t.Errorf("path = %q, want /order_details", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order_details": []map[string]any{
				{"id": 1, "id_order": 42, "product_name": "Widget", "product_quantity": 3},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	details, err := c.Products.ListOrderDetails(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListOrderDetails() error: %v", err)
	}
	if len(details) != 1 {
		t.Fatalf("len(details) = %d, want 1", len(details))
	}
	if details[0].ProductName != "Widget" {
		t.Errorf("ProductName = %q, want %q", details[0].ProductName, "Widget")
	}
	if details[0].Quantity != 3 {
		t.Errorf("Quantity = %d, want 3", details[0].Quantity)
	}
}

func TestStockGetByProduct(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/stock_availables" {
			t.Errorf("path = %q, want /stock_availables", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stock_availables": []map[string]any{
				{"id": 100, "id_product": 7, "quantity": 25},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	stock, err := c.Stock.GetByProduct(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetByProduct() error: %v", err)
	}
	if stock.ID != 100 {
		t.Errorf("ID = %d, want 100", stock.ID)
	}
	if stock.IDProduct != 7 {
		t.Errorf("IDProduct = %d, want 7", stock.IDProduct)
	}
	if stock.Quantity != 25 {
		t.Errorf("Quantity = %d, want 25", stock.Quantity)
	}
}

func TestStockGetByProductNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stock_availables": []any{},
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Stock.GetByProduct(context.Background(), 999)
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

func TestStockUpdateQuantity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/stock_availables/100" {
			t.Errorf("path = %q, want /stock_availables/100", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		sa, ok := body["stock_available"].(map[string]any)
		if !ok {
			t.Fatal("missing stock_available key in body")
		}
		if sa["quantity"] != float64(50) {
			t.Errorf("quantity = %v, want 50", sa["quantity"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Stock.UpdateQuantity(context.Background(), 100, 50)
	if err != nil {
		t.Fatalf("UpdateQuantity() error: %v", err)
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

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.Get(context.Background(), 1)
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
	if !errors.Is(err, ErrServerError) {
		t.Error("expected error to wrap ErrServerError")
	}
}

func TestNotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Resource not found",
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Products.Get(context.Background(), 999)
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
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to wrap ErrNotFound")
	}
}

func TestUnauthorizedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid API key",
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "bad-key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), OrderListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Error("expected error to wrap ErrUnauthorized")
	}
}

func TestRateLimitedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Too many requests",
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), OrderListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrRateLimited) {
		t.Error("expected error to wrap ErrRateLimited")
	}
}

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with message and code",
			err:  APIError{StatusCode: 400, Code: 47, Message: "Bad request"},
			want: "prestashop: HTTP 400 [47]: Bad request",
		},
		{
			name: "with message only",
			err:  APIError{StatusCode: 500, Message: "Server error"},
			want: "prestashop: HTTP 500: Server error",
		},
		{
			name: "status code only",
			err:  APIError{StatusCode: 503},
			want: "prestashop: HTTP 503",
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
		name       string
		statusCode int
		wantErr    error
	}{
		{"unauthorized", 401, ErrUnauthorized},
		{"forbidden", 403, ErrForbidden},
		{"not found", 404, ErrNotFound},
		{"rate limited", 429, ErrRateLimited},
		{"server error", 500, ErrServerError},
		{"bad gateway", 502, ErrServerError},
		{"bad request", 400, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := &APIError{StatusCode: tc.statusCode}
			if tc.wantErr == nil {
				if apiErr.Unwrap() != nil {
					t.Errorf("Unwrap() = %v, want nil", apiErr.Unwrap())
				}
			} else {
				if !errors.Is(apiErr, tc.wantErr) {
					t.Errorf("expected error to wrap %v", tc.wantErr)
				}
			}
		})
	}
}

func TestBadRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    47,
			"message": "Invalid product data",
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Products.Update(context.Background(), 1, map[string]any{"price": "invalid"})
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
	if apiErr.Code != 47 {
		t.Errorf("Code = %d, want 47", apiErr.Code)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.com", "key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
