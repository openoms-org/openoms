package erli

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
	productionBaseURL = "https://api.erli.pl/v2"
	sandboxBaseURL    = "https://api-sandbox.erli.pl/v2"

	// maxResponseBody caps response body reads to prevent memory exhaustion (10 MB).
	maxResponseBody = 10 << 20
	// maxErrorBody caps error response body reads (1 MB).
	maxErrorBody = 1 << 20
)

// Client is an Erli.pl marketplace API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string

	Orders *OrderService
	Offers *OfferService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithSandbox configures the client to use the Erli sandbox environment.
func WithSandbox() Option {
	return func(cl *Client) {
		cl.baseURL = sandboxBaseURL
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(cl *Client) {
		cl.baseURL = url
	}
}

// NewClient creates a new Erli API client.
func NewClient(apiToken string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    productionBaseURL,
		apiToken:   apiToken,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Offers = &OfferService{client: c}

	return c
}

// do performs a JSON API request and decodes the response into result.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("erli: failed to decode response: %w", err)
		}
	}

	return nil
}

// doRaw performs an API request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("erli: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("erli: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erli: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxErrorBody)).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return nil, apiErr
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("erli: failed to read response: %w", err)
	}

	return respBody, nil
}

// APIError represents an error response from the Erli API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("erli: api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("erli: api error %d", e.StatusCode)
}
