package ebay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthorizationURL(t *testing.T) {
	c := NewClient("myAppID", "myCertID", "myDevID", "",
		WithRedirectURI("https://example.com/callback"),
	)

	got := c.AuthorizationURL("random-state-123")

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	if !strings.HasPrefix(got, productionConsentURL) {
		t.Errorf("URL prefix = %q, want %q", u.Scheme+"://"+u.Host+u.Path, productionConsentURL)
	}

	q := u.Query()

	if v := q.Get("client_id"); v != "myAppID" {
		t.Errorf("client_id = %q, want %q", v, "myAppID")
	}
	if v := q.Get("response_type"); v != "code" {
		t.Errorf("response_type = %q, want %q", v, "code")
	}
	if v := q.Get("redirect_uri"); v != "https://example.com/callback" {
		t.Errorf("redirect_uri = %q, want %q", v, "https://example.com/callback")
	}
	if v := q.Get("state"); v != "random-state-123" {
		t.Errorf("state = %q, want %q", v, "random-state-123")
	}

	scope := q.Get("scope")
	for _, want := range []string{
		"https://api.ebay.com/oauth/api_scope/sell.fulfillment",
		"https://api.ebay.com/oauth/api_scope/sell.inventory",
		"https://api.ebay.com/oauth/api_scope/sell.account",
	} {
		if !strings.Contains(scope, want) {
			t.Errorf("scope missing %q, got %q", want, scope)
		}
	}
}

func TestAuthorizationURL_Sandbox(t *testing.T) {
	c := NewClient("myAppID", "myCertID", "myDevID", "",
		WithSandbox(),
		WithRedirectURI("https://example.com/callback"),
	)

	got := c.AuthorizationURL("state-abc")

	if !strings.HasPrefix(got, sandboxConsentURL) {
		t.Errorf("URL does not start with sandbox consent URL, got %q", got)
	}
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Authorization header missing Basic prefix, got %q", auth)
		}

		if err := r.ParseForm(); err != nil { //nolint:gosec // test handler
			t.Fatalf("ParseForm: %v", err)
		}

		if v := r.FormValue("grant_type"); v != "authorization_code" { //nolint:gosec // test handler
			t.Errorf("grant_type = %q, want authorization_code", v)
		}
		if v := r.FormValue("code"); v != "test-auth-code" { //nolint:gosec // test handler
			t.Errorf("code = %q, want test-auth-code", v)
		}
		if v := r.FormValue("redirect_uri"); v != "https://example.com/callback" { //nolint:gosec // test handler
			t.Errorf("redirect_uri = %q, want https://example.com/callback", v)
		}

		resp := ExchangeCodeResponse{
			AccessToken:           "v^1.1#access-token",
			ExpiresIn:             7200,
			RefreshToken:          "v^1.1#refresh-token",
			RefreshTokenExpiresIn: 47304000,
			TokenType:             "User Access Token",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp) //nolint:gosec // G117: test response
	}))
	defer srv.Close()

	c := NewClient("appID", "certID", "devID", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithRedirectURI("https://example.com/callback"),
	)

	result, err := c.ExchangeCode(context.Background(), "test-auth-code")
	if err != nil {
		t.Fatalf("ExchangeCode error: %v", err)
	}

	if result.AccessToken != "v^1.1#access-token" {
		t.Errorf("AccessToken = %q, want %q", result.AccessToken, "v^1.1#access-token")
	}
	if result.ExpiresIn != 7200 {
		t.Errorf("ExpiresIn = %d, want 7200", result.ExpiresIn)
	}
	if result.RefreshToken != "v^1.1#refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", result.RefreshToken, "v^1.1#refresh-token")
	}
	if result.RefreshTokenExpiresIn != 47304000 {
		t.Errorf("RefreshTokenExpiresIn = %d, want 47304000", result.RefreshTokenExpiresIn)
	}
	if result.TokenType != "User Access Token" {
		t.Errorf("TokenType = %q, want %q", result.TokenType, "User Access Token")
	}
}

func TestExchangeCode_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"the authorization code is invalid"}`))
	}))
	defer srv.Close()

	c := NewClient("appID", "certID", "devID", "",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithRedirectURI("https://example.com/callback"),
	)

	result, err := c.ExchangeCode(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("error should contain HTTP 400, got %q", err.Error())
	}
}
