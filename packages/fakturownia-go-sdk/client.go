package fakturownia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// Client is a Fakturownia API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string

	Invoices *InvoiceService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(cl *Client) {
		cl.baseURL = url
	}
}

// NewClient creates a new Fakturownia API client.
// subdomain is the tenant subdomain (e.g. "myfirm" for myfirm.fakturownia.pl).
// apiToken is the API token for authentication.
func NewClient(subdomain, apiToken string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    fmt.Sprintf("https://%s.fakturownia.pl", subdomain),
		apiToken:   apiToken,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Invoices = &InvoiceService{client: c}

	return c
}

// InvoiceService handles invoice-related API calls.
type InvoiceService struct {
	client *Client
}

// Create creates a new invoice.
func (s *InvoiceService) Create(ctx context.Context, data InvoiceRequestData) (*InvoiceResponse, error) {
	req := CreateInvoiceRequest{
		APIToken: s.client.apiToken,
		Invoice:  data,
	}

	var result InvoiceResponse
	if err := s.client.do(ctx, http.MethodPost, "/invoices.json", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves an invoice by its ID.
func (s *InvoiceService) Get(ctx context.Context, id int) (*InvoiceResponse, error) {
	path := fmt.Sprintf("/invoices/%d.json?api_token=%s", id, s.client.apiToken)
	var result InvoiceResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPDF downloads the PDF of an invoice by its ID.
func (s *InvoiceService) GetPDF(ctx context.Context, id int) ([]byte, error) {
	path := fmt.Sprintf("/invoices/%d.pdf?api_token=%s", id, s.client.apiToken)
	raw, err := s.client.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Cancel marks an invoice as cancelled.
func (s *InvoiceService) Cancel(ctx context.Context, id int) error {
	path := fmt.Sprintf("/invoices/%d/cancel.json?api_token=%s", id, s.client.apiToken)
	return s.client.do(ctx, http.MethodPut, path, nil, nil)
}

// do performs a JSON API request and decodes the response.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("fakturownia: failed to decode response: %w", err)
		}
	}

	return nil
}

// doRaw performs an API request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("fakturownia: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("fakturownia: failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fakturownia: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fakturownia: failed to read response: %w", err)
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

// ParseExternalID converts a string external ID to an int for Fakturownia API calls.
func ParseExternalID(externalID string) (int, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return 0, fmt.Errorf("fakturownia: invalid external ID %q: %w", externalID, err)
	}
	return id, nil
}
