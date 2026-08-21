package allegro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	defaultBaseURL     = "https://api.allegro.pl"
	defaultUploadURL   = "https://upload.allegro.pl"
	defaultAuthURL     = "https://allegro.pl/auth/oauth"
	sandboxBaseURL     = "https://api.allegro.pl.allegrosandbox.pl"
	sandboxUploadURL   = "https://upload.allegro.pl.allegrosandbox.pl"
	sandboxAuthURL     = "https://allegro.pl.allegrosandbox.pl/auth/oauth"
	defaultRateLimit   = 150
	defaultRedirectURI = "https://localhost/callback"
	maxRetries         = 3
	// maxResponseBody caps response body reads to prevent memory exhaustion (50 MB).
	maxResponseBody = 50 << 20
	// maxErrorBody caps error response body reads (1 MB).
	maxErrorBody = 1 << 20
)

// Client is the Allegro API client.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	uploadURL      string
	authBaseURL    string
	clientID       string
	clientSecret   string
	redirectURI    string
	mu             sync.Mutex // protects token fields below
	accessToken    string
	refreshToken   string
	tokenExpiry    time.Time
	onTokenRefresh func(accessToken, refreshToken string, expiry time.Time) error
	rateLimiter    *rateLimiter
	retryDelay     func(attempt int) time.Duration

	Orders             *OrderService
	Events             *EventService
	Offers             *OfferService
	Fulfillment        *FulfillmentService
	ShipmentManagement *ShipmentManagementService
	Account            *AccountService
	Messages           *MessageService
	Returns            *ReturnService
	Payments           *PaymentService
	Categories         *CategoryService
	ProductCatalog     *ProductCatalogService
	Pricing            *PricingService
	Disputes           *DisputeService
	Ratings            *RatingService
	Promotions         *PromotionService
	DeliverySettings   *DeliverySettingsService
	AfterSales         *AfterSalesService
	SizeTables         *SizeTableService
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new Allegro API client.
func NewClient(clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient:   http.DefaultClient,
		baseURL:      defaultBaseURL,
		uploadURL:    defaultUploadURL,
		authBaseURL:  defaultAuthURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  defaultRedirectURI,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.rateLimiter == nil {
		c.rateLimiter = newRateLimiter(defaultRateLimit)
	}
	if c.retryDelay == nil {
		c.retryDelay = defaultRetryDelay
	}

	c.Orders = &OrderService{client: c}
	c.Events = &EventService{client: c}
	c.Offers = &OfferService{client: c}
	c.Fulfillment = &FulfillmentService{client: c}
	c.ShipmentManagement = &ShipmentManagementService{client: c}
	c.Account = &AccountService{client: c}
	c.Messages = &MessageService{client: c}
	c.Returns = &ReturnService{client: c}
	c.Payments = &PaymentService{client: c}
	c.Categories = &CategoryService{client: c}
	c.ProductCatalog = &ProductCatalogService{client: c}
	c.Pricing = &PricingService{client: c}
	c.Disputes = &DisputeService{client: c}
	c.Ratings = &RatingService{client: c}
	c.Promotions = &PromotionService{client: c}
	c.DeliverySettings = &DeliverySettingsService{client: c}
	c.AfterSales = &AfterSalesService{client: c}
	c.SizeTables = &SizeTableService{client: c}

	return c
}

// Close releases resources held by the client (e.g. rate limiter goroutine).
func (c *Client) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Close()
	}
}

// getAccessToken returns the current access token (thread-safe).
func (c *Client) getAccessToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.accessToken
}

// getRefreshToken returns the current refresh token (thread-safe).
func (c *Client) getRefreshToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refreshToken
}

// retryableStatusCode returns true for status codes that should be retried.
func retryableStatusCode(code int) bool {
	return code == 429 || code == 502 || code == 503 || code == 504
}

