package fedex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("cid", "csec", "acc123")

	if c.clientID != "cid" {
		t.Fatalf("expected clientID cid, got %s", c.clientID)
	}
	if c.clientSecret != "csec" {
		t.Fatalf("expected clientSecret csec, got %s", c.clientSecret)
	}
	if c.accountNumber != "acc123" {
		t.Fatalf("expected accountNumber acc123, got %s", c.accountNumber)
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
}

func TestAccountNumber(t *testing.T) {
	c := NewClient("cid", "csec", "myacc")
	if c.AccountNumber() != "myacc" {
		t.Fatalf("expected AccountNumber() = myacc, got %s", c.AccountNumber())
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("cid", "csec", "acc", WithSandbox())
	if c.baseURL != sandboxBaseURL {
		t.Fatalf("expected sandbox URL, got %s", c.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("cid", "csec", "acc", WithBaseURL("http://localhost:9999"))
	if c.baseURL != "http://localhost:9999" {
		t.Fatalf("expected custom URL, got %s", c.baseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("cid", "csec", "acc", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Fatal("expected custom http client")
	}
}

func TestOAuth2Authentication(t *testing.T) {
	var authCalled bool
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			authCalled = true
			gotContentType = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(tokenResponse{
				AccessToken: "test-token-123",
				ExpiresIn:   3600,
				TokenType:   "bearer",
			})
			return
		}

		// Verify the bearer token is sent
		gotAuth := r.Header.Get("Authorization")
		if gotAuth != "Bearer test-token-123" {
			t.Errorf("expected Authorization 'Bearer test-token-123', got %q", gotAuth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("my-client-id", "my-client-secret", "acc123", WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !authCalled {
		t.Fatal("expected OAuth2 token endpoint to be called")
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("expected Content-Type application/x-www-form-urlencoded, got %q", gotContentType)
	}
}

func TestDoReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(tokenResponse{
				AccessToken: "tok",
				ExpiresIn:   3600,
			})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"invalid request","code":"INVALID"}`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csec", "acc", WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodPost, "/ship/v1/shipments", nil, nil)

	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid request" {
		t.Fatalf("expected message 'invalid request', got %q", apiErr.Message)
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
			err:  APIError{StatusCode: 400, Message: "bad request"},
			want: "fedex: api error 400: bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "fedex: api error 500",
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
		input   string
		wantOMS string
		wantOK  bool
	}{
		{"DL", "delivered", true},
		{"IT", "in_transit", true},
		{"PU", "picked_up", true},
		{"OD", "out_for_delivery", true},
		{"UNKNOWN", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.input)
		if ok != tc.wantOK || oms != tc.wantOMS {
			t.Errorf("MapStatus(%q) = (%q, %v), want (%q, %v)", tc.input, oms, ok, tc.wantOMS, tc.wantOK)
		}
	}
}
