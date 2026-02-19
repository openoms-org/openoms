package btp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	c := &Client{
		httpClient: http.DefaultClient,
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

// doRaw sends an HTTP request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("btp: send request: %w", err)
	}
	defer resp.Body.Close()

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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("btp: send request: %w", err)
	}
	defer resp.Body.Close()

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
