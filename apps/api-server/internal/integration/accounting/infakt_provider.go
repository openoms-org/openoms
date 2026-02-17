package accounting

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/openoms-org/openoms/packages/infakt-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterInvoicingProvider("infakt", func(credentials json.RawMessage, settings json.RawMessage) (integration.InvoicingProvider, error) {
		return NewInFaktProvider(credentials, settings)
	})
}

// InFaktCredentials is the JSON structure stored in encrypted integration credentials.
type InFaktCredentials struct {
	APIKey string `json:"api_key"`
}

// InFaktProvider implements integration.InvoicingProvider for inFakt.
type InFaktProvider struct {
	client *sdk.Client
}

// NewInFaktProvider creates a new inFakt invoicing provider.
func NewInFaktProvider(credentials json.RawMessage, settings json.RawMessage) (*InFaktProvider, error) {
	var creds InFaktCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("infakt: invalid credentials: %w", err)
	}
	if creds.APIKey == "" {
		return nil, fmt.Errorf("infakt: api_key is required")
	}

	client := sdk.NewClient(creds.APIKey)

	return &InFaktProvider{client: client}, nil
}

func (p *InFaktProvider) ProviderName() string {
	return "infakt"
}

func (p *InFaktProvider) CreateInvoice(ctx context.Context, req integration.InvoiceRequest) (*integration.InvoiceResult, error) {
	services := make([]sdk.InvoiceService_, 0, len(req.Items))
	for _, item := range req.Items {
		taxSymbol := fmt.Sprintf("%d", item.TaxRate)
		services = append(services, sdk.InvoiceService_{
			Name:         item.Name,
			Unit:         item.Unit,
			Quantity:     item.Quantity,
			UnitNetPrice: item.NetPrice,
			TaxSymbol:    taxSymbol,
		})
	}

	data := sdk.CreateInvoiceRequest{
		Kind:          "vat",
		InvoiceDate:   sdk.FormatDate(req.IssueDate),
		SaleDate:      sdk.FormatDate(req.IssueDate),
		PaymentDate:   sdk.FormatDate(req.DueDate),
		PaymentMethod: req.PaymentMethod,
		Currency:      req.Currency,
		Client: sdk.InvoiceClient{
			Name:  req.CustomerName,
			NIP:   req.NIP,
			Email: req.CustomerEmail,
		},
		Services: services,
	}

	resp, err := p.client.Invoices.Create(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("infakt: create invoice: %w", err)
	}

	return &integration.InvoiceResult{
		ExternalID:     strconv.Itoa(resp.ID),
		ExternalNumber: resp.Number,
		Status:         mapInFaktStatus(resp.Status),
	}, nil
}

func (p *InFaktProvider) GetInvoice(ctx context.Context, externalID string) (*integration.InvoiceResult, error) {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Invoices.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("infakt: get invoice: %w", err)
	}

	return &integration.InvoiceResult{
		ExternalID:     strconv.Itoa(resp.ID),
		ExternalNumber: resp.Number,
		Status:         mapInFaktStatus(resp.Status),
	}, nil
}

func (p *InFaktProvider) GetPDF(ctx context.Context, externalID string) ([]byte, error) {
	id, err := sdk.ParseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	data, err := p.client.Invoices.DownloadPDF(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("infakt: get pdf: %w", err)
	}

	return data, nil
}

func (p *InFaktProvider) CancelInvoice(ctx context.Context, externalID string) error {
	// inFakt does not expose a direct cancel endpoint.
	// Invoices should be corrected instead.
	return fmt.Errorf("infakt: cancel not supported, use correction invoice instead")
}

// mapInFaktStatus maps inFakt invoice status to our internal status.
func mapInFaktStatus(status string) string {
	switch status {
	case "created", "saved":
		return "issued"
	case "sent":
		return "sent"
	case "paid":
		return "paid"
	case "partial":
		return "partially_paid"
	case "draft":
		return "draft"
	default:
		return status
	}
}
