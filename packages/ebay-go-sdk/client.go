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
	productionAPIURL     = "https://api.ebay.com"
	sandboxAPIURL        = "https://api.sandbox.ebay.com"
	productionAuthURL    = "https://api.ebay.com/identity/v1/oauth2/token"
	sandboxAuthURL       = "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
	productionConsentURL = "https://auth.ebay.com/oauth2/authorize"
	sandboxConsentURL    = "https://auth.sandbox.ebay.com/oauth2/authorize"
)

// Client is the eBay RESTful API client.
// Authentication uses OAuth2 with application credentials and a refresh token.
type Client struct {
	httpClient   *http.Client
	apiURL       string
	authURL      string
	consentURL   string
	appID        string
	certID       string
	devID        string
	refreshToken string
	redirectURI  string

	accessToken    string
	tokenExpiresAt time.Time
	tokenMu        sync.Mutex

	Orders      *OrderService
	Inventory   *InventoryService
	Fulfillment *FulfillmentService
	Offers      *OfferService
	Account     *AccountService
}

// Option configures a Client.
type Option func(*Client)

// WithSandbox configures the client to use the eBay sandbox environment.
func WithSandbox() Option {
	return func(c *Client) {
		c.apiURL = sandboxAPIURL
		c.authURL = sandboxAuthURL
		c.consentURL = sandboxConsentURL
	}
}

// WithRedirectURI sets the OAuth2 redirect URI for the authorization code flow.
func WithRedirectURI(uri string) Option {
	return func(c *Client) {
		c.redirectURI = uri
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
		consentURL:   productionConsentURL,
		appID:        appID,
		certID:       certID,
		devID:        devID,
		refreshToken: refreshToken,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}
	c.Inventory = &InventoryService{client: c}
	c.Fulfillment = &FulfillmentService{client: c}
	c.Offers = &OfferService{client: c}
	c.Account = &AccountService{client: c}

	return c
}

// AuthorizationURL builds the eBay OAuth2 consent URL for user authorization.
func (c *Client) AuthorizationURL(state string) string {
	v := url.Values{}
	v.Set("client_id", c.appID)
	v.Set("response_type", "code")
	v.Set("redirect_uri", c.redirectURI)
	v.Set("scope", "https://api.ebay.com/oauth/api_scope/sell.fulfillment https://api.ebay.com/oauth/api_scope/sell.inventory https://api.ebay.com/oauth/api_scope/sell.account")
	v.Set("state", state)
	return c.consentURL + "?" + v.Encode()
}

// ExchangeCodeResponse holds the tokens returned by the OAuth2 token exchange.
type ExchangeCodeResponse struct {
	AccessToken           string `json:"access_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
}

// ExchangeCode exchanges an authorization code for access + refresh tokens.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*ExchangeCodeResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("ebay: create exchange request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.appID + ":" + c.certID))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ebay: exchange code request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		return nil, fmt.Errorf("ebay: exchange code failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result ExchangeCodeResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("ebay: decode exchange response: %w", err)
	}

	return &result, nil
}

// TokenResponse represents the OAuth2 token endpoint response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// tokenResponse is an alias kept for internal use.
type tokenResponse = TokenResponse

// RefreshAccessToken forces a token refresh and returns the new token response.
// Used by the OAuth refresher worker to proactively verify the refresh token is valid.
func (c *Client) RefreshAccessToken(ctx context.Context) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.refreshToken)
	data.Set("scope", "https://api.ebay.com/oauth/api_scope/sell.fulfillment https://api.ebay.com/oauth/api_scope/sell.inventory")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("ebay: create token request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.appID + ":" + c.certID))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ebay: token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		return nil, fmt.Errorf("ebay: token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("ebay: decode token response: %w", err)
	}

	c.tokenMu.Lock()
	c.accessToken = tokenResp.AccessToken
	c.tokenExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.tokenMu.Unlock()

	return &tokenResp, nil
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
	data.Set("scope", "https://api.ebay.com/oauth/api_scope/sell.fulfillment https://api.ebay.com/oauth/api_scope/sell.inventory")

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		return fmt.Errorf("ebay: token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&tokenResp); err != nil {
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(result); err != nil {
			return fmt.Errorf("ebay: decode response: %w", err)
		}
	}

	return nil
}

// APIError represents an error response from the eBay API.
type APIError struct {
	StatusCode int     `json:"-"`
	Errors     []EbErr `json:"errors"`
	Message    string  `json:"-"`
}

// EbErr represents a single error in an eBay error response.
type EbErr struct {
	ErrorID  int    `json:"errorId"`
	Domain   string `json:"domain"`
	Category string `json:"category"`
	Message  string `json:"message"`
	LongMsg  string `json:"longMessage"`
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
