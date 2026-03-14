// Package accounting implements invoicing provider integrations (wFirma).
package accounting

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/openoms-org/openoms/packages/wfirma-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterInvoicingProvider("wfirma", func(credentials json.RawMessage, settings json.RawMessage) (integration.InvoicingProvider, error) {
		return NewWFirmaProvider(credentials, settings)
	})
}

// WFirmaCredentials is the JSON structure stored in encrypted integration credentials.
type WFirmaCredentials struct {
	APIKey string `json:"api_key"`
}

// WFirmaProvider implements integration.InvoicingProvider for wFirma.
type WFirmaProvider struct {
	client *sdk.Client
}

// NewWFirmaProvider creates a new wFirma invoicing provider.
func NewWFirmaProvider(credentials json.RawMessage, _ json.RawMessage) (*WFirmaProvider, error) {
	var creds WFirmaCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("wfirma: invalid credentials: %w", err)
	}
	if creds.APIKey == "" {
		return nil, fmt.Errorf("wfirma: api_key is required")
	}

	client := sdk.NewClient(creds.APIKey)

	return &WFirmaProvider{client: client}, nil
}

// ProviderName returns the invoicing provider identifier.
func (p *WFirmaProvider) ProviderName() string {
	return "wfirma"
}

// CreateInvoice creates a new invoice in wFirma and returns the result.
func (p *WFirmaProvider) CreateInvoice(ctx context.Context, req integration.InvoiceRequest) (*integration.InvoiceResult, error) {
	items := make([]sdk.InvoiceItem, 0, len(req.Items))
	for _, item := range req.Items {
		vatRate := fmt.Sprintf("%d", item.TaxRate)
		items = append(items, sdk.InvoiceItem{
			Name:    item.Name,
			Unit:    item.Unit,
			Count:   item.Quantity,
			Price:   item.NetPrice,
			VATRate: vatRate,
		})
	}

	data := sdk.CreateInvoiceRequest{
		Type:          "vat",
		IssueDate:     sdk.FormatDate(req.IssueDate),
		SaleDate:      sdk.FormatDate(req.IssueDate),
		PaymentDate:   sdk.FormatDate(req.DueDate),
		PaymentMethod: req.PaymentMethod,
		Currency:      req.Currency,
		Notes:         req.Notes,
		Buyer: sdk.InvoiceBuyer{
			Name: req.CustomerName,
			NIP:  req.NIP,
		},
		Items: items,
	}

	resp, err := p.client.Invoices.Create(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("wfirma: create invoice: %w", err)
	}

	return &integration.InvoiceResult{
		ExternalID:     strconv.Itoa(resp.ID),
		ExternalNumber: resp.Number,
		Status:         mapWFirmaStatus(resp.Status),
	}, nil
}

// GetInvoice retrieves a wFirma invoice by its external ID.
func (p *WFirmaProvider) GetInvoice(ctx context.Context, externalID string) (*integration.InvoiceResult, error) {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Invoices.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("wfirma: get invoice: %w", err)
	}

	return &integration.InvoiceResult{
		ExternalID:     strconv.Itoa(resp.ID),
		ExternalNumber: resp.Number,
		Status:         mapWFirmaStatus(resp.Status),
	}, nil
}

// GetPDF downloads the PDF for a wFirma invoice by its external ID.
func (p *WFirmaProvider) GetPDF(ctx context.Context, externalID string) ([]byte, error) {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	data, err := p.client.Invoices.DownloadPDF(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("wfirma: get pdf: %w", err)
	}

	return data, nil
}

// CancelInvoice is not supported by wFirma; use correction invoices instead.
func (p *WFirmaProvider) CancelInvoice(_ context.Context, _ string) error {
	// wFirma does not have a direct cancel endpoint exposed via this SDK.
	// The invoice can be corrected instead (correction invoice).
	return fmt.Errorf("wfirma: cancel not supported, use correction invoice instead")
}

// mapWFirmaStatus maps wFirma invoice status to our internal status.
func mapWFirmaStatus(status string) string {
	switch status {
	case "issued", "created":
		return "issued"
	case "sent":
		return "sent"
	case "paid":
		return "paid"
	case "overdue":
		return "overdue"
	default:
		return status
	}
}
