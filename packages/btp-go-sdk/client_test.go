package btp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// csrfTestToken is the fake CSRF token returned by test servers.
const csrfTestToken = "test-csrf-token-12345" //nolint:gosec // G101: test credential

// withCSRF wraps a handler to respond to CSRF token fetch requests.
// For GET requests with X-CSRF-TOKEN: Fetch header, it returns the token.
// All other requests are passed through to the wrapped handler.
func withCSRF(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.Header.Get("X-CSRF-TOKEN") == "Fetch" {
			w.Header().Set("X-CSRF-TOKEN", csrfTestToken)
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("user", "pass")
	if c.username != "user" || c.password != "pass" {
		t.Fatal("credentials not set")
	}
	if c.baseURL != defaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultBaseURL, c.baseURL)
	}
	if c.Inventory == nil || c.Catalogue == nil || c.Orders == nil ||
		c.Invoices == nil || c.Waybills == nil || c.ClientInfo == nil || c.Files == nil {
		t.Fatal("sub-services not initialized")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("u", "p", WithBaseURL("https://custom.example.com"))
	if c.baseURL != "https://custom.example.com" {
		t.Fatalf("expected custom base URL, got %q", c.baseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	hc := &http.Client{}
	c := NewClient("u", "p", WithHTTPClient(hc))
	if c.httpClient != hc {
		t.Fatal("custom HTTP client not set")
	}
}

func TestBasicAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Fatal("no basic auth header")
		}
		if user != "testuser" || pass != "testpass" {
			t.Fatalf("expected testuser:testpass, got %s:%s", user, pass)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("testuser", "testpass", WithBaseURL(srv.URL))
	_ = c.HealthCheck(context.Background())
}

func TestHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/HealthCheck" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	if err := c.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

func TestAPIError_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	err := c.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestAPIError_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(APIResult{
			ResultType: "ERROR",
			Messages:   []APIResultMessage{{MessageType: "ERROR", Message: "No inventory report role"}},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	err := c.HealthCheck(context.Background())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got: %v", err)
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Result == nil || apiErr.Result.Messages[0].Message != "No inventory report role" {
			t.Fatal("expected error message from API result")
		}
	}
}

func TestAPIError_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	err := c.HealthCheck(context.Background())
	if !errors.Is(err, ErrServerError) {
		t.Fatalf("expected ErrServerError, got: %v", err)
	}
}

func TestAPIError_ErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      APIError
		expected string
	}{
		{
			name: "with result messages",
			err: APIError{
				StatusCode: 403,
				Result:     &APIResult{Messages: []APIResultMessage{{Message: "No role"}}},
			},
			expected: "btp: api error 403: No role",
		},
		{
			name:     "with raw body",
			err:      APIError{StatusCode: 500, RawBody: "oops"},
			expected: "btp: api error 500: oops",
		},
		{
			name:     "bare error",
			err:      APIError{StatusCode: 502},
			expected: "btp: api error 502",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	tests := []struct {
		status   int
		sentinel error
	}{
		{401, ErrUnauthorized},
		{403, ErrForbidden},
		{404, ErrNotFound},
		{400, ErrValidation},
		{500, ErrServerError},
		{503, ErrServerError},
		{200, nil},
	}
	for _, tt := range tests {
		err := &APIError{StatusCode: tt.status}
		if got := err.Unwrap(); got != tt.sentinel {
			t.Errorf("status %d: expected %v, got %v", tt.status, tt.sentinel, got)
		}
	}
}

func TestAcceptHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected Accept: application/json, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	_ = c.HealthCheck(context.Background())
}

func TestContentTypeOnPOST(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected Content-Type: application/json on POST, got %q", got)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(OrderResponse{})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	_, _ = c.Orders.Create(context.Background(), &OrderRequest{
		Lines: []OrderRequestLine{{BarCode: "123", Quantity: 1}},
	})
}

func TestCSRFTokenSentOnPOST(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if got := r.Header.Get("X-CSRF-TOKEN"); got != csrfTestToken {
				t.Fatalf("expected CSRF token %q, got %q", csrfTestToken, got)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(OrderResponse{})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	_, err := c.Orders.Create(context.Background(), &OrderRequest{
		Lines: []OrderRequestLine{{BarCode: "123", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
