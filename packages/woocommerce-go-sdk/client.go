package woocommerce

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// DefaultMaxResponseBytes is the maximum JSON response size accepted by default.
	DefaultMaxResponseBytes int64 = 10 * 1024 * 1024
)

// Client is the WooCommerce REST API v3 client.
// Authentication uses Basic Auth with consumer key and consumer secret over HTTPS.
type Client struct {
	httpClient       *http.Client
	baseURL          string
	consumerKey      string
	consumerSecret   string
	maxResponseBytes int64

	Orders   *OrderService
	Products *ProductService
	Webhooks *WebhookService
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new WooCommerce API client.
// storeURL should be the base URL of the WooCommerce store (e.g. "https://example.com").
func NewClient(storeURL, consumerKey, consumerSecret string, opts ...Option) *Client {
	// Normalize: strip trailing slash
	storeURL = strings.TrimRight(storeURL, "/")

	c := &Client{
		httpClient:       http.DefaultClient,
		baseURL:          storeURL + "/wp-json/wc/v3",
		consumerKey:      consumerKey,
		consumerSecret:   consumerSecret,
		maxResponseBytes: DefaultMaxResponseBytes,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Products = &ProductService{client: c}
	c.Webhooks = &WebhookService{client: c}

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

// WithMaxResponseBytes sets the maximum JSON response size accepted by the client.
func WithMaxResponseBytes(maxBytes int64) Option {
	return func(c *Client) {
		if maxBytes > 0 {
			c.maxResponseBytes = maxBytes
		}
	}
}

// do executes an authenticated API request using Basic Auth.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("woocommerce: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("woocommerce: create request: %w", err)
	}

	// Basic Auth: base64(consumer_key:consumer_secret)
	auth := base64.StdEncoding.EncodeToString([]byte(c.consumerKey + ":" + c.consumerSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("woocommerce: execute request: %w", err)
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
			return fmt.Errorf("woocommerce: decode response: %w", err)
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
