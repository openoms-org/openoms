package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultAPIVersion = "2025-01"

// Client is the Shopify Admin REST API client.
// Authentication uses a private app access token via X-Shopify-Access-Token header.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string

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
		shopDomain = shopDomain + ".myshopify.com"
	}
	shopDomain = strings.TrimRight(shopDomain, "/")
	if !strings.HasPrefix(shopDomain, "https://") {
		shopDomain = "https://" + shopDomain
	}

	c := &Client{
		httpClient:  http.DefaultClient,
		baseURL:     shopDomain + "/admin/api/" + defaultAPIVersion,
		accessToken: accessToken,
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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		respBody, _ := io.ReadAll(resp.Body)
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
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("shopify: decode response: %w", err)
		}
	}

	return nil
}
