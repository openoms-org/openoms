package woocommerce

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	woocommercesdk "github.com/openoms-org/openoms/packages/woocommerce-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("woocommerce", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// WooCommerceCredentials is the JSON structure stored in encrypted integration credentials.
type WooCommerceCredentials struct {
	StoreURL       string `json:"store_url"`
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
}

// Provider implements integration.MarketplaceProvider for WooCommerce.
type Provider struct {
	client *woocommercesdk.Client
	logger *slog.Logger
}

// NewProvider creates a WooCommerce MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds WooCommerceCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("woocommerce: parse credentials: %w", err)
	}

	if creds.StoreURL == "" {
		return nil, fmt.Errorf("woocommerce: store_url is required")
	}
	if creds.ConsumerKey == "" {
		return nil, fmt.Errorf("woocommerce: consumer_key is required")
	}
	if creds.ConsumerSecret == "" {
		return nil, fmt.Errorf("woocommerce: consumer_secret is required")
	}

	client := woocommercesdk.NewClient(creds.StoreURL, creds.ConsumerKey, creds.ConsumerSecret)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "woocommerce"),
	}, nil
}

func (p *Provider) ProviderName() string { return "woocommerce" }

// PollOrders polls WooCommerce for orders modified after the given cursor.
// The cursor is the date_modified value (ISO8601 local time) of the last polled order.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := woocommercesdk.OrderListParams{
		PerPage: 50,
		OrderBy: "date",
		Order:   "asc",
	}
	if cursor != "" {
		params.ModifiedAfter = cursor
	}

	wcOrders, err := p.client.Orders.List(ctx, params)
	if err != nil {
		return nil, cursor, fmt.Errorf("woocommerce: poll orders: %w", err)
	}

	if len(wcOrders) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, wco := range wcOrders {
		mo := p.mapWooOrder(&wco)
		orders = append(orders, mo)

		// Track the latest date_modified as the new cursor
		if wco.DateModified > newCursor {
			newCursor = wco.DateModified
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from WooCommerce by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("woocommerce: invalid order ID %q: %w", externalID, err)
	}

	order, err := p.client.Orders.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("woocommerce: get order %s: %w", externalID, err)
	}

	mo := p.mapWooOrder(order)
	return &mo, nil
}

// PushOffer creates a WooCommerce product from a product and listing data.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	// Build product data from listingData, using product fields as defaults
	data := make(map[string]any)
	for k, v := range listingData {
		data[k] = v
	}
	if _, ok := data["name"]; !ok {
		data["name"] = product.Name
	}
	if _, ok := data["sku"]; !ok && product.SKU != nil {
		data["sku"] = *product.SKU
	}

	created, err := p.client.Products.Create(ctx, data)
	if err != nil {
		return "", fmt.Errorf("woocommerce: create product: %w", err)
	}
	return strconv.Itoa(created.ID), nil
}

// UpdateStock updates the stock quantity for a WooCommerce product.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	id, err := strconv.Atoi(externalOfferID)
	if err != nil {
		return fmt.Errorf("woocommerce: invalid product ID %q: %w", externalOfferID, err)
	}
	return p.client.Products.UpdateStock(ctx, id, quantity)
}

// UpdatePrice updates the price for a WooCommerce product.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	id, err := strconv.Atoi(externalOfferID)
	if err != nil {
		return fmt.Errorf("woocommerce: invalid product ID %q: %w", externalOfferID, err)
	}
	return p.client.Products.UpdatePrice(ctx, id, fmt.Sprintf("%.2f", price))
}

// mapWooOrder converts a WooCommerce SDK WooOrder to the normalized MarketplaceOrder.
func (p *Provider) mapWooOrder(o *woocommercesdk.WooOrder) integration.MarketplaceOrder {
	customerName := fmt.Sprintf("%s %s", o.Shipping.FirstName, o.Shipping.LastName)
	if customerName == " " {
		customerName = fmt.Sprintf("%s %s", o.Billing.FirstName, o.Billing.LastName)
	}

	mo := integration.MarketplaceOrder{
		ExternalID:     strconv.Itoa(o.ID),
		ExternalStatus: o.Status,
		CustomerName:   customerName,
		CustomerEmail:  o.Billing.Email,
		CustomerPhone:  o.Billing.Phone,
		ShippingAddress: model.ShippingAddress{
			Name:       fmt.Sprintf("%s %s", o.Shipping.FirstName, o.Shipping.LastName),
			Street:     o.Shipping.Address1,
			City:       o.Shipping.City,
			PostalCode: o.Shipping.PostCode,
			Country:    o.Shipping.Country,
			Phone:      o.Billing.Phone,
			Email:      o.Billing.Email,
		},
		Currency:      o.Currency,
		PaymentMethod: o.PaymentTitle,
	}

	// Billing address
	mo.BillingAddress = &model.ShippingAddress{
		Name:       fmt.Sprintf("%s %s", o.Billing.FirstName, o.Billing.LastName),
		Street:     o.Billing.Address1,
		City:       o.Billing.City,
		PostalCode: o.Billing.PostCode,
		Country:    o.Billing.Country,
		Phone:      o.Billing.Phone,
		Email:      o.Billing.Email,
	}

	// Parse total amount
	totalAmount, _ := strconv.ParseFloat(o.Total, 64)
	mo.TotalAmount = totalAmount

	// Payment status: WooCommerce order statuses that imply payment
	switch o.Status {
	case "processing", "completed", "on-hold":
		mo.PaymentStatus = "paid"
	default:
		mo.PaymentStatus = "pending"
	}

	// Parse ordered_at from date_created
	if t, err := time.Parse("2006-01-02T15:04:05", o.DateCreated); err == nil {
		mo.OrderedAt = t
	}

	// Line items
	for _, li := range o.LineItems {
		unitPrice := li.Price
		totalPrice, _ := strconv.ParseFloat(li.Total, 64)

		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: strconv.Itoa(li.ProductID),
			Name:       li.Name,
			SKU:        li.SKU,
			Quantity:   li.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		})
	}

	// RawData: store customer note and payment method code
	mo.RawData = map[string]any{
		"woocommerce_order_id": o.ID,
		"payment_method":       o.PaymentMethod,
	}
	if o.CustomerNote != "" {
		mo.RawData["customer_note"] = o.CustomerNote
	}

	return mo
}
