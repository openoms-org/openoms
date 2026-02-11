package olx

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
	productionBaseURL = "https://www.olx.pl/api/partner"
	productionAuthURL = "https://www.olx.pl/api/open/oauth/token"
)

// Client is the OLX Partner API client.
// Authentication uses OAuth2 client credentials with an access token.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	authURL      string
	clientID     string
	clientSecret string
	accessToken  string

	tokenExpiresAt time.Time
	tokenMu        sync.Mutex

	Adverts      *AdvertService
	Transactions *TransactionService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL overrides the full API base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(u, "/")
		c.authURL = strings.TrimRight(u, "/") + "/oauth/token"
	}
}

// WithAccessToken sets a pre-existing access token (useful for testing, bypasses OAuth refresh).
func WithAccessToken(token string) Option {
	return func(c *Client) {
		c.accessToken = token
		c.tokenExpiresAt = time.Now().Add(24 * time.Hour)
	}
}

// NewClient creates a new OLX Partner API client.
func NewClient(clientID, clientSecret, accessToken string, opts ...Option) *Client {
	c := &Client{
		httpClient:   http.DefaultClient,
		baseURL:      productionBaseURL,
		authURL:      productionAuthURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		accessToken:  accessToken,
	}

	if accessToken != "" {
		c.tokenExpiresAt = time.Now().Add(1 * time.Hour)
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Adverts = &AdvertService{client: c}
	c.Transactions = &TransactionService{client: c}

	return c
}

// tokenResponse represents the OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ensureAccessToken refreshes the OAuth2 access token if it is expired or missing.
func (c *Client) ensureAccessToken(ctx context.Context) error {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt.Add(-30*time.Second)) {
		return nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("olx: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("olx: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("olx: token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("olx: decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// do executes an authenticated API request.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if err := c.ensureAccessToken(ctx); err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("olx: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("olx: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Version", "2.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("olx: execute request: %w", err)
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
			return fmt.Errorf("olx: decode response: %w", err)
		}
	}

	return nil
}

// APIError represents an error response from the OLX API.
type APIError struct {
	StatusCode int    `json:"-"`
	ErrorType  string `json:"error"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "olx: HTTP %d", e.StatusCode)
	if e.ErrorType != "" {
		fmt.Fprintf(&b, " [%s]", e.ErrorType)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	return b.String()
}

// Unwrap returns a sentinel error based on the HTTP status code.
func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 401:
		return ErrUnauthorized
	case e.StatusCode == 403:
		return ErrForbidden
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 429:
		return ErrRateLimited
	case e.StatusCode >= 500:
		return ErrServerError
	default:
		return nil
	}
}
