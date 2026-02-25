package shoper

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient("https://shop.shoper.pl", "client-id", "client-secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test-token"),
	)
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("https://myshop.shoper.pl", "cid", "csecret")

	if c.clientID != "cid" {
		t.Errorf("clientID = %q, want %q", c.clientID, "cid")
	}
	if c.clientSecret != "csecret" {
		t.Errorf("clientSecret = %q, want %q", c.clientSecret, "csecret")
	}
	if c.baseURL != "https://myshop.shoper.pl/webapi/rest" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://myshop.shoper.pl/webapi/rest")
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Products == nil {
		t.Error("Products service is nil")
	}
	if c.Categories == nil {
		t.Error("Categories service is nil")
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://myshop.shoper.pl/", "cid", "csecret")

	if c.baseURL != "https://myshop.shoper.pl/webapi/rest" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://myshop.shoper.pl/webapi/rest")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("https://shop.shoper.pl", "cid", "csecret",
		WithBaseURL("https://custom.api/rest"))

	if c.baseURL != "https://custom.api/rest" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api/rest")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("https://shop.shoper.pl", "cid", "csecret", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set by WithHTTPClient")
	}
}

func TestWithAccessToken(t *testing.T) {
	c := NewClient("https://shop.shoper.pl", "cid", "csecret",
		WithAccessToken("pre-obtained-token"))

	if c.accessToken != "pre-obtained-token" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "pre-obtained-token")
	}
}

func TestDoSetsBearerToken(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
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
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

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

func TestAuthenticate(t *testing.T) {
	var authCalled atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/webapi/rest/auth" {
			authCalled.Add(1)
			user, pass, ok := r.BasicAuth()
			if !ok {
				t.Error("expected Basic Auth on auth endpoint")
			}
			if user != "cid" {
				t.Errorf("auth user = %q, want %q", user, "cid")
			}
			if pass != "csecret" {
				t.Errorf("auth pass = %q, want %q", pass, "csecret")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "fresh-token",
				"expires_in":   3600,
			})
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer fresh-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer fresh-token")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"order_id": 1})
	}))
	defer srv.Close()

	// Create client without pre-set token to force authentication
	c := NewClient("https://shop.shoper.pl", "cid", "csecret",
		WithBaseURL(srv.URL+"/webapi/rest"),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/orders/1", nil, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}

	if authCalled.Load() != 1 {
		t.Errorf("auth called %d times, want 1", authCalled.Load())
	}
}

func TestAuthenticateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_client",
			"error_description": "Bad credentials",
		})
	}))
	defer srv.Close()

	c := NewClient("https://shop.shoper.pl", "bad-id", "bad-secret",
		WithBaseURL(srv.URL+"/webapi/rest"),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/orders/1", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

func TestOrdersList(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/orders" {
			t.Errorf("path = %q, want /orders", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse[ShoperOrder]{
			Count: 2,
			Pages: 1,
			Page:  1,
			List: []ShoperOrder{
				{ID: 1, OrderSerial: "ORD-001", Sum: 99.99},
				{ID: 2, OrderSerial: "ORD-002", Sum: 49.99},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	result, err := c.Orders.List(context.Background(), OrderListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.List) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(result.List))
	}
	if result.List[0].OrderSerial != "ORD-001" {
		t.Errorf("List[0].OrderSerial = %q, want %q", result.List[0].OrderSerial, "ORD-001")
	}
}

func TestOrdersListWithParams(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("page") != "2" {
			t.Errorf("page = %q, want 2", q.Get("page"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", q.Get("limit"))
		}
		if q.Get("filters[status_id]") != "5" {
			t.Errorf("filters[status_id] = %q, want 5", q.Get("filters[status_id]"))
		}
		if q.Get("sort") != "date_modify" {
			t.Errorf("sort = %q, want date_modify", q.Get("sort"))
		}
		if q.Get("order") != "desc" {
			t.Errorf("order = %q, want desc", q.Get("order"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse[ShoperOrder]{Count: 0, List: []ShoperOrder{}})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.List(context.Background(), OrderListParams{
		Page:          2,
		Limit:         10,
		StatusID:      5,
		SortBy:        "date_modify",
		SortDirection: "desc",
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
		if r.URL.Path != "/orders/42" {
			t.Errorf("path = %q, want /orders/42", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ShoperOrder{
			ID:          42,
			OrderSerial: "ORD-042",
			Email:       "jan@example.com",
			Sum:         199.99,
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	order, err := c.Orders.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if order.ID != 42 {
		t.Errorf("ID = %d, want 42", order.ID)
	}
	if order.Email != "jan@example.com" {
		t.Errorf("Email = %q, want %q", order.Email, "jan@example.com")
	}
}

func TestOrdersUpdateStatus(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/orders/42" {
			t.Errorf("path = %q, want /orders/42", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["status_id"] != float64(3) {
			t.Errorf("status_id = %v, want 3", body["status_id"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Orders.UpdateStatus(context.Background(), 42, 3)
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}
}

func TestProductsList(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/products" {
			t.Errorf("path = %q, want /products", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse[ShoperProduct]{
			Count: 1,
			Pages: 1,
			Page:  1,
			List: []ShoperProduct{
				{ID: 1, Code: "SKU-001", EAN: "5901234123457", Price: 29.99},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	result, err := c.Products.List(context.Background(), ProductListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result.List) != 1 {
		t.Fatalf("len(List) = %d, want 1", len(result.List))
	}
	if result.List[0].Code != "SKU-001" {
		t.Errorf("Code = %q, want %q", result.List[0].Code, "SKU-001")
	}
}

func TestProductsGet(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/7" {
			t.Errorf("path = %q, want /products/7", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ShoperProduct{
			ID:    7,
			Code:  "SKU-007",
			EAN:   "5901234123457",
			Price: 149.99,
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	product, err := c.Products.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if product.ID != 7 {
		t.Errorf("ID = %d, want 7", product.ID)
	}
	if product.Code != "SKU-007" {
		t.Errorf("Code = %q, want %q", product.Code, "SKU-007")
	}
}

func TestProductsUpdate(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/products/7" {
			t.Errorf("path = %q, want /products/7", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["ean"] != "9999999999999" {
			t.Errorf("ean = %v, want 9999999999999", body["ean"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Products.Update(context.Background(), 7, map[string]any{"ean": "9999999999999"})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestProductsUpdateStock(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/products/7" {
			t.Errorf("path = %q, want /products/7", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		stock, ok := body["stock"].(map[string]any)
		if !ok {
			t.Fatal("missing stock key in body")
		}
		if stock["stock"] != float64(50) {
			t.Errorf("stock.stock = %v, want 50", stock["stock"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Products.UpdateStock(context.Background(), 7, 50)
	if err != nil {
		t.Fatalf("UpdateStock() error: %v", err)
	}
}

func TestCategoriesList(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/categories" {
			t.Errorf("path = %q, want /categories", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse[ShoperCategory]{
			Count: 2,
			Pages: 1,
			Page:  1,
			List: []ShoperCategory{
				{ID: 1, Name: "Elektronika", ParentID: 0},
				{ID: 2, Name: "Telefony", ParentID: 1},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	result, err := c.Categories.List(context.Background(), CategoryListParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result.List) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(result.List))
	}
	if result.List[0].Name != "Elektronika" {
		t.Errorf("List[0].Name = %q, want %q", result.List[0].Name, "Elektronika")
	}
	if result.List[1].ParentID != 1 {
		t.Errorf("List[1].ParentID = %d, want 1", result.List[1].ParentID)
	}
}

func TestCategoriesListWithParams(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("page") != "3" {
			t.Errorf("page = %q, want 3", q.Get("page"))
		}
		if q.Get("limit") != "25" {
			t.Errorf("limit = %q, want 25", q.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse[ShoperCategory]{Count: 0, List: []ShoperCategory{}})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Categories.List(context.Background(), CategoryListParams{
		Page:  3,
		Limit: 25,
	})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
}

func TestCategoriesGet(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/categories/5" {
			t.Errorf("path = %q, want /categories/5", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ShoperCategory{
			ID:       5,
			Name:     "Odzież",
			ParentID: 1,
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	cat, err := c.Categories.Get(context.Background(), 5)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if cat.ID != 5 {
		t.Errorf("ID = %d, want 5", cat.ID)
	}
	if cat.Name != "Odzież" {
		t.Errorf("Name = %q, want %q", cat.Name, "Odzież")
	}
}

func TestServerError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "server_error",
			"error_description": "Internal server error",
		})
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
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "not_found",
			"error_description": "Order not found",
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.Get(context.Background(), 999)
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

func TestRateLimitedError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "rate_limit",
			"error_description": "Too many requests",
		})
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

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with code and message",
			err:  APIError{StatusCode: 400, Code: "invalid_request", Message: "Bad request"},
			want: "shoper: HTTP 400 [invalid_request]: Bad request",
		},
		{
			name: "with message only",
			err:  APIError{StatusCode: 500, Message: "Server error"},
			want: "shoper: HTTP 500: Server error",
		},
		{
			name: "status code only",
			err:  APIError{StatusCode: 503},
			want: "shoper: HTTP 503",
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
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Orders.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestProductsListWithFilters(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("filters[category_id]") != "5" {
			t.Errorf("filters[category_id] = %q, want 5", q.Get("filters[category_id]"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse[ShoperProduct]{Count: 0, List: []ShoperProduct{}})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Products.List(context.Background(), ProductListParams{
		Filters: map[string]string{"category_id": "5"},
	})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
}
