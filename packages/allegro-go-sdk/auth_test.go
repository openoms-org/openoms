package allegro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthorizationURL(t *testing.T) {
	c := NewClient("my-client-id", "secret",
		WithRedirectURI("https://example.com/callback"),
	)
	defer c.Close()

	u := c.AuthorizationURL("random-state", "allegro:api:orders:read")

	if !strings.HasPrefix(u, defaultAuthURL+"/authorize?") {
		t.Errorf("URL prefix mismatch: %s", u)
	}
	if !strings.Contains(u, "client_id=my-client-id") {
		t.Error("missing client_id parameter")
	}
	if !strings.Contains(u, "response_type=code") {
		t.Error("missing response_type parameter")
	}
	if !strings.Contains(u, "state=random-state") {
		t.Error("missing state parameter")
	}
	if !strings.Contains(u, "scope=allegro") {
		t.Error("missing scope parameter")
	}
}

func TestAuthorizationURLNoScopes(t *testing.T) {
	c := NewClient("id", "secret")
	defer c.Close()

	u := c.AuthorizationURL("state")
	if strings.Contains(u, "scope=") {
		t.Errorf("scope should not be present when no scopes given: %s", u)
	}
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "cid" || pass != "csecret" {
			t.Errorf("BasicAuth = (%q, %q, %v), want (cid, csecret, true)", user, pass, ok)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm error: %v", err)
		}
		if gt := r.FormValue("grant_type"); gt != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", gt)
		}
		if code := r.FormValue("code"); code != "auth-code-123" {
			t.Errorf("code = %q, want auth-code-123", code)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"access_token": "new-at",
			"token_type": "bearer",
			"refresh_token": "new-rt",
			"expires_in": 3600,
			"scope": "allegro:api:orders:read",
			"jti": "jti-123"
		}`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret",
		WithHTTPClient(srv.Client()),
	)
	c.authBaseURL = srv.URL
	defer c.Close()

	tok, err := c.ExchangeCode(context.Background(), "auth-code-123")
	if err != nil {
		t.Fatalf("ExchangeCode error: %v", err)
	}
	if tok.AccessToken != "new-at" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "new-at")
	}
	if c.accessToken != "new-at" {
		t.Errorf("client accessToken = %q, want %q", c.accessToken, "new-at")
	}
	if c.refreshToken != "new-rt" {
		t.Errorf("client refreshToken = %q, want %q", c.refreshToken, "new-rt")
	}
}

func TestExchangeCodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"invalid_grant"}`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret", WithHTTPClient(srv.Client()))
	c.authBaseURL = srv.URL
	defer c.Close()

	_, err := c.ExchangeCode(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestPostTokenNonJSONErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html>Bad Gateway</html>`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret", WithHTTPClient(srv.Client()))
	c.authBaseURL = srv.URL
	defer c.Close()

	_, err := c.ExchangeCode(context.Background(), "code")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Message != "Bad Gateway" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Bad Gateway")
	}
}

func TestPostTokenBadJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret", WithHTTPClient(srv.Client()))
	c.authBaseURL = srv.URL
	defer c.Close()

	_, err := c.ExchangeCode(context.Background(), "code")
	if err == nil {
		t.Fatal("expected error from bad JSON token response")
	}
}

func TestRefreshAccessTokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"invalid refresh token"}`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret", WithHTTPClient(srv.Client()))
	c.authBaseURL = srv.URL
	defer c.Close()

	_, err := c.RefreshAccessToken(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

func TestRefreshAccessToken(t *testing.T) {
	var callbackCalled bool
	var callbackAT, callbackRT string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm error: %v", err)
		}
		if gt := r.FormValue("grant_type"); gt != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", gt)
		}
		if rt := r.FormValue("refresh_token"); rt != "old-rt" {
			t.Errorf("refresh_token = %q, want old-rt", rt)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"access_token": "refreshed-at",
			"token_type": "bearer",
			"refresh_token": "refreshed-rt",
			"expires_in": 7200
		}`))
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret",
		WithHTTPClient(srv.Client()),
		WithTokens("old-at", "old-rt", time.Now().Add(-time.Hour)),
		WithOnTokenRefresh(func(at, rt string, exp time.Time) {
			callbackCalled = true
			callbackAT = at
			callbackRT = rt
		}),
	)
	c.authBaseURL = srv.URL
	defer c.Close()

	tok, err := c.RefreshAccessToken(context.Background())
	if err != nil {
		t.Fatalf("RefreshAccessToken error: %v", err)
	}
	if tok.AccessToken != "refreshed-at" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "refreshed-at")
	}
	if !callbackCalled {
		t.Error("onTokenRefresh callback was not called")
	}
	if callbackAT != "refreshed-at" {
		t.Errorf("callback accessToken = %q, want %q", callbackAT, "refreshed-at")
	}
	if callbackRT != "refreshed-rt" {
		t.Errorf("callback refreshToken = %q, want %q", callbackRT, "refreshed-rt")
	}
}
