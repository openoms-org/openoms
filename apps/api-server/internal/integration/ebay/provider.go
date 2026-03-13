// Package ebay implements the eBay marketplace provider.
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

// Compile-time interface checks.
var (
	_ integration.BulkStockUpdater   = (*Provider)(nil)
	_ integration.BulkPriceUpdater   = (*Provider)(nil)
	_ integration.ListingDeactivator = (*Provider)(nil)
	_ integration.ListingActivator   = (*Provider)(nil)
)

// Credentials is the JSON structure stored in encrypted integration credentials.
type Credentials struct {
	AppID        string `json:"app_id"`
	CertID       string `json:"cert_id"`
	DevID        string `json:"dev_id"`
	RefreshToken string `json:"refresh_token"`
	Sandbox      bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for eBay.
type Provider struct {
	client   *ebaysdk.Client
	logger   *slog.Logger
	currency string
}

// NewProvider creates an eBay MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds Credentials
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

	type providerSettings struct {
		Currency      string `json:"currency"`
		MarketplaceID string `json:"marketplace_id"`
	}

	var provSettings providerSettings
	if settings != nil {
		_ = json.Unmarshal(settings, &provSettings)
	}
	if provSettings.Currency == "" {
		provSettings.Currency = "PLN"
	}

	return &Provider{
		client:   client,
		logger:   slog.Default().With("provider", "ebay"),
		currency: provSettings.Currency,
	}, nil
}

// ProviderName returns the marketplace provider identifier.
func (p *Provider) ProviderName() string { return "ebay" }

// Client returns the underlying eBay SDK client for direct API calls.
func (p *Provider) Client() *ebaysdk.Client { return p.client }

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

// PushOffer creates an eBay listing via the 3-step flow: inventory item → offer → publish.
// Required listingData keys: category_id, fulfillment_policy_id, return_policy_id.
// Optional: title, description, condition, marketplace_id, payment_policy_id, price_override, stock_override, image_urls.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	if product.SKU == nil || *product.SKU == "" {
		return "", fmt.Errorf("ebay: product SKU is required for listing")
	}
	sku := *product.SKU

	// Step 1: Create/replace inventory item
	item := ebaysdk.InventoryItem{
		Product: ebaysdk.InventoryProduct{
			Title:       getStringOr(listingData, "title", product.Name),
			Description: getStringOr(listingData, "description", ""),
			ImageURLs:   getStringSlice(listingData, "image_urls"),
		},
		Condition: getStringOr(listingData, "condition", "NEW"),
		Availability: &ebaysdk.Availability{
			ShipToLocationAvailability: &ebaysdk.ShipToLocationAvailability{
				Quantity: getIntOr(listingData, "stock_override", product.StockQuantity),
			},
		},
	}

	if product.EAN != nil && *product.EAN != "" {
		item.Product.EAN = []string{*product.EAN}
	}

	if err := p.client.Inventory.CreateOrReplaceInventoryItem(ctx, sku, item); err != nil {
		return "", fmt.Errorf("ebay: create inventory item: %w", err)
	}

	// Step 2: Create offer
	price := product.Price
	if v, ok := listingData["price_override"].(float64); ok && v > 0 {
		price = v
	}

	categoryID := getStringOr(listingData, "category_id", "")
	if categoryID == "" {
		return "", fmt.Errorf("ebay: category_id is required")
	}

	fulfillmentPolicyID := getStringOr(listingData, "fulfillment_policy_id", "")
	returnPolicyID := getStringOr(listingData, "return_policy_id", "")
	if fulfillmentPolicyID == "" || returnPolicyID == "" {
		return "", fmt.Errorf("ebay: fulfillment_policy_id and return_policy_id are required")
	}

	offer := ebaysdk.Offer{
		SKU:           sku,
		MarketplaceID: getStringOr(listingData, "marketplace_id", "EBAY_PL"),
		Format:        "FIXED_PRICE",
		CategoryID:    categoryID,
		PricingSummary: &ebaysdk.OfferPricing{
			Price: ebaysdk.Amount{
				Value:    fmt.Sprintf("%.2f", price),
				Currency: p.currency,
			},
		},
		ListingPolicies: &ebaysdk.ListingPolicies{
			FulfillmentPolicyID: fulfillmentPolicyID,
			ReturnPolicyID:      returnPolicyID,
			PaymentPolicyID:     getStringOr(listingData, "payment_policy_id", ""),
		},
		AvailableQuantity:  getIntOr(listingData, "stock_override", product.StockQuantity),
		ListingDescription: getStringOr(listingData, "description", ""),
	}

	createResp, err := p.client.Offers.CreateOffer(ctx, offer)
	if err != nil {
		return "", fmt.Errorf("ebay: create offer: %w", err)
	}

	// Step 3: Publish
	_, err = p.client.Offers.PublishOffer(ctx, createResp.OfferID)
	if err != nil {
		return "", fmt.Errorf("ebay: publish offer %s: %w", createResp.OfferID, err)
	}

	p.logger.Info("ebay: listing published",
		"sku", sku,
		"offer_id", createResp.OfferID,
	)

	return createResp.OfferID, nil
}

