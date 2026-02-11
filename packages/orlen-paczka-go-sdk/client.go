package orlenpaczka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	productionBaseURL = "https://api.orlenpaczka.pl"
	sandboxBaseURL    = "https://api-sandbox.orlenpaczka.pl"
)

// Client is an Orlen Paczka API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	partnerID  string

	Shipments *ShipmentService
	Points    *PointService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithSandbox configures the client to use the Orlen Paczka sandbox environment.
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

// NewClient creates a new Orlen Paczka API client.
func NewClient(apiKey, partnerID string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    productionBaseURL,
		apiKey:     apiKey,
		partnerID:  partnerID,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Shipments = &ShipmentService{client: c}
	c.Points = &PointService{client: c}

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
			return fmt.Errorf("orlenpaczka: failed to decode response: %w", err)
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
			return nil, fmt.Errorf("orlenpaczka: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("orlenpaczka: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Partner-ID", c.partnerID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("orlenpaczka: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("orlenpaczka: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
		}
		return nil, apiErr
	}

	return respBody, nil
}

// APIError represents an error response from the Orlen Paczka API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("orlenpaczka: api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("orlenpaczka: api error %d", e.StatusCode)
}