// defaultRetryDelay returns the backoff delay for a given retry attempt.
// Delays: 500ms, 1s, 2s (exponential: 500ms << attempt).
func defaultRetryDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	return time.Duration(500<<min(attempt, 30)) * time.Millisecond
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithSandbox configures the client for the Allegro sandbox environment.
func WithSandbox() Option {
	return func(c *Client) {
		c.baseURL = sandboxBaseURL
		c.uploadURL = sandboxUploadURL
		c.authBaseURL = sandboxAuthURL
	}
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithRedirectURI sets the OAuth redirect URI.
func WithRedirectURI(uri string) Option {
	return func(c *Client) {
		c.redirectURI = uri
	}
}

// WithTokens sets initial OAuth tokens.
func WithTokens(accessToken, refreshToken string, expiry time.Time) Option {
	return func(c *Client) {
		c.accessToken = accessToken
		c.refreshToken = refreshToken
		c.tokenExpiry = expiry
	}
}

// WithOnTokenRefresh registers a callback invoked when tokens are refreshed.
// A non-nil error fails the refresh so callers do not continue with unpersisted tokens.
func WithOnTokenRefresh(fn func(string, string, time.Time) error) Option {
	return func(c *Client) {
		c.onTokenRefresh = fn
	}
}

// SetOnTokenRefresh replaces the token-refresh persist callback after construction.
func (c *Client) SetOnTokenRefresh(fn func(string, string, time.Time) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onTokenRefresh = fn
}

// WithAuthBaseURL overrides the OAuth token endpoint base URL.
func WithAuthBaseURL(url string) Option {
	return func(c *Client) {
		c.authBaseURL = url
	}
}

// WithRateLimit sets the rate limiter to the given requests per minute.
func WithRateLimit(requestsPerMinute int) Option {
	return func(c *Client) {
		c.rateLimiter = newRateLimiter(requestsPerMinute)
	}
}

// doRaw executes an authenticated API request and returns the raw response body bytes.
// acceptType specifies the Accept header (e.g. "application/pdf").
// Retries automatically on 401 (token refresh) and 429/5xx (exponential backoff).
func (c *Client) doRaw(ctx context.Context, method, path string, body any, acceptType string) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		data, err := c.doRawOnce(ctx, method, path, body, acceptType)
		if err == nil {
			return data, nil
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			return nil, err
		}

		// On 401, try to refresh the token and retry once (only on first attempt)
		if apiErr.StatusCode == 401 && attempt == 0 && c.getRefreshToken() != "" {
			if _, refreshErr := c.RefreshAccessToken(ctx); refreshErr != nil {
				return nil, fmt.Errorf("allegro: refresh access token after unauthorized: %w", refreshErr)
			}
			lastErr = err
			continue
		}

		// On retryable status codes (429, 502, 503, 504), backoff and retry
		if retryableStatusCode(apiErr.StatusCode) && attempt < maxRetries {
			delay := c.retryDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			lastErr = err
			continue
		}

		return nil, err
	}

	return nil, lastErr
}

// doRawOnce executes a single raw API request without retry.
func (c *Client) doRawOnce(ctx context.Context, method, path string, body any, acceptType string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("allegro: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("allegro: create request: %w", err)
	}

	if token := c.getAccessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", acceptType)
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("allegro: execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxErrorBody)).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return nil, apiErr
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("allegro: read response body: %w", err)
	}
	return data, nil
}

// ensureValidToken proactively refreshes the access token if it is expired or about to expire.
func (c *Client) ensureValidToken(ctx context.Context) error {
	c.mu.Lock()
	needsRefresh := c.refreshToken != "" && !c.tokenExpiry.IsZero() && time.Until(c.tokenExpiry) < 60*time.Second
	c.mu.Unlock()
	if !needsRefresh {
		return nil
	}
	if _, err := c.RefreshAccessToken(ctx); err != nil {
		return fmt.Errorf("allegro: proactive token refresh failed: %w", err)
	}
	return nil
}

// do executes an authenticated API request with automatic token refresh on 401
// and retry with exponential backoff on 429/5xx.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	if err := c.ensureValidToken(ctx); err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.doOnce(ctx, method, path, body, result)
		if err == nil {
			return nil
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			return err
		}

		// On 401, try to refresh the token and retry once (only on first attempt)
		if apiErr.StatusCode == 401 && attempt == 0 && c.getRefreshToken() != "" {
			if _, refreshErr := c.RefreshAccessToken(ctx); refreshErr != nil {
				return fmt.Errorf("allegro: refresh access token after unauthorized: %w", refreshErr)
			}
			lastErr = err
			continue
		}

		// On retryable status codes (429, 502, 503, 504), backoff and retry
		if retryableStatusCode(apiErr.StatusCode) && attempt < maxRetries {
			delay := c.retryDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			lastErr = err
			continue
		}

		return err
	}

	return lastErr
}

// doUpload executes an authenticated request against the upload host (upload.allegro.pl).
func (c *Client) doUpload(ctx context.Context, path string, body any, result any) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	if err := c.ensureValidToken(ctx); err != nil {
		return err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("allegro: marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.uploadURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("allegro: create upload request: %w", err)
	}
	if token := c.getAccessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("allegro: execute upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(rawBody, apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(result); err != nil {
			return fmt.Errorf("allegro: decode upload response: %w", err)
		}
	}
	return nil
}

// doUploadBinary uploads raw binary data (e.g. image bytes) to the upload host.
// Returns the Location URL of the uploaded resource.
func (c *Client) doUploadBinary(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}
	if err := c.ensureValidToken(ctx); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.uploadURL+path, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("allegro: create upload request: %w", err)
	}
	if token := c.getAccessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("allegro: execute upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(rawBody, apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return "", apiErr
	}

	// The response contains the location of the uploaded image
	var result struct {
		Location string `json:"location"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("allegro: decode upload response: %w", err)
	}
	return result.Location, nil
}

// doOnce executes a single authenticated API request without retry.
func (c *Client) doOnce(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("allegro: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("allegro: create request: %w", err)
	}

	if token := c.getAccessToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("allegro: execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(rawBody, apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("allegro: decode response: %w", err)
		}
	}

	return nil
}
