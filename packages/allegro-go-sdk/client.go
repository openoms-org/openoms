package allegro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL    = "https://api.allegro.pl"
	defaultAuthURL    = "https://allegro.pl/auth/oauth"
	sandboxBaseURL    = "https://api.allegro.pl.allegrosandbox.pl"
	sandboxAuthURL    = "https://allegro.pl.allegrosandbox.pl/auth/oauth"
	defaultRateLimit  = 150
	defaultRedirectURI = "https://localhost/callback"
)

// Client is the Allegro API client.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	authBaseURL    string
	clientID       string
	clientSecret   string
	redirectURI    string
	accessToken    string
	refreshToken   string
	tokenExpiry    time.Time
	onTokenRefresh func(accessToken, refreshToken string, expiry time.Time)
	rateLimiter    *rateLimiter

	Orders *OrderService
	Events *EventService
	Offers *OfferService
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new Allegro API client.
func NewClient(clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient:   http.DefaultClient,
		baseURL:      defaultBaseURL,
		authBaseURL:  defaultAuthURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  defaultRedirectURI,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.rateLimiter == nil {
		c.rateLimiter = newRateLimiter(defaultRateLimit)
	}

	c.Orders = &OrderService{client: c}
	c.Events = &EventService{client: c}
	c.Offers = &OfferService{client: c}

	return c
}

// Close releases resources held by the client (e.g. rate limiter goroutine).
func (c *Client) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Close()
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithSandbox configures the client for the Allegro sandbox environment.
func WithSandbox() Option {
	return func(c *Client) {
		c.baseURL = sandboxBaseURL
		c.authBaseURL = sandboxAuthURL
	}
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithRedirectURI sets the OAuth redirect URI.
func WithRedirectURI(uri string) Option {
	return func(c *Client) {
		c.redirectURI = uri
	}
}

// WithTokens sets initial OAuth tokens.
func WithTokens(accessToken, refreshToken string, expiry time.Time) Option {
	return func(c *Client) {
		c.accessToken = accessToken
		c.refreshToken = refreshToken
		c.tokenExpiry = expiry
	}
}

// WithOnTokenRefresh registers a callback invoked when tokens are refreshed.
func WithOnTokenRefresh(fn func(string, string, time.Time)) Option {
	return func(c *Client) {
		c.onTokenRefresh = fn
	}
}

// WithRateLimit sets the rate limiter to the given requests per minute.
func WithRateLimit(requestsPerMinute int) Option {
	return func(c *Client) {
		c.rateLimiter = newRateLimiter(requestsPerMinute)
	}
}

// do executes an authenticated API request.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("allegro: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("allegro: create request: %w", err)
	}

	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("allegro: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			// If we can't decode the error body, return a generic error.
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("allegro: decode response: %w", err)
		}
	}

	return nil
}
