package fedex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	productionBaseURL = "https://apis.fedex.com"
	sandboxBaseURL    = "https://apis-sandbox.fedex.com"
)

// Client is a FedEx REST API client with OAuth2 token management.
type Client struct {
	httpClient    *http.Client
	baseURL       string
	clientID      string
	clientSecret  string
	accountNumber string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time

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

// WithSandbox configures the client to use the FedEx sandbox environment.
func WithSandbox() Option {
	return func(cl *Client) {
		cl.baseURL = sandboxBaseURL
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(cl *Client) {
		cl.baseURL = u
	}
}

// NewClient creates a new FedEx API client.
func NewClient(clientID, clientSecret, accountNumber string, opts ...Option) *Client {
	c := &Client{
		httpClient:    http.DefaultClient,
		baseURL:       productionBaseURL,
		clientID:      clientID,
		clientSecret:  clientSecret,
		accountNumber: accountNumber,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Shipments = &ShipmentService{client: c}

	return c
}

// AccountNumber returns the configured FedEx account number.
func (c *Client) AccountNumber() string {
	return c.accountNumber
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// authenticate obtains or refreshes the OAuth2 access token.
func (c *Client) authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("fedex: failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fedex: auth request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fedex: failed to read auth response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("fedex: authentication failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return fmt.Errorf("fedex: failed to decode auth response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
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
			return fmt.Errorf("fedex: failed to decode response: %w", err)
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
			return nil, fmt.Errorf("fedex: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("fedex: failed to create request: %w", err)
	}

	c.mu.Lock()
	token := c.accessToken
	c.mu.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-locale", "en_US")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fedex: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fedex: failed to read response: %w", err)
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

// APIError represents an error response from the FedEx API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("fedex: api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("fedex: api error %d", e.StatusCode)
}
