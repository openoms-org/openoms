package shoper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client is the Shoper WebAPI REST client.
// Authentication uses OAuth2 with client credentials grant.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	clientID       string
	clientSecret   string

	// OAuth2 token state
	mu           sync.Mutex
	accessToken  string
	tokenExpires time.Time

	Orders     *OrderService
	Products   *ProductService
	Categories *CategoryService
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new Shoper API client.
// shopURL should be the base URL of the Shoper store (e.g. "https://myshop.shoper.pl").
func NewClient(shopURL, clientID, clientSecret string, opts ...Option) *Client {
	shopURL = strings.TrimRight(shopURL, "/")

	c := &Client{
		httpClient:   http.DefaultClient,
		baseURL:      shopURL + "/webapi/rest",
		clientID:     clientID,
		clientSecret: clientSecret,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Products = &ProductService{client: c}
	c.Categories = &CategoryService{client: c}

	return c
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL overrides the full API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithAccessToken sets a pre-obtained access token (skips OAuth2 flow).
func WithAccessToken(token string) Option {
	return func(c *Client) {
		c.accessToken = token
		c.tokenExpires = time.Now().Add(24 * time.Hour)
	}
}

// authenticate obtains an OAuth2 access token using client credentials.
func (c *Client) authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return early if token is still valid (with 30s buffer)
	if c.accessToken != "" && time.Now().Add(30*time.Second).Before(c.tokenExpires) {
		return nil
	}

	// Shoper OAuth2 token endpoint
	tokenURL := strings.Replace(c.baseURL, "/webapi/rest", "/webapi/rest/auth", 1)
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return fmt.Errorf("shoper: create auth request: %w", err)
	}
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shoper: auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.NewDecoder(resp.Body).Decode(apiErr)
		return apiErr
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("shoper: decode auth response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpires = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// do executes an authenticated API request.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("shoper: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("shoper: create request: %w", err)
	}

	c.mu.Lock()
	token := c.accessToken
	c.mu.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shoper: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("shoper: decode response: %w", err)
		}
	}

	return nil
}
