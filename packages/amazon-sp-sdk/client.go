package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	productionBaseURL = "https://sellingpartnerapi-eu.amazon.com"
	sandboxBaseURL    = "https://sandbox.sellingpartnerapi-eu.amazon.com"
	lwaTokenEndpoint  = "https://api.amazon.com/auth/o2/token"
)

// Client is an Amazon SP-API client with LWA (Login with Amazon) OAuth2 auth.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	clientID     string
	clientSecret string
	refreshToken string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time

	Orders  *OrderService
	Catalog *CatalogService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithSandbox configures the client for the Amazon sandbox environment.
func WithSandbox() Option {
	return func(cl *Client) {
		cl.baseURL = sandboxBaseURL
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(cl *Client) {
		cl.baseURL = u
	}
}

// WithRefreshToken sets the LWA refresh token for API authorization.
func WithRefreshToken(token string) Option {
	return func(cl *Client) {
		cl.refreshToken = token
	}
}

// WithTokens sets initial access and refresh tokens with an expiry time.
func WithTokens(accessToken, refreshToken string, expiry time.Time) Option {
	return func(cl *Client) {
		cl.accessToken = accessToken
		cl.refreshToken = refreshToken
		cl.tokenExpiry = expiry
	}
}

// NewClient creates a new Amazon SP-API client.
func NewClient(clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient:   http.DefaultClient,
		baseURL:      productionBaseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Catalog = &CatalogService{client: c}

	return c
}

// refreshAccessToken obtains a new access token using the LWA refresh token.
func (c *Client) refreshAccessToken(ctx context.Context) error {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.refreshToken},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, lwaTokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("amazon: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("amazon: execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("amazon: token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return fmt.Errorf("amazon: decode token response: %w", err)
	}

	c.accessToken = tok.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	return nil
}

// ensureToken refreshes the access token if it is expired or near expiry.
func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Refresh if token expires within 60 seconds.
	if c.accessToken == "" || time.Now().Add(60*time.Second).After(c.tokenExpiry) {
		return c.refreshAccessToken(ctx)
	}
	return nil
}

// do performs an authenticated API request and decodes the JSON response.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("amazon: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("amazon: create request: %w", err)
	}

	req.Header.Set("x-amz-access-token", c.accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("amazon: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("amazon: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
		}
		return apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("amazon: decode response: %w", err)
		}
	}

	return nil
}
