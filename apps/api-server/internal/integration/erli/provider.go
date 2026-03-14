// Package erli implements the Erli.pl marketplace provider.
package erli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	erlisdk "github.com/openoms-org/openoms/packages/erli-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("erli", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// Credentials is the JSON structure stored in encrypted integration credentials.
type Credentials struct {
	APIToken string `json:"api_token"`
	BaseURL  string `json:"base_url,omitempty"`
	Sandbox  bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for Erli.pl.
type Provider struct {
	client *erlisdk.Client
	logger *slog.Logger
}

// NewProvider creates an Erli MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, _ json.RawMessage) (*Provider, error) {
	var creds Credentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("erli: parse credentials: %w", err)
	}

	if creds.APIToken == "" {
		return nil, fmt.Errorf("erli: api_token is required")
	}

	if creds.Sandbox {
		return nil, fmt.Errorf("erli: sandbox mode requires explicit base URL — set 'base_url' in credentials (contact Erli BOK for sandbox URL)")
	}

	var opts []erlisdk.Option
	if creds.BaseURL != "" {
		opts = append(opts, erlisdk.WithBaseURL(creds.BaseURL))
	}

	client := erlisdk.NewClient(creds.APIToken, opts...)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "erli"),
	}, nil
}

// ProviderName returns the marketplace provider identifier.
func (p *Provider) ProviderName() string { return "erli" }

// PollOrders polls Erli for purchased orders using cursor-based pagination.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	resp, err := p.client.Orders.List(ctx, cursor)
	if err != nil {
		return nil, cursor, fmt.Errorf("erli: poll orders: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	for _, o := range resp.Data {
		mo := p.mapErliOrder(&o)
		orders = append(orders, mo)
	}

	newCursor := cursor
	if resp.Meta.NextCursor != "" {
		newCursor = resp.Meta.NextCursor
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from Erli by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	order, err := p.client.Orders.Get(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("erli: get order %s: %w", externalID, err)
	}
	mo := p.mapErliOrder(order)
	return &mo, nil
}

// PushOffer creates a product on Erli using the product SKU as the externalID path parameter.
// SKU is mandatory — Erli identifies products by the seller's own product ID.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	if product == nil || product.SKU == nil || *product.SKU == "" {
		return "", fmt.Errorf("erli: SKU is required as product externalId")
	}
	externalID := *product.SKU

	req := erlisdk.CreateOfferRequest{}
	req.Title = product.Name
	req.Description = model.StripHTMLTags(product.DescriptionLong)
	req.Price = product.Price
	req.Stock = product.StockQuantity
	req.SKU = externalID
	if product.EAN != nil {
		req.EAN = *product.EAN
	}

	// listingData overrides take precedence over product defaults.
	if title, ok := listingData["title"].(string); ok && title != "" {
		req.Title = title
	}
	if price, ok := listingData["price"].(float64); ok {
		req.Price = price
	}
	switch v := listingData["stock"].(type) {
	case int:
		req.Stock = v
	case float64:
		req.Stock = int(v)
	}
	if catID, ok := listingData["category_id"].(string); ok {
		req.CategoryID = catID
	}

	offerID, err := p.client.Offers.Create(ctx, externalID, req)
	if errors.Is(err, erlisdk.ErrProductPendingValidation) {
		p.logger.Warn("erli: product accepted for async validation, not yet live",
			"external_id", offerID)
		return offerID, nil
	}
	if err != nil {
		return "", fmt.Errorf("erli: create offer: %w", err)
	}
	return offerID, nil
}

// UpdateStock updates the stock quantity for an Erli product.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	return p.client.Offers.UpdateStock(ctx, externalOfferID, quantity)
}

// UpdatePrice updates the price for an Erli product.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	return p.client.Offers.UpdatePrice(ctx, externalOfferID, price)
}

// mapErliOrder converts an Erli SDK Order to the normalized MarketplaceOrder.
func (p *Provider) mapErliOrder(o *erlisdk.Order) integration.MarketplaceOrder {
	mo := integration.MarketplaceOrder{
		ExternalID:     o.ID,
		ExternalStatus: o.Status,
		CustomerName:   o.BuyerName,
		CustomerEmail:  o.BuyerEmail,
		CustomerPhone:  o.BuyerPhone,
		ShippingAddress: model.ShippingAddress{
			Name:       o.Address.Name,
			Street:     o.Address.Street,
			City:       o.Address.City,
			PostalCode: o.Address.PostCode,
			Country:    o.Address.Country,
		},
		TotalAmount:   o.TotalAmount,
		Currency:      o.Currency,
		PaymentStatus: o.PaymentStatus,
	}

	// Parse ordered_at
	if t, err := time.Parse(time.RFC3339, o.CreatedAt); err == nil {
		mo.OrderedAt = t
	}

	// Map order status
	if omsStatus, ok := erlisdk.MapStatus(o.Status); ok {
		mo.RawData = map[string]any{
			"erli_status": o.Status,
			"oms_status":  omsStatus,
		}
	} else {
		p.logger.Warn("erli: unknown order status — not in statusmap.go; skipping OMS status mapping",
			"erli_status", o.Status, "order_id", o.ID)
	}

	// Line items
	for _, item := range o.Items {
		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: item.ID,
			Name:       item.Name,
			SKU:        item.SKU,
			Quantity:   item.Quantity,
			UnitPrice:  item.Price,
			TotalPrice: item.Price * float64(item.Quantity),
		})
	}

	return mo
}
