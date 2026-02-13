package amazon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("amazon", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// AmazonCredentials is the JSON structure stored in encrypted integration credentials.
type AmazonCredentials struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	RefreshToken  string `json:"refresh_token"`
	MarketplaceID string `json:"marketplace_id"` // e.g. "A1C3SOZRARQ6R3" for Amazon.pl
	Sandbox       bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for Amazon SP-API.
type Provider struct {
	client        *amazonsdk.Client
	marketplaceID string
	logger        *slog.Logger
}

// NewProvider creates an Amazon MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds AmazonCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("amazon: parse credentials: %w", err)
	}

	var opts []amazonsdk.Option
	opts = append(opts, amazonsdk.WithRefreshToken(creds.RefreshToken))

	if creds.Sandbox {
		opts = append(opts, amazonsdk.WithSandbox())
	}

	client := amazonsdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)

	return &Provider{
		client:        client,
		marketplaceID: creds.MarketplaceID,
		logger:        slog.Default().With("provider", "amazon"),
	}, nil
}

func (p *Provider) ProviderName() string { return "amazon" }

// PollOrders polls Amazon for orders created after the given cursor (ISO8601 timestamp).
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	createdAfter := cursor
	if createdAfter == "" {
		createdAfter = time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	}

	var marketplaceIDs []string
	if p.marketplaceID != "" {
		marketplaceIDs = []string{p.marketplaceID}
	}

	var allOrders []integration.MarketplaceOrder
	newCursor := cursor
	nextToken := ""

	for {
		resp, err := p.client.Orders.List(ctx, createdAfter, marketplaceIDs, nextToken)
		if err != nil {
			return nil, cursor, fmt.Errorf("amazon: list orders: %w", err)
		}

		for _, order := range resp.Payload.Orders {
			// Rate limit: Amazon allows ~1 req/sec for orders
			time.Sleep(time.Second)

			items, err := p.fetchAllItems(ctx, order.AmazonOrderID)
			if err != nil {
				p.logger.Error("failed to fetch order items", "order_id", order.AmazonOrderID, "error", err)
				continue
			}

			mo := p.mapAmazonOrder(&order, items)
			allOrders = append(allOrders, mo)

			// Use LastUpdateDate as cursor if available, else PurchaseDate
			if order.LastUpdateDate != "" {
				newCursor = order.LastUpdateDate
			} else if order.PurchaseDate != "" {
				newCursor = order.PurchaseDate
			}
		}

		if resp.Payload.NextToken == "" {
			break
		}
		nextToken = resp.Payload.NextToken
		time.Sleep(time.Second) // rate limit between pages
	}

	return allOrders, newCursor, nil
}

// fetchAllItems retrieves all order items, handling pagination.
func (p *Provider) fetchAllItems(ctx context.Context, orderID string) ([]amazonsdk.OrderItem, error) {
	var all []amazonsdk.OrderItem
	nextToken := ""

	for {
		resp, err := p.client.Orders.GetItems(ctx, orderID, nextToken)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Payload.OrderItems...)

		if resp.Payload.NextToken == "" {
			break
		}
		nextToken = resp.Payload.NextToken
		time.Sleep(time.Second) // rate limit
	}

	return all, nil
}

// GetOrder retrieves a single order from Amazon by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	order, err := p.client.Orders.Get(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("amazon: get order %s: %w", externalID, err)
	}

	items, err := p.fetchAllItems(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("amazon: get order items %s: %w", externalID, err)
	}

	mo := p.mapAmazonOrder(order, items)
	return &mo, nil
}

// PushOffer is not implemented for Amazon (requires Feeds API).
func (p *Provider) PushOffer(_ context.Context, _ *model.Product, _ map[string]any) (string, error) {
	return "", fmt.Errorf("amazon: PushOffer not implemented (use Amazon Feeds API)")
}

