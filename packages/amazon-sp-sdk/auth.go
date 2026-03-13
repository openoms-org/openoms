package amazon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SellerCentralURLs maps Amazon marketplace IDs to their Seller Central base URLs.
var SellerCentralURLs = map[string]string{
	"A1C3SOZRARQ6R3": "https://sellercentral.amazon.pl",
	"A1PA6795UKMFR9": "https://sellercentral.amazon.de",
	"A1F83G8C2ARO7P": "https://sellercentral.amazon.co.uk",
	"A13V1IB3VIYZZH": "https://sellercentral.amazon.fr",
	"APJ6JRA9NG5V4":  "https://sellercentral.amazon.it",
	"A1RKKUPIHCS9HS": "https://sellercentral.amazon.es",
	"A21TJRUUN4KGV":  "https://sellercentral.amazon.in",
	"ATVPDKIKX0DER":  "https://sellercentral.amazon.com",
}

// AuthorizationURL builds the Amazon Seller Central OAuth authorization URL.
// applicationID is the SP-API application ID (not the LWA client_id).
// marketplaceID determines which Seller Central domain to use.
func (c *Client) AuthorizationURL(applicationID, state, marketplaceID string) (string, error) {
	scURL, ok := SellerCentralURLs[marketplaceID]
	if !ok {
		return "", fmt.Errorf("amazon: unsupported marketplace ID %q", marketplaceID)
	}
	v := url.Values{
		"application_id": {applicationID},
		"state":          {state},
	}
	if c.redirectURI != "" {
		v.Set("redirect_uri", c.redirectURI)
	}
	return scURL + "/apps/authorize/consent?" + v.Encode(), nil
}

// ExchangeCode exchanges an SP-API authorization code (spapi_oauth_code) for tokens.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.redirectURI},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}
	return c.postToken(ctx, data)
}

// RefreshAccessToken refreshes the access token using the stored refresh token.
func (c *Client) RefreshAccessToken(ctx context.Context) (*TokenResponse, error) {
	c.mu.Lock()
	rt := c.refreshToken
	c.mu.Unlock()

	if rt == "" {
		return nil, fmt.Errorf("amazon: no refresh token available")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {rt},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	tok, err := c.postToken(ctx, data)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.accessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		c.refreshToken = tok.RefreshToken
	}
	c.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	callback := c.onTokenRefresh
	at := c.accessToken
	newRT := c.refreshToken
	expiry := c.tokenExpiry
	c.mu.Unlock()

	if callback != nil {
		callback(at, newRT, expiry)
	}

	return tok, nil
}

// SetTokens manually updates the stored OAuth tokens.
func (c *Client) SetTokens(accessToken, refreshToken string, expiry time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = accessToken
	c.refreshToken = refreshToken
	c.tokenExpiry = expiry
}

func (c *Client) postToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("amazon: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("amazon: token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, fmt.Errorf("amazon: token request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&tok); err != nil {
		return nil, fmt.Errorf("amazon: decode token response: %w", err)
	}
	return &tok, nil
}
