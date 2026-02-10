package dpd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const (
	productionBaseURL = "https://dpd.com.pl/api/v1"
	sandboxBaseURL    = "https://dpd-sandbox.com.pl/api/v1"
)

// Client is a DPD Poland API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	login      string
	password   string
	masterFid  string

	mu           sync.Mutex
	sessionToken string

	Shipments *ShipmentService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithSandbox configures the client to use the DPD sandbox environment.
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

// NewClient creates a new DPD API client.
func NewClient(login, password, masterFid string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    productionBaseURL,
		login:      login,
		password:   password,
		masterFid:  masterFid,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Shipments = &ShipmentService{client: c}

	return c
}

// MasterFid returns the configured DPD master FID.
func (c *Client) MasterFid() string {
	return c.masterFid
}

type authRequest struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	MasterFid string `json:"masterFid"`
}

type authResponse struct {
	Token string `json:"token"`
}

// authenticate obtains a session token from the DPD API.
func (c *Client) authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionToken != "" {
		return nil
	}

	body := authRequest{
		Login:     c.login,
		Password:  c.password,
		MasterFid: c.masterFid,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("dpd: failed to encode auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/login", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("dpd: failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dpd: auth request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("dpd: failed to read auth response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("dpd: authentication failed with status %d", resp.StatusCode)
	}

	var authResp authResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return fmt.Errorf("dpd: failed to decode auth response: %w", err)
	}

	c.sessionToken = authResp.Token
	return nil
}

// do performs a JSON API request and decodes the response into result.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("dpd: failed to decode response: %w", err)
		}
	}

	return nil
}

// doRaw performs an API request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("dpd: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dpd: failed to create request: %w", err)
	}

	c.mu.Lock()
	token := c.sessionToken
	c.mu.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dpd: request failed: %w", err)
	}
	defer resp.Body.Close()

	// If 401, clear token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		c.mu.Lock()
		c.sessionToken = ""
		c.mu.Unlock()

		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}

		return c.doRawAuthenticated(ctx, method, path, body)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dpd: failed to read response: %w", err)
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

// doRawAuthenticated performs an authenticated request without retry logic.
func (c *Client) doRawAuthenticated(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("dpd: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dpd: failed to create request: %w", err)
	}

	c.mu.Lock()
	token := c.sessionToken
	c.mu.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dpd: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dpd: failed to read response: %w", err)
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

// APIError represents an error response from the DPD API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("dpd: api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("dpd: api error %d", e.StatusCode)
}
