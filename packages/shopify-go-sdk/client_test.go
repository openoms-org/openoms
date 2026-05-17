package shopify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient("myshop", "shpat_test-token",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
}

func TestNewClientDomainNormalization(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantURL string
	}{
		{
			name:    "short domain gets myshopify.com appended",
			domain:  "myshop",
			wantURL: "https://myshop.myshopify.com/admin/api/" + defaultAPIVersion,
		},
		{
			name:    "full domain preserved",
			domain:  "myshop.myshopify.com",
			wantURL: "https://myshop.myshopify.com/admin/api/" + defaultAPIVersion,
		},
		{
			name:    "domain with https prefix",
			domain:  "https://myshop.myshopify.com",
			wantURL: "https://myshop.myshopify.com/admin/api/" + defaultAPIVersion,
		},
		{
			name:    "trailing slash trimmed",
			domain:  "myshop.myshopify.com/",
			wantURL: "https://myshop.myshopify.com/admin/api/" + defaultAPIVersion,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(tc.domain, "token")
			if c.baseURL != tc.wantURL {
				t.Errorf("baseURL = %q, want %q", c.baseURL, tc.wantURL)
			}
		})
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("myshop", "shpat_token")

	if c.accessToken != "shpat_token" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "shpat_token")
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Products == nil {
		t.Error("Products service is nil")
	}
	if c.Inventory == nil {
		t.Error("Inventory service is nil")
	}
}

