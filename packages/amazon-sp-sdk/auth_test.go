package amazon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthorizationURL(t *testing.T) {
	client := NewClient("test_id", "test_secret", //nolint:gosec // test credentials
		WithRedirectURI("https://app.example.com/marketplaces/amazon"),
	)
	u, err := client.AuthorizationURL("amzn1.sellerapps.app.123", "random-state", "A1C3SOZRARQ6R3")

	assert.NoError(t, err)
	assert.Contains(t, u, "sellercentral.amazon.pl/apps/authorize/consent")
	assert.Contains(t, u, "application_id=amzn1.sellerapps.app.123")
	assert.Contains(t, u, "state=random-state")
	assert.Contains(t, u, "redirect_uri=https")
}

func TestAuthorizationURLNoRedirectURI(t *testing.T) {
	client := NewClient("test_id", "test_secret") //nolint:gosec // test credentials
	u, err := client.AuthorizationURL("app-123", "state-1", "ATVPDKIKX0DER")

	assert.NoError(t, err)
	assert.Contains(t, u, "sellercentral.amazon.com/apps/authorize/consent")
	assert.NotContains(t, u, "redirect_uri")
}

func TestAuthorizationURLInvalidMarketplace(t *testing.T) {
	client := NewClient("test_id", "test_secret") //nolint:gosec // test credentials
	_, err := client.AuthorizationURL("app-123", "state-1", "INVALID_ID")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported marketplace ID")
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.NoError(t, r.ParseForm())                                                 //nolint:gosec // test handler
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))                 //nolint:gosec // test handler
		assert.Equal(t, "spapi-code-123", r.FormValue("code"))                           //nolint:gosec // test handler
		assert.Equal(t, "test_id", r.FormValue("client_id"))                             //nolint:gosec // test handler
		assert.Equal(t, "test_secret", r.FormValue("client_secret"))                     //nolint:gosec // test handler
		assert.Equal(t, "https://app.example.com/callback", r.FormValue("redirect_uri")) //nolint:gosec // test handler

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"new-at","refresh_token":"new-rt","expires_in":3600,"token_type":"bearer"}`)
	}))
	defer srv.Close()

	client := NewClient("test_id", "test_secret", //nolint:gosec // test credentials
		WithTokenEndpoint(srv.URL),
		WithRedirectURI("https://app.example.com/callback"),
	)
	tok, err := client.ExchangeCode(context.Background(), "spapi-code-123")

	assert.NoError(t, err)
	assert.Equal(t, "new-at", tok.AccessToken)
	assert.Equal(t, "new-rt", tok.RefreshToken)
	assert.Equal(t, 3600, tok.ExpiresIn)
}

func TestExchangeCodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"bad code"}`)
	}))
	defer srv.Close()

	client := NewClient("test_id", "test_secret", //nolint:gosec // test credentials
		WithTokenEndpoint(srv.URL),
		WithRedirectURI("https://app.example.com/callback"),
	)
	_, err := client.ExchangeCode(context.Background(), "bad-code")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
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
	client := NewClient("test_id", "test_secret", //nolint:gosec // test credentials
		WithTokenEndpoint(srv.URL),
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
	client := NewClient("test_id", "test_secret") //nolint:gosec // test credentials
	_, err := client.RefreshAccessToken(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no refresh token")
}

func TestSetTokens(t *testing.T) {
	client := NewClient("test_id", "test_secret") //nolint:gosec // test credentials
	expiry := time.Now().Add(2 * time.Hour)
	client.SetTokens("at", "rt", expiry)

	client.mu.Lock()
	assert.Equal(t, "at", client.accessToken)
	assert.Equal(t, "rt", client.refreshToken)
	assert.Equal(t, expiry, client.tokenExpiry)
	client.mu.Unlock()
}
