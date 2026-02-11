package woocommerce

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("https://shop.example.com", "ck_test", "cs_test")

	if c.consumerKey != "ck_test" {
		t.Errorf("consumerKey = %q, want %q", c.consumerKey, "ck_test")
	}
	if c.consumerSecret != "cs_test" {
		t.Errorf("consumerSecret = %q, want %q", c.consumerSecret, "cs_test")
	}
	if c.baseURL != "https://shop.example.com/wp-json/wc/v3" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://shop.example.com/wp-json/wc/v3")
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Products == nil {
		t.Error("Products service is nil")
	}
	if c.Webhooks == nil {
		t.Error("Webhooks service is nil")
	}
}

func TestNewClientTrailingSlash(t *testing.T) {
	c := NewClient("https://shop.example.com/", "ck_test", "cs_test")

	if c.baseURL != "https://shop.example.com/wp-json/wc/v3" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://shop.example.com/wp-json/wc/v3")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("https://shop.example.com", "ck", "cs", WithBaseURL("https://custom.api/v3"))

	if c.baseURL != "https://custom.api/v3" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api/v3")
	}
}

func TestDoSetsBasicAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("ck_abc:cs_xyz"))
		if auth != expected {
			t.Errorf("Authorization = %q, want %q", auth, expected)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck_abc", "cs_xyz",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
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
		w.Write([]byte(`{"code":"woocommerce_rest_shop_order_invalid_id","message":"Invalid ID."}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
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
	if apiErr.Code != "woocommerce_rest_shop_order_invalid_id" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "woocommerce_rest_shop_order_invalid_id")
	}
}

func TestDoNonDecodableErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html>Bad Gateway</html>`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.do(context.Background(), "GET", "/bad", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
	if apiErr.Message != "Bad Gateway" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Bad Gateway")
	}
}

func TestDoWithRequestBody(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":123}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	body := map[string]string{"name": "test"}
	var result map[string]any
	err := c.do(context.Background(), "POST", "/items", body, &result)
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

func TestDoDecodeResponseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json at all`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]string
	err := c.do(context.Background(), "GET", "/bad-json", nil, &result)
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with code and message",
			err:  APIError{StatusCode: 404, Code: "NOT_FOUND", Message: "Resource not found"},
			want: "woocommerce: HTTP 404 [NOT_FOUND]: Resource not found",
		},
		{
			name: "message only",
			err:  APIError{StatusCode: 500, Message: "Internal error"},
			want: "woocommerce: HTTP 500: Internal error",
		},
		{
			name: "status only",
			err:  APIError{StatusCode: 400},
			want: "woocommerce: HTTP 400",
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

// --- Order tests ---

func TestOrdersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders" {
			t.Errorf("path = %q, want /orders", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if pp := r.URL.Query().Get("per_page"); pp != "10" {
			t.Errorf("per_page = %q, want 10", pp)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": 42,
				"status": "processing",
				"currency": "PLN",
				"total": "199.99",
				"total_tax": "37.39",
				"billing": {
					"first_name": "Jan",
					"last_name": "Kowalski",
					"email": "jan@test.pl",
					"phone": "500100200",
					"address_1": "Marszalkowska 1",
					"address_2": "",
					"city": "Warszawa",
					"postcode": "00-001",
					"country": "PL"
				},
				"shipping": {
					"first_name": "Jan",
					"last_name": "Kowalski",
					"address_1": "Marszalkowska 1",
					"city": "Warszawa",
					"postcode": "00-001",
					"country": "PL"
				},
				"payment_method": "bacs",
				"payment_method_title": "Przelew bankowy",
				"line_items": [
					{
						"id": 1,
						"name": "Widget Pro",
						"sku": "WP-001",
						"quantity": 2,
						"total": "199.99",
						"total_tax": "37.39",
						"price": 99.995,
						"product_id": 15
					}
				],
				"date_created": "2024-01-15T10:30:00",
				"date_modified": "2024-01-15T11:00:00",
				"customer_note": "Please deliver ASAP"
			}
		]`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	orders, err := c.Orders.List(context.Background(), OrderListParams{PerPage: 10})
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}
	order := orders[0]
	if order.ID != 42 {
		t.Errorf("ID = %d, want 42", order.ID)
	}
	if order.Status != "processing" {
		t.Errorf("Status = %q, want %q", order.Status, "processing")
	}
	if order.Total != "199.99" {
		t.Errorf("Total = %q, want %q", order.Total, "199.99")
	}
	if order.Billing.Email != "jan@test.pl" {
		t.Errorf("Billing.Email = %q, want %q", order.Billing.Email, "jan@test.pl")
	}
	if order.Billing.Phone != "500100200" {
		t.Errorf("Billing.Phone = %q, want %q", order.Billing.Phone, "500100200")
	}
	if len(order.LineItems) != 1 {
		t.Fatalf("len(LineItems) = %d, want 1", len(order.LineItems))
	}
	if order.LineItems[0].Quantity != 2 {
		t.Errorf("LineItems[0].Quantity = %d, want 2", order.LineItems[0].Quantity)
	}
	if order.LineItems[0].SKU != "WP-001" {
		t.Errorf("LineItems[0].SKU = %q, want %q", order.LineItems[0].SKU, "WP-001")
	}
	if order.CustomerNote != "Please deliver ASAP" {
		t.Errorf("CustomerNote = %q, want %q", order.CustomerNote, "Please deliver ASAP")
	}
}

func TestOrdersListWithModifiedAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ma := r.URL.Query().Get("modified_after"); ma != "2024-01-15T00:00:00" {
			t.Errorf("modified_after = %q, want 2024-01-15T00:00:00", ma)
		}
		if s := r.URL.Query().Get("status"); s != "processing" {
			t.Errorf("status = %q, want processing", s)
		}
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), OrderListParams{
		ModifiedAfter: "2024-01-15T00:00:00",
		Status:        "processing",
	})
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
}

func TestOrdersListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	orders, err := c.Orders.List(context.Background(), OrderListParams{})
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("len(orders) = %d, want 0", len(orders))
	}
}

func TestOrdersListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"internal_error","message":"server error"}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.List(context.Background(), OrderListParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOrdersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/42" {
			t.Errorf("path = %q, want /orders/42", r.URL.Path)
		}
		w.Write([]byte(`{"id":42,"status":"completed","currency":"PLN","total":"99.00","billing":{"first_name":"Anna","last_name":"Nowak"},"shipping":{},"line_items":[],"date_created":"2024-02-01T12:00:00","date_modified":"2024-02-01T13:00:00"}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	order, err := c.Orders.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Orders.Get error: %v", err)
	}
	if order.ID != 42 {
		t.Errorf("ID = %d, want 42", order.ID)
	}
	if order.Status != "completed" {
		t.Errorf("Status = %q, want %q", order.Status, "completed")
	}
	if order.Billing.FirstName != "Anna" {
		t.Errorf("Billing.FirstName = %q, want %q", order.Billing.FirstName, "Anna")
	}
}

func TestOrdersGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"woocommerce_rest_shop_order_invalid_id","message":"Invalid ID."}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Orders.Get(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOrdersUpdateStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/42" {
			t.Errorf("path = %q, want /orders/42", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected request body")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Orders.UpdateStatus(context.Background(), 42, "completed")
	if err != nil {
		t.Fatalf("Orders.UpdateStatus error: %v", err)
	}
}

// --- Product tests ---

func TestProductsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			t.Errorf("path = %q, want /products", r.URL.Path)
		}
		if pp := r.URL.Query().Get("per_page"); pp != "20" {
			t.Errorf("per_page = %q, want 20", pp)
		}
		if search := r.URL.Query().Get("search"); search != "Widget" {
			t.Errorf("search = %q, want Widget", search)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": 15,
				"name": "Widget Pro",
				"sku": "WP-001",
				"price": "99.99",
				"regular_price": "129.99",
				"stock_quantity": 50,
				"stock_status": "instock",
				"status": "publish",
				"description": "A great widget",
				"short_description": "Great widget",
				"categories": [{"id": 1, "name": "Widgets"}]
			},
			{
				"id": 16,
				"name": "Widget Lite",
				"sku": "WL-001",
				"price": "49.99",
				"regular_price": "49.99",
				"stock_quantity": 100,
				"stock_status": "instock",
				"status": "publish"
			}
		]`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	products, err := c.Products.List(context.Background(), ProductListParams{
		PerPage: 20,
		Search:  "Widget",
	})
	if err != nil {
		t.Fatalf("Products.List error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
	if products[0].Name != "Widget Pro" {
		t.Errorf("products[0].Name = %q, want %q", products[0].Name, "Widget Pro")
	}
	if products[0].SKU != "WP-001" {
		t.Errorf("products[0].SKU = %q, want %q", products[0].SKU, "WP-001")
	}
	if products[0].Price != "99.99" {
		t.Errorf("products[0].Price = %q, want %q", products[0].Price, "99.99")
	}
	if products[0].StockQuantity == nil || *products[0].StockQuantity != 50 {
		t.Errorf("products[0].StockQuantity = %v, want 50", products[0].StockQuantity)
	}
	if len(products[0].Categories) != 1 {
		t.Fatalf("len(categories) = %d, want 1", len(products[0].Categories))
	}
	if products[0].Categories[0].Name != "Widgets" {
		t.Errorf("Categories[0].Name = %q, want %q", products[0].Categories[0].Name, "Widgets")
	}
}

func TestProductsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/15" {
			t.Errorf("path = %q, want /products/15", r.URL.Path)
		}
		w.Write([]byte(`{"id":15,"name":"Widget Pro","sku":"WP-001","price":"99.99","stock_quantity":50,"stock_status":"instock","status":"publish"}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	product, err := c.Products.Get(context.Background(), 15)
	if err != nil {
		t.Fatalf("Products.Get error: %v", err)
	}
	if product.ID != 15 {
		t.Errorf("ID = %d, want 15", product.ID)
	}
	if product.Name != "Widget Pro" {
		t.Errorf("Name = %q, want %q", product.Name, "Widget Pro")
	}
}

func TestProductsGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"woocommerce_rest_product_invalid_id","message":"Invalid ID."}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Products.Get(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProductsUpdateStock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/15" {
			t.Errorf("path = %q, want /products/15", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected request body")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Products.UpdateStock(context.Background(), 15, 75)
	if err != nil {
		t.Fatalf("Products.UpdateStock error: %v", err)
	}
}

func TestProductsUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/15" {
			t.Errorf("path = %q, want /products/15", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Products.Update(context.Background(), 15, map[string]any{
		"regular_price": "149.99",
	})
	if err != nil {
		t.Fatalf("Products.Update error: %v", err)
	}
}

// --- Webhook tests ---

func TestWebhooksList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/webhooks" {
			t.Errorf("path = %q, want /webhooks", r.URL.Path)
		}
		w.Write([]byte(`[{"id":1,"name":"OpenOMS order.created","status":"active","topic":"order.created","delivery_url":"https://example.com/hook","secret":"s3cret"}]`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	webhooks, err := c.Webhooks.List(context.Background())
	if err != nil {
		t.Fatalf("Webhooks.List error: %v", err)
	}
	if len(webhooks) != 1 {
		t.Fatalf("len(webhooks) = %d, want 1", len(webhooks))
	}
	if webhooks[0].Topic != "order.created" {
		t.Errorf("Topic = %q, want %q", webhooks[0].Topic, "order.created")
	}
}

func TestWebhooksCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/webhooks" {
			t.Errorf("path = %q, want /webhooks", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":5,"name":"OpenOMS order.created","status":"active","topic":"order.created","delivery_url":"https://example.com/hook","secret":"s3cret"}`))
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	wh, err := c.Webhooks.Create(context.Background(), "order.created", "https://example.com/hook", "s3cret")
	if err != nil {
		t.Fatalf("Webhooks.Create error: %v", err)
	}
	if wh.ID != 5 {
		t.Errorf("ID = %d, want 5", wh.ID)
	}
	if wh.Topic != "order.created" {
		t.Errorf("Topic = %q, want %q", wh.Topic, "order.created")
	}
}

func TestWebhooksDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/webhooks/5" {
			t.Errorf("path = %q, want /webhooks/5", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("https://shop.example.com", "ck", "cs",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Webhooks.Delete(context.Background(), 5)
	if err != nil {
		t.Fatalf("Webhooks.Delete error: %v", err)
	}
}
