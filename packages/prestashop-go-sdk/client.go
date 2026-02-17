package prestashop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is the PrestaShop Web Service API client.
// Authentication uses API key via HTTP Basic Auth (key as username, no password).
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string

	Orders   *OrderService
	Products *ProductService
	Stock    *StockService
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new PrestaShop API client.
// shopURL should be the base URL of the PrestaShop store (e.g. "https://myshop.example.com").
func NewClient(shopURL, apiKey string, opts ...Option) *Client {
	shopURL = strings.TrimRight(shopURL, "/")

	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    shopURL + "/api",
		apiKey:     apiKey,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Products = &ProductService{client: c}
	c.Stock = &StockService{client: c}

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

// do executes an authenticated API request using Basic Auth with API key.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("prestashop: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// PrestaShop API always uses output_format=JSON
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	fullURL := c.baseURL + path + sep + "output_format=JSON"

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("prestashop: create request: %w", err)
	}

	// PrestaShop uses API key as Basic Auth username with empty password
	req.SetBasicAuth(c.apiKey, "")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("prestashop: execute request: %w", err)
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
			return fmt.Errorf("prestashop: decode response: %w", err)
		}
	}

	return nil
}
