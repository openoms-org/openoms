package allegro

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
	defaultBaseURL     = "https://api.allegro.pl"
	defaultUploadURL   = "https://upload.allegro.pl"
	defaultAuthURL     = "https://allegro.pl/auth/oauth"
	sandboxBaseURL     = "https://api.allegro.pl.allegrosandbox.pl"
	sandboxUploadURL   = "https://upload.allegro.pl.allegrosandbox.pl"
	sandboxAuthURL     = "https://allegro.pl.allegrosandbox.pl/auth/oauth"
	defaultRateLimit   = 150
	defaultRedirectURI = "https://localhost/callback"
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
	accessToken    string
	refreshToken   string
	tokenExpiry    time.Time
	onTokenRefresh func(accessToken, refreshToken string, expiry time.Time)
	rateLimiter    *rateLimiter

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
func WithOnTokenRefresh(fn func(string, string, time.Time)) Option {
	return func(c *Client) {
		c.onTokenRefresh = fn
	}
}

// WithRateLimit sets the rate limiter to the given requests per minute.
func WithRateLimit(requestsPerMinute int) Option {
	return func(c *Client) {
		c.rateLimiter = newRateLimiter(requestsPerMinute)
	}
}

// doRaw executes an authenticated API request and returns the raw response body bytes.
// This is used for endpoints that return binary data such as PDF labels or protocols.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	c.ensureValidToken(ctx)

	data, err := c.doRawOnce(ctx, method, path, body)
	if err == nil {
		return data, nil
	}

	// On 401, try to refresh the token and retry once
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 401 && c.refreshToken != "" {
		if _, refreshErr := c.RefreshAccessToken(ctx); refreshErr == nil {
			return c.doRawOnce(ctx, method, path, body)
		}
	}

	return nil, err
}

// doRawOnce executes a single raw API request without retry.
func (c *Client) doRawOnce(ctx context.Context, method, path string, body any) ([]byte, error) {
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

	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	req.Header.Set("Accept", "application/pdf")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("allegro: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return nil, apiErr
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("allegro: read response body: %w", err)
	}
	return data, nil
}

// ensureValidToken proactively refreshes the access token if it is expired or about to expire.
func (c *Client) ensureValidToken(ctx context.Context) {
	if c.refreshToken == "" {
		return
	}
	// Refresh if token expires within 60 seconds
	if !c.tokenExpiry.IsZero() && time.Until(c.tokenExpiry) < 60*time.Second {
		_, _ = c.RefreshAccessToken(ctx)
	}
}

// do executes an authenticated API request with automatic token refresh on 401.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	c.ensureValidToken(ctx)

	err := c.doOnce(ctx, method, path, body, result)
	if err == nil {
		return nil
	}

	// On 401, try to refresh the token and retry once
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 401 && c.refreshToken != "" {
		if _, refreshErr := c.RefreshAccessToken(ctx); refreshErr == nil {
			return c.doOnce(ctx, method, path, body, result)
		}
	}

	return err
}

// doUpload executes an authenticated request against the upload host (upload.allegro.pl).
func (c *Client) doUpload(ctx context.Context, path string, body any, result any) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	c.ensureValidToken(ctx)

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("allegro: marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.uploadURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("allegro: create upload request: %w", err)
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("allegro: execute upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(rawBody, apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
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
	c.ensureValidToken(ctx)

	req, err := http.NewRequestWithContext(ctx, "POST", c.uploadURL+path, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("allegro: create upload request: %w", err)
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("allegro: execute upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{StatusCode: resp.StatusCode, RawBody: string(rawBody)}
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

	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	req.Header.Set("Accept", "application/vnd.allegro.public.v1+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.allegro.public.v1+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("allegro: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rawBody, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{StatusCode: resp.StatusCode, RawBody: string(rawBody)}
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
