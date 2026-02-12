package ksef

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is a KSeF API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	env        Environment

	Session *SessionService
	Invoice *InvoiceService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithBaseURL overrides the default base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(cl *Client) {
		cl.baseURL = url
	}
}

// NewClient creates a new KSeF API client for the specified environment.
func NewClient(env Environment, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    env.BaseURL(),
		env:        env,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Session = &SessionService{client: c}
	c.Invoice = &InvoiceService{client: c}

	return c
}

// doJSON performs a JSON API request and decodes the response.
func (c *Client) doJSON(ctx context.Context, method, path string, body any, result any, headers map[string]string) error {
	raw, err := c.doRaw(ctx, method, path, body, headers)
	if err != nil {
		return err
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("ksef: failed to decode response: %w (body: %s)", err, truncate(raw, 500))
		}
	}

	return nil
}

// doRawBody performs an API request with a raw io.Reader body and returns the raw response.
func (c *Client) doRawBody(ctx context.Context, method, path string, body io.Reader, contentType string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("ksef: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ksef: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ksef: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
		}
		if apiErr.Message == "" {
			apiErr.Message = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(respBody, 200))
		}
		return nil, apiErr
	}

	return respBody, nil
}

// doRaw performs an API request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any, headers map[string]string) ([]byte, error) {
	var reqBody io.Reader
	contentType := "application/json"

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ksef: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	return c.doRawBody(ctx, method, path, reqBody, contentType, headers)
}

// truncate truncates a byte slice for error messages.
func truncate(data []byte, maxLen int) string {
	s := string(data)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
