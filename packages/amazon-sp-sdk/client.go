package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	httpClient    *http.Client
	baseURL       string
	tokenEndpoint string
	clientID      string
	clientSecret  string
	refreshToken  string
	redirectURI   string

	mu             sync.Mutex
	accessToken    string
	tokenExpiry    time.Time
	onTokenRefresh func(accessToken, refreshToken string, expiry time.Time)

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

// WithRedirectURI sets the OAuth redirect URI for authorization code flow.
func WithRedirectURI(uri string) Option {
	return func(cl *Client) {
		cl.redirectURI = uri
	}
}

// WithOnTokenRefresh sets a callback invoked after a token refresh.
func WithOnTokenRefresh(fn func(accessToken, refreshToken string, expiry time.Time)) Option {
	return func(cl *Client) {
		cl.onTokenRefresh = fn
	}
}

// WithTokenEndpoint overrides the LWA token endpoint (useful for testing).
func WithTokenEndpoint(u string) Option {
	return func(cl *Client) {
		cl.tokenEndpoint = u
	}
}

// NewClient creates a new Amazon SP-API client.
func NewClient(clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient:    http.DefaultClient,
		baseURL:       productionBaseURL,
		tokenEndpoint: lwaTokenEndpoint,
		clientID:      clientID,
		clientSecret:  clientSecret,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Catalog = &CatalogService{client: c}

	return c
}

// ensureToken refreshes the access token if it is expired or near expiry.
func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	needsRefresh := c.accessToken == "" || time.Until(c.tokenExpiry) < 60*time.Second
	c.mu.Unlock()

	if needsRefresh {
		_, err := c.RefreshAccessToken(ctx)
		return err
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

	c.mu.Lock()
	token := c.accessToken
	c.mu.Unlock()

	req.Header.Set("x-amz-access-token", token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("amazon: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
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
