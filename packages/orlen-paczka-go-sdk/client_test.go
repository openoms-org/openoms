package orlenpaczka

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("key123", "partner456")

	if c.apiKey != "key123" {
		t.Fatalf("expected apiKey key123, got %s", c.apiKey)
	}
	if c.partnerID != "partner456" {
		t.Fatalf("expected partnerID partner456, got %s", c.partnerID)
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
	if c.Points == nil {
		t.Fatal("Points service not initialized")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("key", "partner", WithSandbox())
	if c.baseURL != sandboxBaseURL {
		t.Fatalf("expected sandbox URL, got %s", c.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("key", "partner", WithBaseURL("http://localhost:9999"))
	if c.baseURL != "http://localhost:9999" {
		t.Fatalf("expected custom URL, got %s", c.baseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("key", "partner", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Fatal("expected custom http client")
	}
}

func TestAuthHeaders(t *testing.T) {
	var gotAuth, gotPartner string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPartner = r.Header.Get("X-Partner-ID")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("my-api-key", "partner1", WithBaseURL(srv.URL))
	_ = c.do(context.Background(), http.MethodGet, "/test", nil, nil)

	expectedAuth := "Bearer my-api-key"
	if gotAuth != expectedAuth {
		t.Fatalf("expected Authorization %q, got %q", expectedAuth, gotAuth)
	}
	if gotPartner != "partner1" {
		t.Fatalf("expected X-Partner-ID %q, got %q", "partner1", gotPartner)
	}
}

func TestDoReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"access denied"}`))
	}))
	defer srv.Close()

	c := NewClient("bad", "partner", WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodGet, "/test", nil, nil)

	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "access denied" {
		t.Fatalf("expected message 'access denied', got %q", apiErr.Message)
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
			err:  APIError{StatusCode: 403, Message: "access denied"},
			want: "orlenpaczka: api error 403: access denied",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "orlenpaczka: api error 500",
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
		input    string
		wantOMS  string
		wantOK   bool
	}{
		{"DELIVERED", "delivered", true},
		{"IN_TRANSIT", "in_transit", true},
		{"READY_FOR_PICKUP", "out_for_delivery", true},
		{"UNKNOWN_STATUS", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.input)
		if ok != tc.wantOK || oms != tc.wantOMS {
			t.Errorf("MapStatus(%q) = (%q, %v), want (%q, %v)", tc.input, oms, ok, tc.wantOMS, tc.wantOK)
		}
	}
}
