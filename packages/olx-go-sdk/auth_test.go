package olx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthorizationURL(t *testing.T) {
	client := NewClient("test_id", "test_secret", "", //nolint:gosec // test credentials
		WithRedirectURI("https://app.example.com/callback"),
		WithBaseURL("https://olx.example.com"),
	)
	u := client.AuthorizationURL("random-state")

	assert.Contains(t, u, "response_type=code")
	assert.Contains(t, u, "client_id=test_id")
	assert.Contains(t, u, "redirect_uri=https")
	assert.Contains(t, u, "state=random-state")
	assert.Contains(t, u, "olx.example.com/oauth/authorize")
}

func TestAuthorizationURLWithScopes(t *testing.T) {
	client := NewClient("test_id", "test_secret", "", //nolint:gosec // test credentials
		WithRedirectURI("https://app.example.com/callback"),
		WithBaseURL("https://olx.example.com"),
	)
	u := client.AuthorizationURL("state-123", "read", "write")

	assert.Contains(t, u, "scope=read+write")
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.NoError(t, r.ParseForm())                                 //nolint:gosec // test handler
		assert.Equal(t, "authorization_code", r.FormValue("grant_type")) //nolint:gosec // test handler
		assert.Equal(t, "test-code", r.FormValue("code"))                //nolint:gosec // test handler
		assert.Equal(t, "test_id", r.FormValue("client_id"))             //nolint:gosec // test handler
		assert.Equal(t, "test_secret", r.FormValue("client_secret"))     //nolint:gosec // test handler

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"new-at","refresh_token":"new-rt","expires_in":3600,"token_type":"bearer"}`)
	}))
	defer srv.Close()

	client := NewClient("test_id", "test_secret", "", //nolint:gosec // test credentials
		WithBaseURL(srv.URL),
		WithRedirectURI("https://app.example.com/callback"),
	)
	tok, err := client.ExchangeCode(context.Background(), "test-code")

	assert.NoError(t, err)
	assert.Equal(t, "new-at", tok.AccessToken)
	assert.Equal(t, "new-rt", tok.RefreshToken)
	assert.Equal(t, 3600, tok.ExpiresIn)
}

func TestExchangeCodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","message":"bad code"}`)
	}))
	defer srv.Close()

	client := NewClient("test_id", "test_secret", "", //nolint:gosec // test credentials
		WithBaseURL(srv.URL),
		WithRedirectURI("https://app.example.com/callback"),
	)
	_, err := client.ExchangeCode(context.Background(), "bad-code")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.True(t, errors.Is(err, ErrInvalidGrant))
}

func TestRefreshAccessTokenInvalidGrant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"Refresh token has expired"}`)
	}))
	defer srv.Close()

	client := NewClient("test_id", "test_secret", "old-at", //nolint:gosec // test credentials
		WithBaseURL(srv.URL),
		WithTokens("old-at", "expired-rt", time.Now().Add(-time.Hour)),
	)
	_, err := client.RefreshAccessToken(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidGrant))
	assert.Contains(t, err.Error(), "invalid_grant")
}

func TestRefreshAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())                            //nolint:gosec // test handler
		assert.Equal(t, "refresh_token", r.FormValue("grant_type")) //nolint:gosec // test handler
		assert.Equal(t, "old-rt", r.FormValue("refresh_token"))     //nolint:gosec // test handler
		assert.Equal(t, "test_id", r.FormValue("client_id"))        //nolint:gosec // test handler

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"refreshed-at","refresh_token":"refreshed-rt","expires_in":3600}`)
	}))
	defer srv.Close()

	var callbackCalled bool
	client := NewClient("test_id", "test_secret", "old-at", //nolint:gosec // test credentials
		WithBaseURL(srv.URL),
		WithTokens("old-at", "old-rt", time.Now().Add(time.Hour)),
		WithOnTokenRefresh(func(at, rt string, _ time.Time) {
			callbackCalled = true
			assert.Equal(t, "refreshed-at", at)
			assert.Equal(t, "refreshed-rt", rt)
		}),
	)
	tok, err := client.RefreshAccessToken(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "refreshed-at", tok.AccessToken)
	assert.Equal(t, "refreshed-rt", tok.RefreshToken)
	assert.True(t, callbackCalled)
}

func TestRefreshAccessTokenNoRefreshToken(t *testing.T) {
	client := NewClient("test_id", "test_secret", "at") //nolint:gosec // test credentials
	_, err := client.RefreshAccessToken(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no refresh token")
}

func TestSetTokens(t *testing.T) {
	client := NewClient("test_id", "test_secret", "") //nolint:gosec // test credentials
	expiry := time.Now().Add(2 * time.Hour)
	client.SetTokens("at", "rt", expiry)

	client.tokenMu.Lock()
	assert.Equal(t, "at", client.accessToken)
	assert.Equal(t, "rt", client.refreshToken)
	assert.Equal(t, expiry, client.tokenExpiresAt)
	client.tokenMu.Unlock()
}
