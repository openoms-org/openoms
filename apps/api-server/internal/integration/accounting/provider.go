package accounting

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AccountingProvider defines the interface for accounting/invoicing integrations
// beyond the core InvoicingProvider. It supports richer invoice data models
// (buyer address, per-item VAT rates) needed by wFirma and inFakt.
type AccountingProvider interface {
	Name() string
	CreateInvoice(ctx context.Context, invoice InvoiceData) (*InvoiceResult, error)
	GetInvoice(ctx context.Context, externalID string) (*InvoiceResult, error)
	DownloadPDF(ctx context.Context, externalID string) ([]byte, error)
}

// InvoiceData contains all data needed to create an invoice with an accounting provider.
type InvoiceData struct {
	Type          string // vat, proforma, correction
	OrderID       uuid.UUID
	OrderNumber   string
	IssueDate     time.Time
	SaleDate      time.Time
	PaymentDate   time.Time
	Buyer         BuyerData
	Items         []InvoiceItemData
	Currency      string
	PaymentMethod string
	Notes         string
}

// BuyerData represents the buyer/customer on an invoice.
type BuyerData struct {
	Name       string
	NIP        string
	Street     string
	City       string
	PostalCode string
	Country    string
	Email      string
}

// InvoiceItemData represents a single line item on an invoice.
type InvoiceItemData struct {
	Name      string
	Unit      string
	Quantity  int
	UnitPrice float64 // net price
	VATRate   string  // "23", "8", "5", "0", "zw", "np"
}

// InvoiceResult is returned after an invoice operation.
type InvoiceResult struct {
	ExternalID string
	Number     string
	PDFURL     string
	Status     string
}
