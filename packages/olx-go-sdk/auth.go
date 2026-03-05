package olx

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

// TokenResponse is the response from the OLX OAuth2 token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope,omitempty"`
}

// AuthorizationURL builds the OAuth2 authorization URL the user should visit.
func (c *Client) AuthorizationURL(state string, scopes ...string) string {
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {c.clientID},
		"redirect_uri":  {c.redirectURI},
		"state":         {state},
	}
	if len(scopes) > 0 {
		v.Set("scope", strings.Join(scopes, " "))
	}
	return c.authorizeURL + "?" + v.Encode()
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
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
	c.tokenMu.Lock()
	rt := c.refreshToken
	c.tokenMu.Unlock()

	if rt == "" {
		return nil, fmt.Errorf("olx: no refresh token available")
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

	c.tokenMu.Lock()
	c.accessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		c.refreshToken = tok.RefreshToken
	}
	c.tokenExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	callback := c.onTokenRefresh
	at := c.accessToken
	newRT := c.refreshToken
	expiry := c.tokenExpiresAt
	c.tokenMu.Unlock()

	if callback != nil {
		callback(at, newRT, expiry)
	}

	return tok, nil
}

// SetTokens manually updates the stored OAuth tokens.
func (c *Client) SetTokens(accessToken, refreshToken string, expiry time.Time) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.accessToken = accessToken
	c.refreshToken = refreshToken
	c.tokenExpiresAt = expiry
}

func (c *Client) postToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("olx: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("olx: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return nil, fmt.Errorf("olx: token request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(&tok); err != nil {
		return nil, fmt.Errorf("olx: decode token response: %w", err)
	}
	return &tok, nil
}
