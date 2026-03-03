package prestashop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"time"

	prestashopsdk "github.com/openoms-org/openoms/packages/prestashop-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

func init() {
	integration.RegisterMarketplaceProvider("prestashop", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// PrestaShopCredentials is the JSON structure stored in encrypted integration credentials.
type PrestaShopCredentials struct {
	ShopURL string `json:"shop_url"`
	APIKey  string `json:"api_key"`
}

// Provider implements integration.MarketplaceProvider for PrestaShop.
type Provider struct {
	client *prestashopsdk.Client
	logger *slog.Logger
}

// NewProvider creates a PrestaShop MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds PrestaShopCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("prestashop: parse credentials: %w", err)
	}

	if creds.ShopURL == "" {
		return nil, fmt.Errorf("prestashop: shop_url is required")
	}
	if creds.APIKey == "" {
		return nil, fmt.Errorf("prestashop: api_key is required")
	}

	client := prestashopsdk.NewClient(creds.ShopURL, creds.APIKey,
		prestashopsdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)),
	)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "prestashop"),
	}, nil
}

func (p *Provider) ProviderName() string { return "prestashop" }

// PollOrders polls PrestaShop for orders updated after the given cursor.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := prestashopsdk.OrderListParams{
		Limit:  50,
		SortBy: "date_upd_ASC",
	}
	if cursor != "" {
		params.DateUpdFrom = cursor
	}

	psOrders, err := p.client.Orders.List(ctx, params)
	if err != nil {
		return nil, cursor, fmt.Errorf("prestashop: poll orders: %w", err)
	}

	if len(psOrders) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, pso := range psOrders {
		mo, err := p.mapPrestaShopOrder(ctx, &pso)
		if err != nil {
			p.logger.Error("failed to map PrestaShop order", "order_id", pso.ID, "error", err)
			continue
		}
		orders = append(orders, *mo)

		if pso.DateUpd > newCursor {
			newCursor = pso.DateUpd
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order from PrestaShop by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("prestashop: invalid order ID %q: %w", externalID, err)
	}

	order, err := p.client.Orders.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("prestashop: get order %s: %w", externalID, err)
	}

	return p.mapPrestaShopOrder(ctx, order)
}

// PushOffer creates a PrestaShop product from a product and listing data.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	data := make(map[string]any)
	maps.Copy(data, listingData)
	if _, ok := data["reference"]; !ok && product.SKU != nil {
		data["reference"] = *product.SKU
	}

	// PrestaShop product creation requires specific XML/JSON structure.
	// Return a placeholder external ID; full listing support TBD.
	return fmt.Sprintf("prestashop-%s", product.ID.String()), nil
}

// UpdateStock updates the stock quantity for a PrestaShop product.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	productID, err := strconv.Atoi(externalOfferID)
	if err != nil {
		return fmt.Errorf("prestashop: invalid product ID %q: %w", externalOfferID, err)
	}

	stock, err := p.client.Stock.GetByProduct(ctx, productID)
	if err != nil {
		return fmt.Errorf("prestashop: get stock for product %d: %w", productID, err)
	}

	return p.client.Stock.UpdateQuantity(ctx, stock.ID, quantity)
}

// UpdatePrice updates the price for a PrestaShop product.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	id, err := strconv.Atoi(externalOfferID)
	if err != nil {
		return fmt.Errorf("prestashop: invalid product ID %q: %w", externalOfferID, err)
	}
	return p.client.Products.UpdatePrice(ctx, id, fmt.Sprintf("%.6f", price))
}

// mapPrestaShopOrder converts a PrestaShop order to a MarketplaceOrder.
// It makes additional API calls to fetch address and customer details.
func (p *Provider) mapPrestaShopOrder(ctx context.Context, o *prestashopsdk.PSOrder) (*integration.MarketplaceOrder, error) {
	// Fetch customer details
	customer, err := p.client.Orders.GetCustomer(ctx, o.IDCustomer)
	if err != nil {
		return nil, fmt.Errorf("prestashop: get customer %d: %w", o.IDCustomer, err)
	}

	// Fetch delivery address
	deliveryAddr, err := p.client.Orders.GetAddress(ctx, o.IDAddressDelivery)
	if err != nil {
		return nil, fmt.Errorf("prestashop: get delivery address %d: %w", o.IDAddressDelivery, err)
	}

	// Fetch order details (line items)
	orderDetails, err := p.client.Products.ListOrderDetails(ctx, o.ID)
	if err != nil {
		p.logger.Warn("failed to fetch order details", "order_id", o.ID, "error", err)
		// Continue without line items
	}

	customerName := fmt.Sprintf("%s %s", deliveryAddr.FirstName, deliveryAddr.LastName)
	phone := deliveryAddr.Phone
	if phone == "" {
		phone = deliveryAddr.PhoneMobile
	}

	totalAmount, _ := strconv.ParseFloat(o.TotalPaidTaxIncl, 64)

	mo := integration.MarketplaceOrder{
		ExternalID:     strconv.Itoa(o.ID),
		ExternalStatus: strconv.Itoa(o.CurrentState),
		CustomerName:   customerName,
		CustomerEmail:  customer.Email,
		CustomerPhone:  phone,
		ShippingAddress: model.ShippingAddress{
			Name:       customerName,
			Street:     deliveryAddr.Address1,
			City:       deliveryAddr.City,
			PostalCode: deliveryAddr.PostCode,
			Country:    strconv.Itoa(deliveryAddr.IDCountry),
			Phone:      phone,
			Email:      customer.Email,
		},
		TotalAmount:   totalAmount,
		Currency:      strconv.Itoa(o.IDCurrency),
		PaymentMethod: o.Payment,
	}

	// Billing address (if different)
	if o.IDAddressInvoice != o.IDAddressDelivery {
		invoiceAddr, err := p.client.Orders.GetAddress(ctx, o.IDAddressInvoice)
		if err == nil {
			invoicePhone := invoiceAddr.Phone
			if invoicePhone == "" {
				invoicePhone = invoiceAddr.PhoneMobile
			}
			mo.BillingAddress = &model.ShippingAddress{
				Name:       fmt.Sprintf("%s %s", invoiceAddr.FirstName, invoiceAddr.LastName),
				Street:     invoiceAddr.Address1,
				City:       invoiceAddr.City,
				PostalCode: invoiceAddr.PostCode,
				Country:    strconv.Itoa(invoiceAddr.IDCountry),
				Phone:      invoicePhone,
				Email:      customer.Email,
			}
		}
	}

	// Payment status: state IDs 2 (payment accepted) and 4+ generally mean paid
	// This is a simplification; real mapping depends on shop configuration
	switch o.CurrentState {
	case 2, 3, 4, 5:
		mo.PaymentStatus = "paid"
	default:
		mo.PaymentStatus = "pending"
	}

	// Parse ordered_at
	if t, err := time.Parse("2006-01-02 15:04:05", o.DateAdd); err == nil {
		mo.OrderedAt = t
	}

	// Line items
	for _, detail := range orderDetails {
		unitPrice, _ := strconv.ParseFloat(detail.UnitPrice, 64)
		totalPrice, _ := strconv.ParseFloat(detail.TotalPrice, 64)

		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: strconv.Itoa(detail.ProductID),
			Name:       detail.ProductName,
			SKU:        detail.ProductRef,
			EAN:        detail.ProductEAN,
			Quantity:   detail.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		})
	}

	// RawData
	mo.RawData = map[string]any{
		"prestashop_order_id": o.ID,
		"reference":           o.Reference,
		"payment_module":      o.Module,
	}

	return &mo, nil
}
