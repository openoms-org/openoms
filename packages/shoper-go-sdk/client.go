package shoper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultMaxResponseBytes is the maximum JSON response size accepted by default.
	DefaultMaxResponseBytes int64 = 10 * 1024 * 1024
)

// Client is the Shoper WebAPI REST client.
// Authentication uses OAuth2 with client credentials grant.
type Client struct {
	httpClient       *http.Client
	baseURL          string
	clientID         string
	clientSecret     string
	maxResponseBytes int64

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
		httpClient:       http.DefaultClient,
		baseURL:          shopURL + "/webapi/rest",
		clientID:         clientID,
		clientSecret:     clientSecret,
		maxResponseBytes: DefaultMaxResponseBytes,
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

// WithMaxResponseBytes sets the maximum JSON response size accepted by the client.
func WithMaxResponseBytes(maxBytes int64) Option {
	return func(c *Client) {
		if maxBytes > 0 {
			c.maxResponseBytes = maxBytes
		}
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := c.decodeResponseJSON(resp, apiErr); err != nil && errors.Is(err, ErrResponseTooLarge) {
			return err
		}
		return apiErr
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := c.decodeResponseJSON(resp, &tokenResp); err != nil {
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := c.decodeResponseJSON(resp, apiErr); err != nil {
			if errors.Is(err, ErrResponseTooLarge) {
				return err
			}
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := c.decodeResponseJSON(resp, result); err != nil {
			return fmt.Errorf("shoper: decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) decodeResponseJSON(resp *http.Response, result any) error {
	if resp.ContentLength > c.maxResponseBytes {
		return fmt.Errorf("%w: content length %d exceeds max %d bytes", ErrResponseTooLarge, resp.ContentLength, c.maxResponseBytes)
	}
	reader := &responseLimitReader{
		r:         resp.Body,
		remaining: c.maxResponseBytes + 1,
	}
	if err := json.NewDecoder(reader).Decode(result); err != nil {
		if errors.Is(err, ErrResponseTooLarge) {
			return fmt.Errorf("%w: max %d bytes", ErrResponseTooLarge, c.maxResponseBytes)
		}
		return err
	}
	return nil
}

type responseLimitReader struct {
	r         io.Reader
	remaining int64
}

func (r *responseLimitReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, ErrResponseTooLarge
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.r.Read(p)
	r.remaining -= int64(n)
	if r.remaining <= 0 && n > 0 {
		return n, ErrResponseTooLarge
	}
	return n, err
}
