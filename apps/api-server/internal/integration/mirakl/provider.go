package mirakl

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	miraklsdk "github.com/openoms-org/openoms/packages/mirakl-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("mirakl", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// MiraklCredentials is the JSON structure stored in encrypted integration credentials.
type MiraklCredentials struct {
	BaseURL string `json:"base_url"` // marketplace-specific URL
	APIKey  string `json:"api_key"`
}

// Provider implements integration.MarketplaceProvider for Mirakl-based marketplaces.
type Provider struct {
	client *miraklsdk.Client
	logger *slog.Logger
}

// NewProvider creates a Mirakl MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds MiraklCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("mirakl: parse credentials: %w", err)
	}

	if creds.BaseURL == "" {
		return nil, fmt.Errorf("mirakl: base_url is required")
	}
	if creds.APIKey == "" {
		return nil, fmt.Errorf("mirakl: api_key is required")
	}

	client := miraklsdk.NewClient(creds.BaseURL, creds.APIKey)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "mirakl"),
	}, nil
}

func (p *Provider) ProviderName() string { return "mirakl" }

// PollOrders polls Mirakl for orders updated since the cursor timestamp.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	resp, err := p.client.Orders.List(ctx, cursor)
	if err != nil {
		return nil, cursor, fmt.Errorf("mirakl: poll orders: %w", err)
	}

	if len(resp.Orders) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, o := range resp.Orders {
		mo := p.mapMiraklOrder(&o)
		orders = append(orders, mo)

		// Use the order's created date as cursor for next poll
		if o.CreatedDate > newCursor {
			newCursor = o.CreatedDate
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from Mirakl by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	order, err := p.client.Orders.Get(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("mirakl: get order %s: %w", externalID, err)
	}
	mo := p.mapMiraklOrder(order)
	return &mo, nil
}

// PushOffer is a stub — Mirakl offer creation uses a separate import process.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	return "", fmt.Errorf("mirakl: PushOffer not supported (use Mirakl import)")
}

// UpdateStock updates the stock quantity for a Mirakl offer identified by SKU.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	updates := []miraklsdk.OfferUpdate{
		{SKU: externalOfferID, Quantity: quantity},
	}
	return p.client.Offers.UpdateOffers(ctx, updates)
}

// UpdatePrice updates the price for a Mirakl offer identified by SKU.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	updates := []miraklsdk.OfferUpdate{
		{SKU: externalOfferID, Price: price},
	}
	return p.client.Offers.UpdateOffers(ctx, updates)
}

// mapMiraklOrder converts a Mirakl SDK Order to the normalized MarketplaceOrder.
func (p *Provider) mapMiraklOrder(o *miraklsdk.Order) integration.MarketplaceOrder {
	customerName := fmt.Sprintf("%s %s", o.Customer.FirstName, o.Customer.LastName)

	mo := integration.MarketplaceOrder{
		ExternalID:     o.ID,
		ExternalStatus: o.Status,
		CustomerName:   customerName,
		CustomerEmail:  o.Customer.Email,
		ShippingAddress: integration.ShippingAddress{
			Name:       fmt.Sprintf("%s %s", o.ShippingAddress.FirstName, o.ShippingAddress.LastName),
			Street:     o.ShippingAddress.Street1,
			City:       o.ShippingAddress.City,
			PostalCode: o.ShippingAddress.ZipCode,
			Country:    o.ShippingAddress.Country,
			Phone:      o.ShippingAddress.Phone,
			Email:      o.Customer.Email,
		},
		TotalAmount:   o.TotalPrice,
		Currency:      o.CurrencyCode,
		PaymentMethod: o.PaymentType,
	}

	// Parse ordered_at from Mirakl's created_date
	if t, err := time.Parse(time.RFC3339, o.CreatedDate); err == nil {
		mo.OrderedAt = t
	}

	// Map order status
	if omsStatus, ok := miraklsdk.MapStatus(o.Status); ok {
		mo.RawData = map[string]any{
			"mirakl_status": o.Status,
			"oms_status":    omsStatus,
		}
	}

	// Shipping address street2
	if o.ShippingAddress.Street2 != "" {
		mo.ShippingAddress.Street += ", " + o.ShippingAddress.Street2
	}

	// Line items
	for _, li := range o.OrderLines {
		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: li.ID,
			Name:       li.ProductTitle,
			SKU:        li.OfferSKU,
			Quantity:   li.Quantity,
			UnitPrice:  li.Price,
			TotalPrice: li.Price * float64(li.Quantity),
		})
	}

	return mo
}
