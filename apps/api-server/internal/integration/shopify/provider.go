// Package shopify implements the Shopify marketplace provider.
package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"time"

	shopifysdk "github.com/openoms-org/openoms/packages/shopify-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

func init() {
	integration.RegisterMarketplaceProvider("shopify", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// Credentials is the JSON structure stored in encrypted integration credentials.
type Credentials struct {
	ShopDomain  string `json:"shop_domain"`
	AccessToken string `json:"access_token"`
	APIVersion  string `json:"api_version,omitempty"`
}

// Provider implements integration.MarketplaceProvider for Shopify.
type Provider struct {
	client *shopifysdk.Client
	logger *slog.Logger
}

// NewProvider creates a Shopify MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, _ json.RawMessage) (*Provider, error) {
	var creds Credentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("shopify: parse credentials: %w", err)
	}

	if creds.ShopDomain == "" {
		return nil, fmt.Errorf("shopify: shop_domain is required")
	}
	if creds.AccessToken == "" {
		return nil, fmt.Errorf("shopify: access_token is required")
	}

	opts := []shopifysdk.Option{
		shopifysdk.WithHTTPClient(netutil.SafeHTTPClient(30 * time.Second)),
	}
	if creds.APIVersion != "" {
		opts = append(opts, shopifysdk.WithAPIVersion(creds.APIVersion))
	}

	client := shopifysdk.NewClient(creds.ShopDomain, creds.AccessToken, opts...)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "shopify"),
	}, nil
}

// ProviderName returns the marketplace provider identifier.
func (p *Provider) ProviderName() string { return "shopify" }

// PollOrders polls Shopify for orders updated after the given cursor.
// The cursor is the updated_at value (ISO8601) of the last polled order.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := shopifysdk.OrderListParams{
		Limit:  50,
		Status: "any",
	}
	if cursor != "" {
		params.UpdatedAtMin = cursor
	}

	shopifyOrders, err := p.client.Orders.List(ctx, params)
	if err != nil {
		return nil, cursor, fmt.Errorf("shopify: poll orders: %w", err)
	}

	if len(shopifyOrders) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, so := range shopifyOrders {
		mo := p.mapShopifyOrder(&so)
		orders = append(orders, mo)

		if so.UpdatedAt > newCursor {
			newCursor = so.UpdatedAt
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from Shopify by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	id, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("shopify: invalid order ID %q: %w", externalID, err)
	}

	order, err := p.client.Orders.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("shopify: get order %s: %w", externalID, err)
	}

	mo := p.mapShopifyOrder(order)
	return &mo, nil
}

// PushOffer creates a Shopify product from a product and listing data.
func (p *Provider) PushOffer(_ context.Context, product *model.Product, listingData map[string]any) (string, error) {
	data := make(map[string]any)
	maps.Copy(data, listingData)
	if _, ok := data["title"]; !ok {
		data["title"] = product.Name
	}

	// Full Shopify product creation requires complex variant/image handling.
	// Return a placeholder external ID; full listing support TBD.
	return fmt.Sprintf("shopify-%s", product.ID.String()), nil
}

// UpdateStock updates the stock quantity for a Shopify product variant.
// externalOfferID should be "inventoryItemID:locationID" or just "inventoryItemID".
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	inventoryItemID, err := strconv.ParseInt(externalOfferID, 10, 64)
	if err != nil {
		return fmt.Errorf("shopify: invalid inventory item ID %q: %w", externalOfferID, err)
	}

	// Get first location's inventory level
	levels, err := p.client.Inventory.GetLevels(ctx, inventoryItemID)
	if err != nil {
		return fmt.Errorf("shopify: get inventory levels: %w", err)
	}
	if len(levels) == 0 {
		return fmt.Errorf("shopify: no inventory levels found for item %d", inventoryItemID)
	}

	return p.client.Inventory.SetLevel(ctx, inventoryItemID, levels[0].LocationID, quantity)
}

