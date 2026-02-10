package allegro

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("id", "secret")
	defer c.Close()

	if c.clientID != "id" {
		t.Errorf("clientID = %q, want %q", c.clientID, "id")
	}
	if c.clientSecret != "secret" {
		t.Errorf("clientSecret = %q, want %q", c.clientSecret, "secret")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.authBaseURL != defaultAuthURL {
		t.Errorf("authBaseURL = %q, want %q", c.authBaseURL, defaultAuthURL)
	}
	if c.Orders == nil {
		t.Error("Orders service is nil")
	}
	if c.Events == nil {
		t.Error("Events service is nil")
	}
	if c.Offers == nil {
		t.Error("Offers service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("id", "secret", WithSandbox())
	defer c.Close()

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
	if c.authBaseURL != sandboxAuthURL {
		t.Errorf("authBaseURL = %q, want %q", c.authBaseURL, sandboxAuthURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("id", "secret", WithBaseURL("https://custom.api"))
	defer c.Close()

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithTokens(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	c := NewClient("id", "secret", WithTokens("at", "rt", exp))
	defer c.Close()

	if c.accessToken != "at" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "at")
	}
	if c.refreshToken != "rt" {
		t.Errorf("refreshToken = %q, want %q", c.refreshToken, "rt")
	}
}

func TestWithRedirectURI(t *testing.T) {
	c := NewClient("id", "secret", WithRedirectURI("https://example.com/cb"))
	defer c.Close()

	if c.redirectURI != "https://example.com/cb" {
		t.Errorf("redirectURI = %q, want %q", c.redirectURI, "https://example.com/cb")
	}
}

func TestDoSetsAuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithTokens("test-token", "", time.Now().Add(time.Hour)),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestDoHandlesErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"NOT_FOUND","message":"Resource not found"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

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
	if apiErr.Code != "NOT_FOUND" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "NOT_FOUND")
	}
}

func TestDoNonDecodableErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html>Bad Gateway</html>`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

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
		w.Write([]byte(`{"id":"123"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithTokens("tok", "", time.Now().Add(time.Hour)),
	)
	defer c.Close()

	body := map[string]string{"name": "test"}
	var result map[string]string
	err := c.do(context.Background(), "POST", "/items", body, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
	if gotContentType != "application/vnd.allegro.public.v1+json" {
		t.Errorf("Content-Type = %q, want application/vnd.allegro.public.v1+json", gotContentType)
	}
	if len(gotBody) == 0 {
		t.Error("expected request body, got empty")
	}
	if result["id"] != "123" {
		t.Errorf("result[id] = %q, want %q", result["id"], "123")
	}
}

func TestDoContextCancelled(t *testing.T) {
	c := NewClient("id", "secret", WithRateLimit(1))
	defer c.Close()

	// Drain the single token.
	ctx := context.Background()
	_ = c.do(ctx, "GET", "http://localhost/noop", nil, nil)

	// Now rate limiter is empty; cancelled context should fail at Wait.
	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.do(ctx2, "GET", "http://localhost/noop", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestSetTokens(t *testing.T) {
	c := NewClient("id", "secret")
	defer c.Close()

	exp := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	c.SetTokens("new-at", "new-rt", exp)

	if c.accessToken != "new-at" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "new-at")
	}
	if c.refreshToken != "new-rt" {
		t.Errorf("refreshToken = %q, want %q", c.refreshToken, "new-rt")
	}
	if !c.tokenExpiry.Equal(exp) {
		t.Errorf("tokenExpiry = %v, want %v", c.tokenExpiry, exp)
	}
}

func TestWithRateLimitOption(t *testing.T) {
	c := NewClient("id", "secret", WithRateLimit(50))
	defer c.Close()

	if c.rateLimiter == nil {
		t.Fatal("rateLimiter is nil")
	}
	// Verify custom rate limiter works by consuming a token.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.rateLimiter.Wait(ctx); err != nil {
		t.Errorf("Wait() error: %v", err)
	}
}

func TestDoDecodeResponseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json at all`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

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
			want: "allegro: HTTP 404 [NOT_FOUND]: Resource not found",
		},
		{
			name: "message only",
			err:  APIError{StatusCode: 500, Message: "Internal error"},
			want: "allegro: HTTP 500: Internal error",
		},
		{
			name: "status only",
			err:  APIError{StatusCode: 400},
			want: "allegro: HTTP 400",
		},
		{
			name: "with details",
			err: APIError{
				StatusCode: 422,
				Code:       "VALIDATION",
				Message:    "Invalid",
				Details:    []ErrorDetail{{Code: "REQUIRED", Message: "field required", Path: "name"}},
			},
			want: "allegro: HTTP 422 [VALIDATION]: Invalid\n  - REQUIRED: field required (path: name)",
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
