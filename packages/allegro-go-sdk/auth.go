package allegro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthorizationURL builds the URL a user should visit to authorize the application.
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
	return c.authBaseURL + "/authorize?" + v.Encode()
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.redirectURI},
	}

	tok, err := c.postToken(ctx, data)
	if err != nil {
		return nil, err
	}

	c.applyTokenResponse(tok)
	return tok, nil
}

// RefreshAccessToken refreshes the access token using the stored refresh token.
func (c *Client) RefreshAccessToken(ctx context.Context) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.refreshToken},
	}

	tok, err := c.postToken(ctx, data)
	if err != nil {
		return nil, err
	}

	c.applyTokenResponse(tok)

	if c.onTokenRefresh != nil {
		c.onTokenRefresh(c.accessToken, c.refreshToken, c.tokenExpiry)
	}

	return tok, nil
}

// SetTokens manually updates the stored OAuth tokens.
func (c *Client) SetTokens(accessToken, refreshToken string, expiry time.Time) {
	c.accessToken = accessToken
	c.refreshToken = refreshToken
	c.tokenExpiry = expiry
}

func (c *Client) postToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	endpoint := c.authBaseURL + "/token"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("allegro: create token request: %w", err)
	}

	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("allegro: execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return nil, apiErr
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("allegro: decode token response: %w", err)
	}

	return &tok, nil
}

func (c *Client) applyTokenResponse(tok *TokenResponse) {
	c.accessToken = tok.AccessToken
	c.refreshToken = tok.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
}
