package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// InvoiceRequest contains all data needed to create an invoice via a provider.
type InvoiceRequest struct {
	OrderID       string        `json:"order_id"`
	CustomerName  string        `json:"customer_name"`
	CustomerEmail string        `json:"customer_email,omitempty"`
	NIP           string        `json:"nip,omitempty"`
	Items         []InvoiceItem `json:"items"`
	TotalNet      float64       `json:"total_net"`
	TotalGross    float64       `json:"total_gross"`
	Currency      string        `json:"currency"`
	IssueDate     time.Time     `json:"issue_date"`
	DueDate       time.Time     `json:"due_date"`
	PaymentMethod string        `json:"payment_method,omitempty"`
	Notes         string        `json:"notes,omitempty"`
}

// InvoiceItem represents a single line item on an invoice.
type InvoiceItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	NetPrice float64 `json:"net_price"`
	TaxRate  int     `json:"tax_rate"`
	Unit     string  `json:"unit,omitempty"`
}

// InvoiceResult is returned after an invoice operation.
type InvoiceResult struct {
	ExternalID     string `json:"external_id"`
	ExternalNumber string `json:"external_number"`
	PDFURL         string `json:"pdf_url,omitempty"`
	Status         string `json:"status"`
}

// InvoicingProvider defines the interface for invoicing integrations.
type InvoicingProvider interface {
	ProviderName() string
	CreateInvoice(ctx context.Context, req InvoiceRequest) (*InvoiceResult, error)
	GetInvoice(ctx context.Context, externalID string) (*InvoiceResult, error)
	GetPDF(ctx context.Context, externalID string) ([]byte, error)
	CancelInvoice(ctx context.Context, externalID string) error
}

// InvoicingProviderFactory is a constructor function for invoicing providers.
type InvoicingProviderFactory func(credentials json.RawMessage, settings json.RawMessage) (InvoicingProvider, error)

var (
	invoicingFactories   = map[string]InvoicingProviderFactory{}
	invoicingFactoriesMu sync.RWMutex
)

// RegisterInvoicingProvider registers a factory for the given invoicing provider name.
func RegisterInvoicingProvider(name string, factory InvoicingProviderFactory) {
	invoicingFactoriesMu.Lock()
	defer invoicingFactoriesMu.Unlock()
	invoicingFactories[name] = factory
}

// NewInvoicingProvider creates an InvoicingProvider for the given provider name.
func NewInvoicingProvider(provider string, credentials json.RawMessage, settings json.RawMessage) (InvoicingProvider, error) {
	invoicingFactoriesMu.RLock()
	factory, ok := invoicingFactories[provider]
	invoicingFactoriesMu.RUnlock()

	if ok {
		return factory(credentials, settings)
	}

	return nil, fmt.Errorf("unknown invoicing provider: %q", provider)
}