func TestWithAPIVersion(t *testing.T) {
	c := NewClient("myshop", "token", WithAPIVersion("2024-10"))

	if !strings.Contains(c.baseURL, "/admin/api/2024-10") {
		t.Errorf("baseURL = %q, expected to contain /admin/api/2024-10", c.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("myshop", "token", WithBaseURL("https://custom.api/v1"))

	if c.baseURL != "https://custom.api/v1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api/v1")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("myshop", "token", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set by WithHTTPClient")
	}
}

func TestDoSetsAccessTokenHeader(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Shopify-Access-Token")
		if token != "shpat_test-token" {
			t.Errorf("X-Shopify-Access-Token = %q, want %q", token, "shpat_test-token")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test.json", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoWithRequestBody(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	body := map[string]string{"key": "value"}
	err := c.do(context.Background(), "POST", "/test.json", body, nil)
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

func TestProductServiceCreate(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/products.json" {
			t.Fatalf("request = %s %s, want POST /products.json", r.Method, r.URL.Path)
		}

		var payload struct {
			Product map[string]any `json:"product"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload.Product["title"] != "Created product" {
			t.Fatalf("title = %#v, want Created product", payload.Product["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"product": {
				"id": 123,
				"title": "Created product",
				"variants": [
					{"id": 456, "product_id": 123, "inventory_item_id": 789}
				]
			}
		}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	product, err := c.Products.Create(context.Background(), map[string]any{
		"title": "Created product",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if product.ID != 123 {
		t.Fatalf("product ID = %d, want 123", product.ID)
	}
	if len(product.Variants) != 1 {
		t.Fatalf("len(Variants) = %d, want 1", len(product.Variants))
	}
	if product.Variants[0].ID != 456 {
		t.Fatalf("variant ID = %d, want 456", product.Variants[0].ID)
	}
	if product.Variants[0].InventoryItemID != 789 {
		t.Fatalf("inventory item ID = %d, want 789", product.Variants[0].InventoryItemID)
	}
}

func TestProductServiceGetVariant(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/variants/456.json" {
			t.Fatalf("request = %s %s, want GET /variants/456.json", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"variant": {
				"id": 456,
				"product_id": 123,
				"inventory_item_id": 789,
				"sku": "SKU-123"
			}
		}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	variant, err := c.Products.GetVariant(context.Background(), 456)
	if err != nil {
		t.Fatalf("GetVariant returned error: %v", err)
	}
	if variant.ID != 456 {
		t.Fatalf("variant ID = %d, want 456", variant.ID)
	}
	if variant.InventoryItemID != 789 {
		t.Fatalf("inventory item ID = %d, want 789", variant.InventoryItemID)
	}
}

func TestDoRejectsOversizedSuccessResponse(t *testing.T) {
	largeJSON := `{"value":"` + strings.Repeat("x", 64) + `"}`
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(largeJSON))
	})
	defer srv.Close()

	c := NewClient("myshop", "shpat_test-token",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithMaxResponseBytes(16),
	)

	var result map[string]string
	err := c.do(context.Background(), "GET", "/large.json", nil, &result)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", err)
	}
}

func TestDoRejectsOversizedErrorResponse(t *testing.T) {
	largeJSON := `{"errors":"` + strings.Repeat("x", 64) + `"}`
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(largeJSON))
	})
	defer srv.Close()

	c := NewClient("myshop", "shpat_test-token",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithMaxResponseBytes(16),
	)

	err := c.do(context.Background(), "GET", "/large-error.json", nil, nil)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", err)
	}
}

func TestOrdersList(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders.json" {
			t.Errorf("path = %q, want /orders.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orders": []map[string]any{
				{"id": 1001, "name": "#1001", "total_price": "99.99", "currency": "PLN"},
				{"id": 1002, "name": "#1002", "total_price": "49.99", "currency": "PLN"},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	orders, err := c.Orders.List(context.Background(), OrderListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("len(orders) = %d, want 2", len(orders))
	}
	if orders[0].Name != "#1001" {
		t.Errorf("orders[0].Name = %q, want %q", orders[0].Name, "#1001")
	}
	if orders[1].TotalPrice != "49.99" {
		t.Errorf("orders[1].TotalPrice = %q, want %q", orders[1].TotalPrice, "49.99")
	}
}

func TestOrdersListWithParams(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "50" {
			t.Errorf("limit = %q, want 50", q.Get("limit"))
		}
		if q.Get("since_id") != "1000" {
			t.Errorf("since_id = %q, want 1000", q.Get("since_id"))
		}
		if q.Get("status") != "open" {
			t.Errorf("status = %q, want open", q.Get("status"))
		}
		if q.Get("financial_status") != "paid" {
			t.Errorf("financial_status = %q, want paid", q.Get("financial_status"))
		}
		if q.Get("fulfillment_status") != "unshipped" {
			t.Errorf("fulfillment_status = %q, want unshipped", q.Get("fulfillment_status"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"orders": []any{}})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.List(context.Background(), OrderListParams{
		Limit:             50,
		SinceID:           1000,
		Status:            "open",
		FinancialStatus:   "paid",
		FulfillmentStatus: "unshipped",
	})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
}

func TestOrdersGet(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders/1001.json" {
			t.Errorf("path = %q, want /orders/1001.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{
				"id":               1001,
				"name":             "#1001",
				"email":            "jan@example.com",
				"total_price":      "299.99",
				"financial_status": "paid",
				"currency":         "PLN",
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	order, err := c.Orders.Get(context.Background(), 1001)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if order.ID != 1001 {
		t.Errorf("ID = %d, want 1001", order.ID)
	}
	if order.Email != "jan@example.com" {
		t.Errorf("Email = %q, want %q", order.Email, "jan@example.com")
	}
	if order.FinancialStatus != "paid" {
		t.Errorf("FinancialStatus = %q, want %q", order.FinancialStatus, "paid")
	}
}

func TestOrdersUpdate(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/orders/1001.json" {
			t.Errorf("path = %q, want /orders/1001.json", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		order, ok := body["order"].(map[string]any)
		if !ok {
			t.Fatal("missing order key in body")
		}
		if order["note"] != "Updated note" {
			t.Errorf("note = %v, want Updated note", order["note"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Orders.Update(context.Background(), 1001, map[string]any{"note": "Updated note"})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestProductsList(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/products.json" {
			t.Errorf("path = %q, want /products.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{
				{"id": 1, "title": "Widget", "status": "active", "vendor": "OpenOMS"},
				{"id": 2, "title": "Gadget", "status": "draft", "vendor": "OpenOMS"},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	products, err := c.Products.List(context.Background(), ProductListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
	if products[0].Title != "Widget" {
		t.Errorf("products[0].Title = %q, want %q", products[0].Title, "Widget")
	}
	if products[1].Status != "draft" {
		t.Errorf("products[1].Status = %q, want %q", products[1].Status, "draft")
	}
}

func TestProductsListWithParams(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "25" {
			t.Errorf("limit = %q, want 25", q.Get("limit"))
		}
		if q.Get("since_id") != "500" {
			t.Errorf("since_id = %q, want 500", q.Get("since_id"))
		}
		if q.Get("status") != "active" {
			t.Errorf("status = %q, want active", q.Get("status"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Products.List(context.Background(), ProductListParams{
		Limit:   25,
		SinceID: 500,
		Status:  "active",
	})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
}

func TestProductsGet(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/42.json" {
			t.Errorf("path = %q, want /products/42.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{
				"id":           42,
				"title":        "Premium Widget",
				"product_type": "Electronics",
				"status":       "active",
				"vendor":       "WidgetCo",
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	product, err := c.Products.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if product.ID != 42 {
		t.Errorf("ID = %d, want 42", product.ID)
	}
	if product.Title != "Premium Widget" {
		t.Errorf("Title = %q, want %q", product.Title, "Premium Widget")
	}
	if product.ProductType != "Electronics" {
		t.Errorf("ProductType = %q, want %q", product.ProductType, "Electronics")
	}
}

func TestProductsUpdate(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/products/42.json" {
			t.Errorf("path = %q, want /products/42.json", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		prod, ok := body["product"].(map[string]any)
		if !ok {
			t.Fatal("missing product key in body")
		}
		if prod["title"] != "Updated Widget" {
			t.Errorf("title = %v, want Updated Widget", prod["title"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Products.Update(context.Background(), 42, map[string]any{"title": "Updated Widget"})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestProductsUpdateVariant(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/variants/100.json" {
			t.Errorf("path = %q, want /variants/100.json", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		variant, ok := body["variant"].(map[string]any)
		if !ok {
			t.Fatal("missing variant key in body")
		}
		if variant["price"] != "29.99" {
			t.Errorf("price = %v, want 29.99", variant["price"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Products.UpdateVariant(context.Background(), 100, map[string]any{"price": "29.99"})
	if err != nil {
		t.Fatalf("UpdateVariant() error: %v", err)
	}
}

func TestInventoryGetLevels(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/inventory_levels.json" {
			t.Errorf("path = %q, want /inventory_levels.json", r.URL.Path)
		}
		if r.URL.Query().Get("inventory_item_ids") != "12345" {
			t.Errorf("inventory_item_ids = %q, want 12345", r.URL.Query().Get("inventory_item_ids"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"inventory_levels": []map[string]any{
				{"inventory_item_id": 12345, "location_id": 1, "available": 50},
				{"inventory_item_id": 12345, "location_id": 2, "available": 10},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	levels, err := c.Inventory.GetLevels(context.Background(), 12345)
	if err != nil {
		t.Fatalf("GetLevels() error: %v", err)
	}
	if len(levels) != 2 {
		t.Fatalf("len(levels) = %d, want 2", len(levels))
	}
	if levels[0].Available != 50 {
		t.Errorf("levels[0].Available = %d, want 50", levels[0].Available)
	}
	if levels[1].LocationID != 2 {
		t.Errorf("levels[1].LocationID = %d, want 2", levels[1].LocationID)
	}
}

func TestInventorySetLevel(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/inventory_levels/set.json" {
			t.Errorf("path = %q, want /inventory_levels/set.json", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["inventory_item_id"] != float64(12345) {
			t.Errorf("inventory_item_id = %v, want 12345", body["inventory_item_id"])
		}
		if body["location_id"] != float64(1) {
			t.Errorf("location_id = %v, want 1", body["location_id"])
		}
		if body["available"] != float64(100) {
			t.Errorf("available = %v, want 100", body["available"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Inventory.SetLevel(context.Background(), 12345, 1, 100)
	if err != nil {
		t.Fatalf("SetLevel() error: %v", err)
	}
}

func TestInventoryAdjustLevel(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/inventory_levels/adjust.json" {
			t.Errorf("path = %q, want /inventory_levels/adjust.json", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["available_adjustment"] != float64(-5) {
			t.Errorf("available_adjustment = %v, want -5", body["available_adjustment"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Inventory.AdjustLevel(context.Background(), 12345, 1, -5)
	if err != nil {
		t.Fatalf("AdjustLevel() error: %v", err)
	}
}

func TestInventoryListLocations(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/locations.json" {
			t.Errorf("path = %q, want /locations.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"locations": []map[string]any{
				{"id": 1, "name": "Main Warehouse", "active": true, "city": "Warszawa", "country": "PL"},
				{"id": 2, "name": "Backup", "active": false, "city": "Krakow", "country": "PL"},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	locations, err := c.Inventory.ListLocations(context.Background())
	if err != nil {
		t.Fatalf("ListLocations() error: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("len(locations) = %d, want 2", len(locations))
	}
	if locations[0].Name != "Main Warehouse" {
		t.Errorf("locations[0].Name = %q, want %q", locations[0].Name, "Main Warehouse")
	}
	if locations[0].Active != true {
		t.Error("locations[0].Active = false, want true")
	}
	if locations[1].Active != false {
		t.Error("locations[1].Active = true, want false")
	}
}

func TestInventoryGetInventoryItem(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/inventory_items/99.json" {
			t.Errorf("path = %q, want /inventory_items/99.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"inventory_item": map[string]any{
				"id":      99,
				"sku":     "SKU-099",
				"tracked": true,
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	item, err := c.Inventory.GetInventoryItem(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetInventoryItem() error: %v", err)
	}
	if item.ID != 99 {
		t.Errorf("ID = %d, want 99", item.ID)
	}
	if item.SKU != "SKU-099" {
		t.Errorf("SKU = %q, want %q", item.SKU, "SKU-099")
	}
	if !item.Tracked {
		t.Error("Tracked = false, want true")
	}
}

func TestServerError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors": "Internal server error"}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

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
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Not Found"}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

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
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors": "[API] Invalid API key or access token (unrecognized login or wrong password)"}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.List(context.Background(), OrderListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Error("expected error to wrap ErrUnauthorized")
	}
}

func TestRateLimitedError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors": "Exceeded 2 calls per second for api client."}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.List(context.Background(), OrderListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrRateLimited) {
		t.Error("expected error to wrap ErrRateLimited")
	}
}

func TestShopifyErrorsStringFormat(t *testing.T) {
	// Shopify sometimes returns {"errors": "string"} instead of map
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors": "Required parameter missing or invalid"}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Message != "Required parameter missing or invalid" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Required parameter missing or invalid")
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
			want: "shopify: HTTP 400: Bad request",
		},
		{
			name: "status code only",
			err:  APIError{StatusCode: 503},
			want: "shopify: HTTP 503",
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

func TestInvalidJSONResponse(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestEmptyErrorBody(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
	// When body is empty, message should fall back to http.StatusText
	if apiErr.Message != "Bad Gateway" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Bad Gateway")
	}
}
