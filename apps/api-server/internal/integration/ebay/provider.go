package ebay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("ebay", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// EbayCredentials is the JSON structure stored in encrypted integration credentials.
type EbayCredentials struct {
	AppID        string `json:"app_id"`
	CertID       string `json:"cert_id"`
	DevID        string `json:"dev_id"`
	RefreshToken string `json:"refresh_token"`
	Sandbox      bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for eBay.
type Provider struct {
	client *ebaysdk.Client
	logger *slog.Logger
}

// NewProvider creates an eBay MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds EbayCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("ebay: parse credentials: %w", err)
	}

	if creds.AppID == "" {
		return nil, fmt.Errorf("ebay: app_id is required")
	}
	if creds.CertID == "" {
		return nil, fmt.Errorf("ebay: cert_id is required")
	}
	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("ebay: refresh_token is required")
	}

	var opts []ebaysdk.Option
	if creds.Sandbox {
		opts = append(opts, ebaysdk.WithSandbox())
	}

	client := ebaysdk.NewClient(creds.AppID, creds.CertID, creds.DevID, creds.RefreshToken, opts...)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "ebay"),
	}, nil
}

func (p *Provider) ProviderName() string { return "ebay" }

// PollOrders polls eBay for orders created after the given cursor (ISO8601 timestamp).
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	filter := ""
	if cursor != "" {
		filter = fmt.Sprintf("creationdate:[%s..]", cursor)
	}

	resp, err := p.client.Orders.GetOrders(ctx, ebaysdk.OrderSearchParams{
		Filter: filter,
		Limit:  50,
	})
	if err != nil {
		return nil, cursor, fmt.Errorf("ebay: poll orders: %w", err)
	}

	if len(resp.Orders) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, o := range resp.Orders {
		mo := p.mapEbayOrder(&o)
		orders = append(orders, mo)

		if o.CreationDate > newCursor {
			newCursor = o.CreationDate
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from eBay by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	order, err := p.client.Orders.GetOrder(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("ebay: get order %s: %w", externalID, err)
	}
	mo := p.mapEbayOrder(order)
	return &mo, nil
}

// PushOffer is not supported for eBay (Trading API is complex and requires separate implementation).
func (p *Provider) PushOffer(_ context.Context, _ *model.Product, _ map[string]any) (string, error) {
	return "", fmt.Errorf("ebay: PushOffer not supported (use eBay Trading API or Inventory API)")
}

// UpdateStock is not supported for eBay via the Fulfillment API.
func (p *Provider) UpdateStock(_ context.Context, _ string, _ int) error {
	return fmt.Errorf("ebay: UpdateStock not supported (use eBay Inventory API)")
}

// UpdatePrice is not supported for eBay via the Fulfillment API.
func (p *Provider) UpdatePrice(_ context.Context, _ string, _ float64) error {
	return fmt.Errorf("ebay: UpdatePrice not supported (use eBay Inventory API)")
}

// mapEbayOrder converts an eBay SDK Order to the normalized MarketplaceOrder.
func (p *Provider) mapEbayOrder(o *ebaysdk.Order) integration.MarketplaceOrder {
	// Parse total amount
	totalAmount, _ := strconv.ParseFloat(o.PricingSummary.Total.Value, 64)

	mo := integration.MarketplaceOrder{
		ExternalID:     o.OrderID,
		ExternalStatus: o.OrderFulfStatus,
		CustomerName:   o.Buyer.Username,
		TotalAmount:    totalAmount,
		Currency:       o.PricingSummary.Total.Currency,
	}

	// Extract buyer registration info if available
	if o.Buyer.BuyerRegInfo != nil {
		mo.CustomerName = o.Buyer.BuyerRegInfo.FullName
		mo.CustomerEmail = o.Buyer.BuyerRegInfo.Email
		if o.Buyer.BuyerRegInfo.PrimaryPhone != nil {
			mo.CustomerPhone = o.Buyer.BuyerRegInfo.PrimaryPhone.PhoneNumber
		}
	}

	// Map payment status
	switch o.OrderPaymentStat {
	case "PAID", "FULLY_REFUNDED", "PARTIALLY_REFUNDED":
		mo.PaymentStatus = "paid"
	default:
		mo.PaymentStatus = "pending"
	}

	// Parse ordered_at from creationDate (ISO8601)
	if t, err := time.Parse(time.RFC3339, o.CreationDate); err == nil {
		mo.OrderedAt = t
	} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", o.CreationDate); err == nil {
		mo.OrderedAt = t
	}

	// Shipping address from fulfillment instructions
	if len(o.FulfillmentSOs) > 0 {
		shipTo := o.FulfillmentSOs[0].ShippingStep.ShipTo
		mo.ShippingAddress = integration.ShippingAddress{
			Name: shipTo.FullName,
		}
		if shipTo.ContactAddress != nil {
			addr := shipTo.ContactAddress
			mo.ShippingAddress.Street = addr.AddressLine1
			if addr.AddressLine2 != "" {
				mo.ShippingAddress.Street += ", " + addr.AddressLine2
			}
			mo.ShippingAddress.City = addr.City
			mo.ShippingAddress.PostalCode = addr.PostalCode
			mo.ShippingAddress.Country = addr.CountryCode
		}
		if shipTo.PrimaryPhone != nil {
			mo.ShippingAddress.Phone = shipTo.PrimaryPhone.PhoneNumber
		}
		mo.ShippingAddress.Email = shipTo.Email
	}

	// Line items
	for _, li := range o.LineItems {
		unitPrice, _ := strconv.ParseFloat(li.LineItemCost.Value, 64)
		totalPrice, _ := strconv.ParseFloat(li.Total.Value, 64)

		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: li.LineItemID,
			Name:       li.Title,
			SKU:        li.SKU,
			Quantity:   li.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		})
	}

	// RawData
	mo.RawData = map[string]any{
		"ebay_order_id":          o.OrderID,
		"ebay_legacy_order_id":   o.LegacyOrderID,
		"ebay_fulfillment_status": o.OrderFulfStatus,
		"ebay_payment_status":    o.OrderPaymentStat,
	}

	return mo
}
