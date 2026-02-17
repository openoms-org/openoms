package wfirma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api2.wfirma.pl"
)

// Client is a wFirma API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string

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

// NewClient creates a new wFirma API client.
// apiKey is the API key for authentication via the X-API-KEY header.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
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

// Invoice represents a wFirma invoice.
type Invoice struct {
	ID            int           `json:"id"`
	Type          string        `json:"type"` // vat, proforma, correction
	Number        string        `json:"fullnumber"`
	IssueDate     string        `json:"date"`
	SaleDate      string        `json:"disposaldate"`
	PaymentDate   string        `json:"paymentdate"`
	PaymentMethod string        `json:"paymentmethod"`
	Currency      string        `json:"currency"`
	Notes         string        `json:"description"`
	Status        string        `json:"status"`
	TotalNet      string        `json:"netto"`
	TotalGross    string        `json:"brutto"`
	Buyer         InvoiceBuyer  `json:"contractor_detail"`
	Items         []InvoiceItem `json:"invoicecontents"`
}

// InvoiceBuyer represents buyer information on a wFirma invoice.
type InvoiceBuyer struct {
	Name       string `json:"name"`
	NIP        string `json:"nip"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"zip"`
	Country    string `json:"country"`
}

// InvoiceItem represents a line item on a wFirma invoice.
type InvoiceItem struct {
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Count     int     `json:"count"`
	Price     float64 `json:"price"`
	VATRate   string  `json:"vat"` // "23", "8", "5", "0", "zw", "np"
}

// CreateInvoiceRequest is the request body for creating an invoice.
type CreateInvoiceRequest struct {
	Type          string        `json:"type"`
	IssueDate     string        `json:"date"`
	SaleDate      string        `json:"disposaldate"`
	PaymentDate   string        `json:"paymentdate"`
	PaymentMethod string        `json:"paymentmethod,omitempty"`
	Currency      string        `json:"currency,omitempty"`
	Notes         string        `json:"description,omitempty"`
	Buyer         InvoiceBuyer  `json:"contractor_detail"`
	Items         []InvoiceItem `json:"invoicecontents"`
}

// ListInvoicesParams are query parameters for listing invoices.
type ListInvoicesParams struct {
	Page  int
	Limit int
}

// Create creates a new invoice in wFirma.
func (s *InvoiceService) Create(ctx context.Context, req CreateInvoiceRequest) (*Invoice, error) {
	body := map[string]any{
		"invoices": []map[string]any{
			{
				"invoice": req,
			},
		},
	}

	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodPost, "/invoices/add", body, &resp); err != nil {
		return nil, err
	}

	if len(resp.Invoices) == 0 {
		return nil, fmt.Errorf("wfirma: empty response from create invoice")
	}

	return &resp.Invoices[0], nil
}

// Get retrieves an invoice by its ID.
func (s *InvoiceService) Get(ctx context.Context, id int) (*Invoice, error) {
	path := fmt.Sprintf("/invoices/get/%d", id)
	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	if len(resp.Invoices) == 0 {
		return nil, fmt.Errorf("wfirma: invoice %d not found", id)
	}

	return &resp.Invoices[0], nil
}

// List lists invoices with pagination.
func (s *InvoiceService) List(ctx context.Context, params ListInvoicesParams) ([]Invoice, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	body := map[string]any{
		"page": map[string]any{
			"page":  page,
			"limit": limit,
		},
	}

	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodPost, "/invoices/find", body, &resp); err != nil {
		return nil, err
	}

	return resp.Invoices, nil
}

// DownloadPDF downloads the PDF of an invoice by its ID.
func (s *InvoiceService) DownloadPDF(ctx context.Context, id int) ([]byte, error) {
	path := fmt.Sprintf("/invoices/download/%d", id)
	raw, err := s.client.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// invoiceListResponse wraps the wFirma API list response.
type invoiceListResponse struct {
	Invoices []Invoice `json:"invoices"`
}

// do performs a JSON API request and decodes the response.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("wfirma: failed to decode response: %w", err)
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
			return nil, fmt.Errorf("wfirma: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("wfirma: failed to create request: %w", err)
	}

	req.Header.Set("X-API-KEY", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wfirma: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wfirma: failed to read response: %w", err)
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

// ParseExternalID converts a string external ID to an int for wFirma API calls.
func ParseExternalID(externalID string) (int, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return 0, fmt.Errorf("wfirma: invalid external ID %q: %w", externalID, err)
	}
	return id, nil
}

// FormatDate formats a time.Time to the wFirma date format (YYYY-MM-DD).
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// APIError represents an error response from the wFirma API.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return "wfirma: " + e.Message
	}
	return fmt.Sprintf("wfirma: unexpected status %d", e.StatusCode)
}
