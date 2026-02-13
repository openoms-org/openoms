package kaufland

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	kauflandsdk "github.com/openoms-org/openoms/packages/kaufland-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("kaufland", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// KauflandCredentials is the JSON structure stored in encrypted integration credentials.
type KauflandCredentials struct {
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
	Sandbox   bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for Kaufland.
type Provider struct {
	client *kauflandsdk.Client
	logger *slog.Logger
}

// NewProvider creates a Kaufland MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds KauflandCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("kaufland: parse credentials: %w", err)
	}

	if creds.APIKey == "" {
		return nil, fmt.Errorf("kaufland: api_key is required")
	}
	if creds.SecretKey == "" {
		return nil, fmt.Errorf("kaufland: secret_key is required")
	}

	var opts []kauflandsdk.Option
	if creds.Sandbox {
		opts = append(opts, kauflandsdk.WithSandbox())
	}

	client := kauflandsdk.NewClient(creds.APIKey, creds.SecretKey, opts...)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "kaufland"),
	}, nil
}

func (p *Provider) ProviderName() string { return "kaufland" }

// PollOrders polls Kaufland for order units created after the given cursor (ISO8601 timestamp).
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := kauflandsdk.OrderUnitListParams{
		Limit: 50,
		Sort:  "ts_created_iso:asc",
	}
	if cursor != "" {
		params.CreatedAfter = cursor
	}

	resp, err := p.client.Orders.GetOrderUnits(ctx, params)
	if err != nil {
		return nil, cursor, fmt.Errorf("kaufland: poll orders: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, cursor, nil
	}

	// Group order units by order ID to build aggregate orders
	orderMap := make(map[int64][]kauflandsdk.OrderUnit)
	for _, ou := range resp.Data {
		orderMap[ou.IDOrder] = append(orderMap[ou.IDOrder], ou)
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for orderID, units := range orderMap {
		mo := p.mapKauflandOrder(orderID, units)
		orders = append(orders, mo)

		for _, u := range units {
			if u.CreatedAt > newCursor {
				newCursor = u.CreatedAt
			}
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single order unit from Kaufland by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	id, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("kaufland: invalid order unit ID %q: %w", externalID, err)
	}

	unit, err := p.client.Orders.GetOrderUnit(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("kaufland: get order unit %s: %w", externalID, err)
	}

	mo := p.mapKauflandOrder(unit.IDOrder, []kauflandsdk.OrderUnit{*unit})
	return &mo, nil
}

// PushOffer is not yet supported for Kaufland.
func (p *Provider) PushOffer(_ context.Context, _ *model.Product, _ map[string]any) (string, error) {
	return "", fmt.Errorf("kaufland: PushOffer not yet implemented")
}

// UpdateStock is not yet supported for Kaufland.
func (p *Provider) UpdateStock(_ context.Context, _ string, _ int) error {
	return fmt.Errorf("kaufland: UpdateStock not yet implemented")
}

// UpdatePrice is not yet supported for Kaufland.
func (p *Provider) UpdatePrice(_ context.Context, _ string, _ float64) error {
	return fmt.Errorf("kaufland: UpdatePrice not yet implemented")
}

// mapKauflandOrder converts Kaufland order units to the normalized MarketplaceOrder.
func (p *Provider) mapKauflandOrder(orderID int64, units []kauflandsdk.OrderUnit) integration.MarketplaceOrder {
	// Use the first unit for order-level details
	first := units[0]

	customerName := fmt.Sprintf("%s %s", first.ShippingAddr.FirstName, first.ShippingAddr.LastName)

	street := first.ShippingAddr.Street
	if first.ShippingAddr.HouseNumber != "" {
		street += " " + first.ShippingAddr.HouseNumber
	}

	mo := integration.MarketplaceOrder{
		ExternalID:     strconv.FormatInt(orderID, 10),
		ExternalStatus: first.Status,
		CustomerName:   customerName,
		CustomerEmail:  first.Buyer.Email,
		ShippingAddress: model.ShippingAddress{
			Name:       customerName,
			Street:     street,
			City:       first.ShippingAddr.City,
			PostalCode: first.ShippingAddr.PostalCode,
			Country:    first.ShippingAddr.Country,
			Phone:      first.ShippingAddr.Phone,
			Email:      first.Buyer.Email,
		},
		Currency: first.Currency,
	}

	// Billing address
	billingName := fmt.Sprintf("%s %s", first.BillingAddr.FirstName, first.BillingAddr.LastName)
	billingStreet := first.BillingAddr.Street
	if first.BillingAddr.HouseNumber != "" {
		billingStreet += " " + first.BillingAddr.HouseNumber
	}
	mo.BillingAddress = &model.ShippingAddress{
		Name:       billingName,
		Street:     billingStreet,
		City:       first.BillingAddr.City,
		PostalCode: first.BillingAddr.PostalCode,
		Country:    first.BillingAddr.Country,
		Phone:      first.BillingAddr.Phone,
		Email:      first.Buyer.Email,
	}

	// Parse ordered_at from first unit's created date
	if t, err := time.Parse(time.RFC3339, first.CreatedAt); err == nil {
		mo.OrderedAt = t
	}

	// Map payment status based on Kaufland order unit status
	switch first.Status {
	case "received", "sent", "returned":
		mo.PaymentStatus = "paid"
	default:
		mo.PaymentStatus = "pending"
	}

	// Line items from all units
	var total float64
	for _, u := range units {
		totalPrice := u.Price * float64(u.Quantity)
		total += totalPrice

		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: strconv.FormatInt(u.IDOrderUnit, 10),
			Name:       u.Item.Title,
			EAN:        u.Item.EAN,
			Quantity:   u.Quantity,
			UnitPrice:  u.Price,
			TotalPrice: totalPrice,
		})
	}
	mo.TotalAmount = total

	// RawData
	mo.RawData = map[string]any{
		"kaufland_order_id": orderID,
		"kaufland_status":   first.Status,
	}

	return mo
}
