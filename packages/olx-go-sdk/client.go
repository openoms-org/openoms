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
	productionBaseURL      = "https://www.olx.pl/api/partner"
	productionAuthURL      = "https://www.olx.pl/api/open/oauth/token"
	productionAuthorizeURL = "https://www.olx.pl/oauth/authorize"
)

const (
	maxResponseBody = 10 << 20 // 10 MB
	maxErrorBody    = 1 << 20  // 1 MB
)

// Client is the OLX Partner API client.
// Supports both client_credentials and authorization_code OAuth2 flows.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	authURL      string
	authorizeURL string
	clientID     string
	clientSecret string
	accessToken  string
	refreshToken string
	redirectURI  string

	tokenExpiresAt time.Time
	tokenMu        sync.Mutex

	onTokenRefresh func(accessToken, refreshToken string, expiry time.Time)

	Adverts      *AdvertService
	Categories   *CategoryService
	Cities       *CityService
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
		c.authorizeURL = strings.TrimRight(u, "/") + "/oauth/authorize"
	}
}

// WithRedirectURI sets the OAuth redirect URI for the authorization code flow.
func WithRedirectURI(uri string) Option {
	return func(c *Client) { c.redirectURI = uri }
}

// WithTokens sets initial OAuth tokens (access token, refresh token, and expiry).
func WithTokens(accessToken, refreshToken string, expiry time.Time) Option {
	return func(c *Client) {
		c.accessToken = accessToken
		c.refreshToken = refreshToken
		c.tokenExpiresAt = expiry
	}
}

// WithOnTokenRefresh registers a callback invoked when tokens are refreshed.
func WithOnTokenRefresh(fn func(string, string, time.Time)) Option {
	return func(c *Client) { c.onTokenRefresh = fn }
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
		authorizeURL: productionAuthorizeURL,
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
	c.Categories = &CategoryService{client: c}
	c.Cities = &CityService{client: c}
	c.Transactions = &TransactionService{client: c}

	return c
}

// ensureAccessToken refreshes the OAuth2 access token if it is expired or missing.
// Prefers refresh_token grant when available (preserves user-level scopes from
// authorization_code flow). Falls back to client_credentials for API-key-only setups.
func (c *Client) ensureAccessToken(ctx context.Context) error {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt.Add(-30*time.Second)) {
		return nil
	}

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	if c.refreshToken != "" {
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", c.refreshToken)
	} else {
		data.Set("grant_type", "client_credentials")
		data.Set("scope", "read write v2 partner")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("olx: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("olx: token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("olx: token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(&tok); err != nil {
		return fmt.Errorf("olx: decode token response: %w", err)
	}

	c.accessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		c.refreshToken = tok.RefreshToken
	}
	c.tokenExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	if c.onTokenRefresh != nil {
		c.onTokenRefresh(c.accessToken, c.refreshToken, c.tokenExpiresAt)
	}

	return nil
}

// do executes an authenticated API request.
// On 401, it invalidates the cached token, refreshes, and retries once.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	return c.doInternal(ctx, method, path, body, result, true)
}

func (c *Client) doInternal(ctx context.Context, method, path string, body any, result any, canRetry bool) error {
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
	defer func() { _ = resp.Body.Close() }()

	// On 401 with a stale token, force refresh and retry once.
	if resp.StatusCode == 401 && canRetry {
		c.tokenMu.Lock()
		c.tokenExpiresAt = time.Time{}
		c.tokenMu.Unlock()
		return c.doInternal(ctx, method, path, body, result, false)
	}

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(rawBody, apiErr); err != nil {
			apiErr.Message = string(rawBody)
			if apiErr.Message == "" {
				apiErr.Message = http.StatusText(resp.StatusCode)
			}
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(result); err != nil {
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