// UpdateStock updates the available quantity for an eBay offer via the Inventory API.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	return p.client.Inventory.UpdateStock(ctx, externalOfferID, quantity)
}

// UpdatePrice updates the price for an eBay offer via the Inventory API.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	return p.client.Inventory.UpdatePrice(ctx, externalOfferID, price, p.currency)
}

// BulkUpdateStock updates stock for multiple eBay offers in a single API call.
// Implements integration.BulkStockUpdater.
func (p *Provider) BulkUpdateStock(ctx context.Context, updates []integration.StockUpdate) error {
	requests := make([]ebaysdk.BulkPriceQuantityRequest, len(updates))
	for i, u := range updates {
		qty := u.Quantity
		requests[i] = ebaysdk.BulkPriceQuantityRequest{
			OfferID:           u.ExternalOfferID,
			AvailableQuantity: &qty,
		}
	}
	_, err := p.client.Inventory.BulkUpdatePriceQuantity(ctx, requests)
	return err
}

// BulkUpdatePrice updates prices for multiple eBay offers in a single API call.
// Implements integration.BulkPriceUpdater.
func (p *Provider) BulkUpdatePrice(ctx context.Context, updates []integration.PriceUpdate) error {
	requests := make([]ebaysdk.BulkPriceQuantityRequest, len(updates))
	for i, u := range updates {
		requests[i] = ebaysdk.BulkPriceQuantityRequest{
			OfferID: u.ExternalOfferID,
			Price: &ebaysdk.Amount{
				Value:    fmt.Sprintf("%.2f", u.Amount),
				Currency: u.Currency,
			},
		}
	}
	_, err := p.client.Inventory.BulkUpdatePriceQuantity(ctx, requests)
	return err
}

// DeactivateOffer withdraws a published offer from eBay.
// Implements integration.ListingDeactivator.
func (p *Provider) DeactivateOffer(ctx context.Context, externalOfferID string) error {
	return p.client.Offers.WithdrawOffer(ctx, externalOfferID)
}

// ActivateOffer re-publishes a withdrawn offer on eBay.
// Implements integration.ListingActivator.
func (p *Provider) ActivateOffer(ctx context.Context, externalOfferID string) error {
	_, err := p.client.Offers.PublishOffer(ctx, externalOfferID)
	return err
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
		mo.ShippingAddress = model.ShippingAddress{
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
		"ebay_order_id":           o.OrderID,
		"ebay_legacy_order_id":    o.LegacyOrderID,
		"ebay_fulfillment_status": o.OrderFulfStatus,
		"ebay_payment_status":     o.OrderPaymentStat,
	}

	return mo
}

func getStringOr(data map[string]any, key, fallback string) string {
	if v, ok := data[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func getIntOr(data map[string]any, key string, fallback int) int {
	switch v := data[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return fallback
}

func getStringSlice(data map[string]any, key string) []string {
	if v, ok := data[key].([]string); ok {
		return v
	}
	if v, ok := data[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
