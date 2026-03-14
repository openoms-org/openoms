package btp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://ext.btp.pro"
)

// Client is the BTP.pro API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string

	// CSRF token state
	csrfMu    sync.Mutex
	csrfToken string
	csrfTime  time.Time

	Inventory  *InventoryService
	Catalogue  *CatalogueService
	Orders     *OrderService
	Invoices   *InvoiceService
	Waybills   *WaybillService
	ClientInfo *ClientInfoService
	Files      *FileService
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL overrides the default BTP API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new BTP API client with HTTP Basic Auth credentials.
func NewClient(username, password string, opts ...Option) *Client {
	jar, _ := cookiejar.New(nil)
	c := &Client{
		httpClient: &http.Client{Jar: jar},
		baseURL:    defaultBaseURL,
		username:   username,
		password:   password,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Inventory = &InventoryService{client: c}
	c.Catalogue = &CatalogueService{client: c}
	c.Orders = &OrderService{client: c}
	c.Invoices = &InvoiceService{client: c}
	c.Waybills = &WaybillService{client: c}
	c.ClientInfo = &ClientInfoService{client: c}
	c.Files = &FileService{client: c}
	return c
}

// HealthCheck verifies that the BTP API service is running.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.doRaw(ctx, http.MethodGet, "/Gateway/ClientApi/HealthCheck", nil)
	if err != nil {
		return fmt.Errorf("btp: health check: %w", err)
	}
	return nil
}

// do sends an HTTP request and decodes the JSON response into result.
func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}
	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("btp: decode response: %w", err)
		}
	}
	return nil
}

// csrfTTL is how long a fetched CSRF token is considered valid.
const csrfTTL = 10 * time.Minute

// needsCSRF returns true for methods that typically require a CSRF token.
func needsCSRF(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}

// fetchCSRFToken obtains a fresh CSRF token via a GET request with the X-CSRF-TOKEN: Fetch header.
func (c *Client) fetchCSRFToken(ctx context.Context) (string, error) {
	c.csrfMu.Lock()
	defer c.csrfMu.Unlock()

	if c.csrfToken != "" && time.Since(c.csrfTime) < csrfTTL {
		return c.csrfToken, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/Gateway/ClientApi/HealthCheck", nil)
	if err != nil {
		return "", fmt.Errorf("btp: csrf fetch request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("X-CSRF-TOKEN", "Fetch")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("btp: csrf fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	token := resp.Header.Get("X-CSRF-TOKEN")
	if token == "" {
		token = resp.Header.Get("X-Csrf-Token")
	}
	if token == "" {
		return "", nil // API might not require CSRF after all
	}

	c.csrfToken = token
	c.csrfTime = time.Now()
	return token, nil
}

// invalidateCSRF clears the cached CSRF token so the next request fetches a fresh one.
func (c *Client) invalidateCSRF() {
	c.csrfMu.Lock()
	c.csrfToken = ""
	c.csrfMu.Unlock()
}

// doRaw sends an HTTP request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	raw, err := c.doRawOnce(ctx, method, path, body)
	if err != nil {
		// Retry once on CSRF error — token may have expired
		if apiErr, ok := err.(*APIError); ok && isCSRFError(apiErr) {
			c.invalidateCSRF()
			return c.doRawOnce(ctx, method, path, body)
		}
	}
	return raw, err
}

func isCSRFError(err *APIError) bool {
	if err.StatusCode == 403 || err.StatusCode == 419 {
		return true
	}
	if strings.Contains(strings.ToLower(err.RawBody), "csrf") {
		return true
	}
	return false
}

func (c *Client) doRawOnce(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("btp: encode request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("btp: create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if needsCSRF(method) {
		token, err := c.fetchCSRFToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("btp: csrf: %w", err)
		}
		if token != "" {
			req.Header.Set("X-CSRF-TOKEN", token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("btp: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("btp: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, RawBody: string(raw)}
		var result APIResult
		if json.Unmarshal(raw, &result) == nil && result.ResultType != "" {
			apiErr.Result = &result
		}
		return nil, apiErr
	}

	return raw, nil
}

// doMultipart sends a multipart form-data request and decodes the JSON response.
func (c *Client) doMultipart(ctx context.Context, path string, bodyBuf *bytes.Buffer, contentType string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyBuf)
	if err != nil {
		return fmt.Errorf("btp: create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)

	token, err := c.fetchCSRFToken(ctx)
	if err != nil {
		return fmt.Errorf("btp: csrf: %w", err)
	}
	if token != "" {
		req.Header.Set("X-CSRF-TOKEN", token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("btp: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("btp: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, RawBody: string(raw)}
		var apiResult APIResult
		if json.Unmarshal(raw, &apiResult) == nil && apiResult.ResultType != "" {
			apiErr.Result = &apiResult
		}
		return apiErr
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("btp: decode response: %w", err)
		}
	}
	return nil
}
