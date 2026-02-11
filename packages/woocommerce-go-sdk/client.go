package woocommerce

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is the WooCommerce REST API v3 client.
// Authentication uses Basic Auth with consumer key and consumer secret over HTTPS.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	consumerKey    string
	consumerSecret string

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
		httpClient:     http.DefaultClient,
		baseURL:        storeURL + "/wp-json/wc/v3",
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
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
			return fmt.Errorf("woocommerce: decode response: %w", err)
		}
	}

	return nil
}
