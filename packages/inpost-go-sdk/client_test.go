package inpost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("tok123", "org456")

	if c.token != "tok123" {
		t.Fatalf("expected token tok123, got %s", c.token)
	}
	if c.orgID != "org456" {
		t.Fatalf("expected orgID org456, got %s", c.orgID)
	}
	if c.baseURL != productionBaseURL {
		t.Fatalf("expected production URL, got %s", c.baseURL)
	}
	if c.httpClient != http.DefaultClient {
		t.Fatal("expected default http client")
	}
	if c.Shipments == nil {
		t.Fatal("Shipments service not initialized")
	}
	if c.Labels == nil {
		t.Fatal("Labels service not initialized")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("tok", "org", WithSandbox())
	if c.baseURL != sandboxBaseURL {
		t.Fatalf("expected sandbox URL, got %s", c.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("tok", "org", WithBaseURL("http://localhost:9999"))
	if c.baseURL != "http://localhost:9999" {
		t.Fatalf("expected custom URL, got %s", c.baseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("tok", "org", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Fatal("expected custom http client")
	}
}

func TestBearerTokenHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("my-secret-token", "org1", WithBaseURL(srv.URL))
	_ = c.do(context.Background(), http.MethodGet, "/test", nil, nil)

	expected := "Bearer my-secret-token"
	if gotAuth != expected {
		t.Fatalf("expected Authorization %q, got %q", expected, gotAuth)
	}
}

func TestDoReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"invalid token"}`))
	}))
	defer srv.Close()

	c := NewClient("bad", "org", WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodGet, "/test", nil, nil)

	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid token" {
		t.Fatalf("expected message 'invalid token', got %q", apiErr.Message)
	}
}

func TestURLConstruction(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org", WithBaseURL(srv.URL))
	_ = c.do(context.Background(), http.MethodGet, "/v1/shipments/123", nil, nil)

	if gotPath != "/v1/shipments/123" {
		t.Fatalf("expected path /v1/shipments/123, got %s", gotPath)
	}
}

func TestDoReturnsAPIErrorNonDecodable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html>error</html>`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org", WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodGet, "/bad", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
}

func TestDoWithRequestBody(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"created"}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org", WithBaseURL(srv.URL))
	body := map[string]string{"name": "parcel"}
	var result map[string]string
	err := c.do(context.Background(), http.MethodPost, "/shipments", body, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if result["id"] != "created" {
		t.Errorf("result[id] = %q, want %q", result["id"], "created")
	}
}

func TestDoContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org", WithBaseURL(srv.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.do(ctx, http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with message",
			err:  APIError{StatusCode: 401, Message: "invalid token"},
			want: "inpost: api error 401: invalid token",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "inpost: api error 500",
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
		{404, ErrNotFound},
		{422, ErrValidation},
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

func TestDoDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org", WithBaseURL(srv.URL))
	var result map[string]string
	err := c.do(context.Background(), http.MethodGet, "/test", nil, &result)
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestDoRawReturnsContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("%PDF-1.4"))
	}))
	defer srv.Close()

	c := NewClient("tok", "org", WithBaseURL(srv.URL))
	body, ct, err := c.doRaw(context.Background(), http.MethodGet, "/label.pdf", nil)
	if err != nil {
		t.Fatalf("doRaw() error: %v", err)
	}
	if ct != "application/pdf" {
		t.Errorf("content-type = %q, want application/pdf", ct)
	}
	if string(body) != "%PDF-1.4" {
		t.Errorf("body = %q, want %%PDF-1.4", string(body))
	}
}
