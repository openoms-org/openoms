package infakt

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
	defaultBaseURL = "https://api.infakt.pl/api/v3"
)

// Client is an inFakt API client.
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

// NewClient creates a new inFakt API client.
// apiKey is the API key for authentication via the X-inFakt-ApiKey header.
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

// Invoice represents an inFakt invoice.
type Invoice struct {
	ID            int               `json:"id"`
	Kind          string            `json:"kind"` // vat, proforma, final, correction
	Number        string            `json:"number"`
	InvoiceDate   string            `json:"invoice_date"`
	SaleDate      string            `json:"sale_date"`
	PaymentDate   string            `json:"payment_date"`
	PaymentMethod string            `json:"payment_method"`
	Currency      string            `json:"currency"`
	Status        string            `json:"status"`
	NetPrice      string            `json:"net_price"`
	GrossPrice    string            `json:"gross_price"`
	Client        InvoiceClient     `json:"client"`
	Services      []InvoiceLineItem `json:"services"`
}

// InvoiceClient represents client (buyer) information on an inFakt invoice.
type InvoiceClient struct {
	Name       string `json:"name"`
	NIP        string `json:"nip"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	Email      string `json:"email,omitempty"`
}

// InvoiceLineItem represents a line item (service) on an inFakt invoice.
type InvoiceLineItem struct {
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	Quantity     int     `json:"quantity"`
	UnitNetPrice float64 `json:"unit_net_price"`
	TaxSymbol    string  `json:"tax_symbol"` // "23", "8", "5", "0", "zw", "np"
}

// CreateInvoiceRequest is the request body for creating an invoice.
type CreateInvoiceRequest struct {
	Kind          string            `json:"kind"` // vat, proforma, final, correction
	InvoiceDate   string            `json:"invoice_date"`
	SaleDate      string            `json:"sale_date"`
	PaymentDate   string            `json:"payment_date"`
	PaymentMethod string            `json:"payment_method,omitempty"`
	Currency      string            `json:"currency,omitempty"`
	Client        InvoiceClient     `json:"client"`
	Services      []InvoiceLineItem `json:"services"`
}

// ListInvoicesParams are query parameters for listing invoices.
type ListInvoicesParams struct {
	Page  int
	Limit int
}

// Create creates a new invoice in inFakt.
func (s *InvoiceService) Create(ctx context.Context, req CreateInvoiceRequest) (*Invoice, error) {
	body := map[string]any{
		"invoice": req,
	}

	var resp invoiceResponse
	if err := s.client.do(ctx, http.MethodPost, "/invoices.json", body, &resp); err != nil {
		return nil, err
	}

	return &resp.Invoice, nil
}

// Get retrieves an invoice by its ID.
func (s *InvoiceService) Get(ctx context.Context, id int) (*Invoice, error) {
	path := fmt.Sprintf("/invoices/%d.json", id)
	var resp invoiceResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp.Invoice, nil
}

// List lists invoices with pagination.
func (s *InvoiceService) List(ctx context.Context, params ListInvoicesParams) ([]Invoice, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 25
	}

	path := fmt.Sprintf("/invoices.json?page=%d&per_page=%d", page, limit)
	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Entities, nil
}

// DownloadPDF downloads the PDF of an invoice by its ID.
func (s *InvoiceService) DownloadPDF(ctx context.Context, id int) ([]byte, error) {
	path := fmt.Sprintf("/invoices/%d.pdf", id)
	raw, err := s.client.doRawWithLimit(ctx, http.MethodGet, path, nil, 50<<20)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// SendByEmail sends an invoice to the specified email address.
func (s *InvoiceService) SendByEmail(ctx context.Context, id int, email string) error {
	path := fmt.Sprintf("/invoices/%d/deliver_via_email.json", id)
	body := map[string]any{
		"print_type": "original",
		"locale":     "pl",
		"recipient":  email,
	}
	return s.client.do(ctx, http.MethodPost, path, body, nil)
}

// invoiceResponse wraps a single invoice response.
type invoiceResponse struct {
	Invoice Invoice `json:"invoice"`
}

// invoiceListResponse wraps the inFakt API list response.
type invoiceListResponse struct {
	Entities []Invoice `json:"entities"`
	Total    int       `json:"total"`
}

// do performs a JSON API request and decodes the response.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}

	if result != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("infakt: failed to decode response: %w", err)
		}
	}

	return nil
}

// doRaw performs an API request and returns the raw response body,
// capped at 10 MiB (sufficient for JSON API responses).
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	return c.doRawWithLimit(ctx, method, path, body, 10<<20)
}

// doRawWithLimit performs an API request and returns the raw response body,
// capped at maxBytes. Callers downloading binary payloads (e.g. PDFs) should
// pass a higher limit than the default used by doRaw.
func (c *Client) doRawWithLimit(ctx context.Context, method, path string, body any, maxBytes int64) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("infakt: failed to encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("infakt: failed to create request: %w", err)
	}

	req.Header.Set("X-inFakt-ApiKey", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("infakt: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("infakt: failed to read response: %w", err)
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

// ParseExternalID converts a string external ID to an int for inFakt API calls.
func ParseExternalID(externalID string) (int, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return 0, fmt.Errorf("infakt: invalid external ID %q: %w", externalID, err)
	}
	return id, nil
}

// FormatDate formats a time.Time to the inFakt date format (YYYY-MM-DD).
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// APIError represents an error response from the inFakt API.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return "infakt: " + e.Message
	}
	return fmt.Sprintf("infakt: unexpected status %d", e.StatusCode)
}
