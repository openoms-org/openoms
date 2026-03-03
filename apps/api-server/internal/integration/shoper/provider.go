package shoper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"time"

	shopersdk "github.com/openoms-org/openoms/packages/shoper-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

func init() {
	integration.RegisterMarketplaceProvider("shoper", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// ShoperCredentials is the JSON structure stored in encrypted integration credentials.
type ShoperCredentials struct {
	ShopURL      string `json:"shop_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Provider implements integration.MarketplaceProvider for Shoper.
type Provider struct {
	client *shopersdk.Client
	logger *slog.Logger
}

// NewProvider creates a Shoper MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds ShoperCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("shoper: parse credentials: %w", err)
	}

	if creds.ShopURL == "" {
		return nil, fmt.Errorf("shoper: shop_url is required")
	}
	if creds.ClientID == "" {
		return nil, fmt.Errorf("shoper: client_id is required")
	}
	if creds.ClientSecret == "" {
		return nil, fmt.Errorf("shoper: client_secret is required")
	}

	client := shopersdk.NewClient(creds.ShopURL, creds.ClientID, creds.ClientSecret,
		shopersdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)),
	)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "shoper"),
	}, nil
}

func (p *Provider) ProviderName() string { return "shoper" }

// PollOrders polls Shoper for orders modified after the given cursor.
// The cursor is the date_modify value of the last polled order.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := shopersdk.OrderListParams{
		Limit:         50,
		SortBy:        "date_modify",
		SortDirection: "asc",
	}
	if cursor != "" {
		params.ModifiedAfter = cursor
	}

	resp, err := p.client.Orders.List(ctx, params)
	if err != nil {
		return nil, cursor, fmt.Errorf("shoper: poll orders: %w", err)
	}

	if len(resp.List) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, so := range resp.List {
		mo := p.mapShoperOrder(&so)
		orders = append(orders, mo)

		if so.DateModify > newCursor {
			newCursor = so.DateModify
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from Shoper by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("shoper: invalid order ID %q: %w", externalID, err)
	}

	order, err := p.client.Orders.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("shoper: get order %s: %w", externalID, err)
	}

	mo := p.mapShoperOrder(order)
	return &mo, nil
}

// PushOffer creates a Shoper product from a product and listing data.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	data := make(map[string]any)
	maps.Copy(data, listingData)
	if _, ok := data["code"]; !ok && product.SKU != nil {
		data["code"] = *product.SKU
	}

	// Shoper does not have a simple product create via REST that returns the ID directly.
	// For now we return the SKU as external reference; full listing support TBD.
	return fmt.Sprintf("shoper-%s", product.ID.String()), nil
}

// UpdateStock updates the stock quantity for a Shoper product.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	id, err := strconv.Atoi(externalOfferID)
	if err != nil {
		return fmt.Errorf("shoper: invalid product ID %q: %w", externalOfferID, err)
	}
	return p.client.Products.UpdateStock(ctx, id, quantity)
}

// UpdatePrice updates the price for a Shoper product.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	id, err := strconv.Atoi(externalOfferID)
	if err != nil {
		return fmt.Errorf("shoper: invalid product ID %q: %w", externalOfferID, err)
	}
	return p.client.Products.Update(ctx, id, map[string]any{
		"stock": map[string]any{
			"price": price,
		},
	})
}

// mapShoperOrder converts a Shoper SDK ShoperOrder to the normalized MarketplaceOrder.
func (p *Provider) mapShoperOrder(o *shopersdk.ShoperOrder) integration.MarketplaceOrder {
	customerName := fmt.Sprintf("%s %s", o.DeliveryAddress.FirstName, o.DeliveryAddress.LastName)
	if customerName == " " {
		customerName = fmt.Sprintf("%s %s", o.BillingAddress.FirstName, o.BillingAddress.LastName)
	}

	mo := integration.MarketplaceOrder{
		ExternalID:     strconv.Itoa(o.ID),
		ExternalStatus: strconv.Itoa(o.StatusID),
		CustomerName:   customerName,
		CustomerEmail:  o.Email,
		CustomerPhone:  o.Phone,
		ShippingAddress: model.ShippingAddress{
			Name:       fmt.Sprintf("%s %s", o.DeliveryAddress.FirstName, o.DeliveryAddress.LastName),
			Street:     o.DeliveryAddress.Street1,
			City:       o.DeliveryAddress.City,
			PostalCode: o.DeliveryAddress.PostCode,
			Country:    o.DeliveryAddress.Country,
			Phone:      o.DeliveryAddress.Phone,
			Email:      o.Email,
		},
		Currency: o.Currency,
	}

	// Billing address
	mo.BillingAddress = &model.ShippingAddress{
		Name:       fmt.Sprintf("%s %s", o.BillingAddress.FirstName, o.BillingAddress.LastName),
		Street:     o.BillingAddress.Street1,
		City:       o.BillingAddress.City,
		PostalCode: o.BillingAddress.PostCode,
		Country:    o.BillingAddress.Country,
		Phone:      o.BillingAddress.Phone,
		Email:      o.Email,
	}

	mo.TotalAmount = o.Sum

	// Payment status based on paid amount
	if o.Paid >= o.Sum && o.Sum > 0 {
		mo.PaymentStatus = "paid"
	} else {
		mo.PaymentStatus = "pending"
	}

	// Parse ordered_at from date_add
	if t, err := time.Parse("2006-01-02 15:04:05", o.DateAdd); err == nil {
		mo.OrderedAt = t
	}

	// Line items
	for _, li := range o.Products {
		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: strconv.Itoa(li.ProductID),
			Name:       li.Name,
			SKU:        li.Code,
			Quantity:   li.Quantity,
			UnitPrice:  li.Price,
			TotalPrice: li.Price * float64(li.Quantity),
		})
	}

	// RawData
	mo.RawData = map[string]any{
		"shoper_order_id": o.ID,
		"order_serial":    o.OrderSerial,
		"shipping_id":     o.ShippingID,
		"payment_id":      o.PaymentID,
	}
	if o.Notes != "" {
		mo.RawData["customer_note"] = o.Notes
	}

	return mo
}
