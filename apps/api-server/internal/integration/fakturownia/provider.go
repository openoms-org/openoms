package fakturownia

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/openoms-org/openoms/packages/fakturownia-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterInvoicingProvider("fakturownia", func(credentials json.RawMessage, settings json.RawMessage) (integration.InvoicingProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// Credentials is the JSON structure stored in encrypted integration credentials.
type Credentials struct {
	APIToken  string `json:"api_token"`
	Subdomain string `json:"subdomain"`
}

// Provider implements integration.InvoicingProvider for Fakturownia.
type Provider struct {
	client *sdk.Client
}

// NewProvider creates a new Fakturownia invoicing provider.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds Credentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("fakturownia: invalid credentials: %w", err)
	}
	if creds.APIToken == "" || creds.Subdomain == "" {
		return nil, fmt.Errorf("fakturownia: api_token and subdomain are required")
	}

	client := sdk.NewClient(creds.Subdomain, creds.APIToken)

	return &Provider{client: client}, nil
}

func (p *Provider) ProviderName() string {
	return "fakturownia"
}

func (p *Provider) CreateInvoice(ctx context.Context, req integration.InvoiceRequest) (*integration.InvoiceResult, error) {
	positions := make([]sdk.InvoicePosition, 0, len(req.Items))
	for _, item := range req.Items {
		positions = append(positions, sdk.InvoicePosition{
			Name:     item.Name,
			Quantity: item.Quantity,
			NetPrice: item.NetPrice,
			Tax:      item.TaxRate,
			Unit:     item.Unit,
		})
	}

	data := sdk.InvoiceRequestData{
		Kind:        "vat",
		SellDate:    req.IssueDate.Format("2006-01-02"),
		IssueDate:   req.IssueDate.Format("2006-01-02"),
		PaymentTo:   req.DueDate.Format("2006-01-02"),
		BuyerName:   req.CustomerName,
		BuyerEmail:  req.CustomerEmail,
		BuyerTaxNo:  req.NIP,
		PaymentType: req.PaymentMethod,
		Currency:    req.Currency,
		Description: req.Notes,
		Positions:   positions,
	}

	resp, err := p.client.Invoices.Create(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("fakturownia: create invoice: %w", err)
	}

	return &integration.InvoiceResult{
		ExternalID:     strconv.Itoa(resp.ID),
		ExternalNumber: resp.Number,
		PDFURL:         resp.ViewURL,
		Status:         mapStatus(resp.Status),
	}, nil
}

func (p *Provider) GetInvoice(ctx context.Context, externalID string) (*integration.InvoiceResult, error) {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Invoices.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fakturownia: get invoice: %w", err)
	}

	return &integration.InvoiceResult{
		ExternalID:     strconv.Itoa(resp.ID),
		ExternalNumber: resp.Number,
		PDFURL:         resp.ViewURL,
		Status:         mapStatus(resp.Status),
	}, nil
}

func (p *Provider) GetPDF(ctx context.Context, externalID string) ([]byte, error) {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	data, err := p.client.Invoices.GetPDF(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fakturownia: get pdf: %w", err)
	}

	return data, nil
}

func (p *Provider) CancelInvoice(ctx context.Context, externalID string) error {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return err
	}

	if err := p.client.Invoices.Cancel(ctx, id); err != nil {
		return fmt.Errorf("fakturownia: cancel invoice: %w", err)
	}
	return nil
}

// mapStatus maps Fakturownia invoice status to our internal status.
func mapStatus(status string) string {
	switch status {
	case "issued":
		return "issued"
	case "sent":
		return "sent"
	case "paid":
		return "paid"
	case "partial":
		return "partially_paid"
	case "rejected":
		return "cancelled"
	default:
		return status
	}
}
