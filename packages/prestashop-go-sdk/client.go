package prestashop

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
	// DefaultMaxResponseBytes is the maximum JSON response size accepted by default.
	DefaultMaxResponseBytes int64 = 10 * 1024 * 1024
)

// Client is the PrestaShop Web Service API client.
// Authentication uses API key via HTTP Basic Auth (key as username, no password).
type Client struct {
	httpClient       *http.Client
	baseURL          string
	apiKey           string
	maxResponseBytes int64

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
		httpClient:       http.DefaultClient,
		baseURL:          shopURL + "/api",
		apiKey:           apiKey,
		maxResponseBytes: DefaultMaxResponseBytes,
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

// WithMaxResponseBytes sets the maximum JSON response size accepted by the client.
func WithMaxResponseBytes(maxBytes int64) Option {
	return func(c *Client) {
		if maxBytes > 0 {
			c.maxResponseBytes = maxBytes
		}
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
			return fmt.Errorf("prestashop: decode response: %w", err)
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
