package ebay

import (
	"bytes"
	"context"
	"encoding/base64"
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
	productionAPIURL  = "https://api.ebay.com"
	sandboxAPIURL     = "https://api.sandbox.ebay.com"
	productionAuthURL = "https://api.ebay.com/identity/v1/oauth2/token"
	sandboxAuthURL    = "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
)

// Client is the eBay RESTful API client.
// Authentication uses OAuth2 with application credentials and a refresh token.
type Client struct {
	httpClient   *http.Client
	apiURL       string
	authURL      string
	appID        string
	certID       string
	devID        string
	refreshToken string

	accessToken    string
	tokenExpiresAt time.Time
	tokenMu        sync.Mutex

	Orders *OrderService
}

// Option configures a Client.
type Option func(*Client)

// WithSandbox configures the client to use the eBay sandbox environment.
func WithSandbox() Option {
	return func(c *Client) {
		c.apiURL = sandboxAPIURL
		c.authURL = sandboxAuthURL
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL overrides the full API base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.apiURL = strings.TrimRight(u, "/")
		c.authURL = strings.TrimRight(u, "/") + "/identity/v1/oauth2/token"
	}
}

// WithAccessToken sets a pre-existing access token (useful for testing, bypasses OAuth refresh).
func WithAccessToken(token string) Option {
	return func(c *Client) {
		c.accessToken = token
		c.tokenExpiresAt = time.Now().Add(24 * time.Hour)
	}
}

// NewClient creates a new eBay API client.
func NewClient(appID, certID, devID, refreshToken string, opts ...Option) *Client {
	c := &Client{
		httpClient:   http.DefaultClient,
		apiURL:       productionAPIURL,
		authURL:      productionAuthURL,
		appID:        appID,
		certID:       certID,
		devID:        devID,
		refreshToken: refreshToken,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}

	return c
}

// tokenResponse represents the OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// ensureAccessToken refreshes the OAuth2 access token if it is expired or missing.
func (c *Client) ensureAccessToken(ctx context.Context) error {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Token still valid — skip refresh
	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt.Add(-30*time.Second)) {
		return nil
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.refreshToken)
	data.Set("scope", "https://api.ebay.com/oauth/api_scope/sell.fulfillment")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("ebay: create token request: %w", err)
	}

	// Basic auth with appID:certID
	auth := base64.StdEncoding.EncodeToString([]byte(c.appID + ":" + c.certID))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ebay: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ebay: token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("ebay: decode token response: %w", err)
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
			return fmt.Errorf("ebay: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("ebay: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ebay: execute request: %w", err)
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
			return fmt.Errorf("ebay: decode response: %w", err)
		}
	}

	return nil
}

// APIError represents an error response from the eBay API.
type APIError struct {
	StatusCode int      `json:"-"`
	Errors     []EbErr  `json:"errors"`
	Message    string   `json:"-"`
}

// EbErr represents a single error in an eBay error response.
type EbErr struct {
	ErrorID   int    `json:"errorId"`
	Domain    string `json:"domain"`
	Category  string `json:"category"`
	Message   string `json:"message"`
	LongMsg   string `json:"longMessage"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ebay: HTTP %d", e.StatusCode)
	if len(e.Errors) > 0 {
		fmt.Fprintf(&b, ": %s", e.Errors[0].Message)
	} else if e.Message != "" {
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