// UpdatePrice updates the price for a Shopify product variant.
// externalOfferID should be the variant ID.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	variantID, err := strconv.ParseInt(externalOfferID, 10, 64)
	if err != nil {
		return fmt.Errorf("shopify: invalid variant ID %q: %w", externalOfferID, err)
	}
	return p.client.Products.UpdateVariant(ctx, variantID, map[string]any{
		"price": fmt.Sprintf("%.2f", price),
	})
}

// mapShopifyOrder converts a Shopify order to the normalized MarketplaceOrder.
func (p *Provider) mapShopifyOrder(o *shopifysdk.Order) integration.MarketplaceOrder {
	customerName := ""
	customerEmail := o.Email
	customerPhone := o.Phone

	if o.ShippingAddress != nil {
		customerName = fmt.Sprintf("%s %s", o.ShippingAddress.FirstName, o.ShippingAddress.LastName)
	} else if o.Customer != nil {
		customerName = fmt.Sprintf("%s %s", o.Customer.FirstName, o.Customer.LastName)
		if customerEmail == "" {
			customerEmail = o.Customer.Email
		}
		if customerPhone == "" {
			customerPhone = o.Customer.Phone
		}
	}

	mo := integration.MarketplaceOrder{
		ExternalID:     strconv.FormatInt(o.ID, 10),
		ExternalStatus: o.FinancialStatus,
		CustomerName:   customerName,
		CustomerEmail:  customerEmail,
		CustomerPhone:  customerPhone,
		Currency:       o.Currency,
	}

	// Shipping address
	if o.ShippingAddress != nil {
		mo.ShippingAddress = model.ShippingAddress{
			Name:       fmt.Sprintf("%s %s", o.ShippingAddress.FirstName, o.ShippingAddress.LastName),
			Street:     o.ShippingAddress.Address1,
			City:       o.ShippingAddress.City,
			PostalCode: o.ShippingAddress.Zip,
			Country:    o.ShippingAddress.CountryCode,
			Phone:      o.ShippingAddress.Phone,
			Email:      customerEmail,
		}
	}

	// Billing address
	if o.BillingAddress != nil {
		mo.BillingAddress = &model.ShippingAddress{
			Name:       fmt.Sprintf("%s %s", o.BillingAddress.FirstName, o.BillingAddress.LastName),
			Street:     o.BillingAddress.Address1,
			City:       o.BillingAddress.City,
			PostalCode: o.BillingAddress.Zip,
			Country:    o.BillingAddress.CountryCode,
			Phone:      o.BillingAddress.Phone,
			Email:      customerEmail,
		}
	}

	// Total amount
	totalAmount, _ := strconv.ParseFloat(o.TotalPrice, 64)
	mo.TotalAmount = totalAmount

	// Payment status
	switch o.FinancialStatus {
	case "paid", "partially_paid":
		mo.PaymentStatus = "paid"
	case "refunded", "partially_refunded":
		mo.PaymentStatus = "refunded"
	default:
		mo.PaymentStatus = "pending"
	}

	mo.PaymentMethod = o.Gateway

	// Parse ordered_at
	if t, err := time.Parse(time.RFC3339, o.CreatedAt); err == nil {
		mo.OrderedAt = t
	}

	// Line items
	for _, li := range o.LineItems {
		unitPrice, _ := strconv.ParseFloat(li.Price, 64)
		totalDiscount, _ := strconv.ParseFloat(li.TotalDiscount, 64)
		totalPrice := unitPrice*float64(li.Quantity) - totalDiscount

		externalID := ""
		if li.VariantID != nil {
			externalID = strconv.FormatInt(*li.VariantID, 10)
		} else if li.ProductID != nil {
			externalID = strconv.FormatInt(*li.ProductID, 10)
		}

		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: externalID,
			Name:       li.Name,
			SKU:        li.SKU,
			Quantity:   li.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		})
	}

	// RawData
	mo.RawData = map[string]any{
		"shopify_order_id":   o.ID,
		"order_name":         o.Name,
		"fulfillment_status": o.FulfillmentStatus,
		"gateway":            o.Gateway,
	}
	if o.Note != "" {
		mo.RawData["customer_note"] = o.Note
	}

	return mo
}
