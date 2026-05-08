package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultAPIVersion = "2025-01"
	// DefaultMaxResponseBytes is the maximum JSON response size accepted by default.
	DefaultMaxResponseBytes int64 = 10 * 1024 * 1024
)

// Client is the Shopify Admin REST API client.
// Authentication uses a private app access token via X-Shopify-Access-Token header.
type Client struct {
	httpClient       *http.Client
	baseURL          string
	accessToken      string
	maxResponseBytes int64

	Orders    *OrderService
	Products  *ProductService
	Inventory *InventoryService
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new Shopify Admin API client.
// shopDomain should be the shop domain (e.g. "myshop" or "myshop.myshopify.com").
func NewClient(shopDomain, accessToken string, opts ...Option) *Client {
	// Normalize shop domain
	if !strings.Contains(shopDomain, ".") {
		shopDomain += ".myshopify.com"
	}
	shopDomain = strings.TrimRight(shopDomain, "/")
	if !strings.HasPrefix(shopDomain, "https://") {
		shopDomain = "https://" + shopDomain
	}

	c := &Client{
		httpClient:       http.DefaultClient,
		baseURL:          shopDomain + "/admin/api/" + defaultAPIVersion,
		accessToken:      accessToken,
		maxResponseBytes: DefaultMaxResponseBytes,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Products = &ProductService{client: c}
	c.Inventory = &InventoryService{client: c}

	return c
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithAPIVersion overrides the API version (default: 2025-01).
func WithAPIVersion(version string) Option {
	return func(c *Client) {
		// Reconstruct baseURL with new version
		parts := strings.Split(c.baseURL, "/admin/api/")
		if len(parts) == 2 {
			c.baseURL = parts[0] + "/admin/api/" + version
		}
	}
}

// WithBaseURL overrides the full API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
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

// do executes an authenticated API request.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("shopify: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("shopify: create request: %w", err)
	}

	req.Header.Set("X-Shopify-Access-Token", c.accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shopify: execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		respBody, err := c.readResponseBody(resp)
		if err != nil {
			return err
		}
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
			if apiErr.Message == "" {
				// Shopify sometimes returns {"errors": "string"} instead of map
				var simple struct {
					Errors string `json:"errors"`
				}
				if json.Unmarshal(respBody, &simple) == nil && simple.Errors != "" {
					apiErr.Message = simple.Errors
				}
			}
		}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := c.decodeResponseJSON(resp, result); err != nil {
			return fmt.Errorf("shopify: decode response: %w", err)
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

func (c *Client) readResponseBody(resp *http.Response) ([]byte, error) {
	if resp.ContentLength > c.maxResponseBytes {
		return nil, fmt.Errorf("%w: content length %d exceeds max %d bytes", ErrResponseTooLarge, resp.ContentLength, c.maxResponseBytes)
	}
	reader := &responseLimitReader{
		r:         resp.Body,
		remaining: c.maxResponseBytes + 1,
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		if errors.Is(err, ErrResponseTooLarge) {
			return nil, fmt.Errorf("%w: max %d bytes", ErrResponseTooLarge, c.maxResponseBytes)
		}
		return nil, err
	}
	return data, nil
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