// UpdateStock is not implemented for Amazon (requires Feeds API).
func (p *Provider) UpdateStock(_ context.Context, _ string, _ int) error {
	return fmt.Errorf("amazon: UpdateStock not implemented (use Amazon Feeds API)")
}

// UpdatePrice is not implemented for Amazon (requires Feeds API).
func (p *Provider) UpdatePrice(_ context.Context, _ string, _ float64) error {
	return fmt.Errorf("amazon: UpdatePrice not implemented (use Amazon Feeds API)")
}

// mapAmazonOrder converts Amazon order + items to the normalized MarketplaceOrder.
func (p *Provider) mapAmazonOrder(o *amazonsdk.Order, items []amazonsdk.OrderItem) integration.MarketplaceOrder {
	mo := integration.MarketplaceOrder{
		ExternalID:     o.AmazonOrderID,
		ExternalStatus: o.OrderStatus,
		Currency:       "PLN",
		RawData: map[string]any{
			"fulfillment_channel": o.FulfillmentChannel,
			"marketplace_id":     o.MarketplaceID,
			"payment_method":     o.PaymentMethod,
		},
	}

	// Parse purchase date
	if t, err := time.Parse(time.RFC3339, o.PurchaseDate); err == nil {
		mo.OrderedAt = t
	}

	// Buyer info
	if o.BuyerInfo != nil {
		mo.CustomerEmail = o.BuyerInfo.BuyerEmail
	}

	// Shipping address
	if o.ShippingAddress != nil {
		mo.CustomerName = o.ShippingAddress.Name
		mo.ShippingAddress = model.ShippingAddress{
			Name:       o.ShippingAddress.Name,
			Street:     o.ShippingAddress.AddressLine1,
			City:       o.ShippingAddress.City,
			PostalCode: o.ShippingAddress.PostalCode,
			Country:    o.ShippingAddress.CountryCode,
			Phone:      o.ShippingAddress.Phone,
		}
		if o.ShippingAddress.AddressLine2 != "" {
			mo.ShippingAddress.Street += ", " + o.ShippingAddress.AddressLine2
		}
		if o.BuyerInfo != nil {
			mo.ShippingAddress.Email = o.BuyerInfo.BuyerEmail
		}
	}

	// Phone from address
	if o.ShippingAddress != nil && o.ShippingAddress.Phone != "" {
		mo.CustomerPhone = o.ShippingAddress.Phone
	}

	// Payment method
	mo.PaymentMethod = o.PaymentMethod

	// Order total
	if o.OrderTotal != nil {
		if amount, err := strconv.ParseFloat(o.OrderTotal.Amount, 64); err == nil {
			mo.TotalAmount = amount
		}
		if o.OrderTotal.CurrencyCode != "" {
			mo.Currency = o.OrderTotal.CurrencyCode
		}
	}

	// Payment status inference
	omsStatus, _ := amazonsdk.MapStatus(o.OrderStatus)
	if omsStatus == "shipped" || omsStatus == "confirmed" {
		mo.PaymentStatus = "paid"
	} else {
		mo.PaymentStatus = "pending"
	}

	// Line items
	var itemsTotal float64
	for _, item := range items {
		unitPrice := 0.0
		if item.ItemPrice != nil {
			if p, err := strconv.ParseFloat(item.ItemPrice.Amount, 64); err == nil {
				// Amazon ItemPrice is total for all quantity
				if item.QuantityOrdered > 0 {
					unitPrice = p / float64(item.QuantityOrdered)
				} else {
					unitPrice = p
				}
			}
		}

		totalPrice := unitPrice * float64(item.QuantityOrdered)
		itemsTotal += totalPrice

		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: item.OrderItemID,
			Name:       item.Title,
			SKU:        item.SellerSKU,
			EAN:        item.ASIN,
			Quantity:   item.QuantityOrdered,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		})
	}

	// If no OrderTotal, use items sum
	if mo.TotalAmount <= 0 {
		mo.TotalAmount = itemsTotal
	}

	return mo
}
