// Package allegro implements the Allegro marketplace provider.
package allegro

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

func init() {
	integration.RegisterMarketplaceProvider("allegro", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// Credentials is the JSON structure stored in encrypted integration credentials.
type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenExpiry  string `json:"token_expiry"` // RFC3339
	Sandbox      bool   `json:"sandbox,omitempty"`
}

var _ integration.ListingDeactivator = (*Provider)(nil)

// Provider implements integration.MarketplaceProvider for Allegro.
type Provider struct {
	client *allegrosdk.Client
	logger *slog.Logger
}

// NewProvider creates an Allegro MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, _ json.RawMessage) (*Provider, error) {
	var creds Credentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("allegro: parse credentials: %w", err)
	}

	var opts []allegrosdk.Option

	expiry, _ := time.Parse(time.RFC3339, creds.TokenExpiry)
	opts = append(opts, allegrosdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry))

	if creds.Sandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}

	opts = append(opts, allegrosdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)))

	client := allegrosdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "allegro"),
	}, nil
}

// ProviderName returns the marketplace provider identifier.
func (p *Provider) ProviderName() string { return "allegro" }

// SDKClient returns the underlying Allegro SDK client for direct API access.
func (p *Provider) SDKClient() *allegrosdk.Client {
	return p.client
}

// Close releases resources held by the underlying SDK client.
func (p *Provider) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

const checkoutFormPageSize = 100
const maxCheckoutFormPages = 50

// checkoutFormLookback is used when PollOrders has no RFC3339 cursor (manual
// SyncOrders passes ""). Allegro GET /order/checkout-forms returns 400 when the
// request has no status / boughtAt / updatedAt filter. 365d is the documented
// history window.
const checkoutFormLookback = 365 * 24 * time.Hour

// PollOrders lists Allegro checkout-forms and maps them to MarketplaceOrder rows.
// Cursor is an RFC3339 updatedAt.gte watermark. Non-timestamp leftovers (event IDs)
// are ignored so a first list after the events-poller era still backfills.
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := &allegrosdk.ListOrdersParams{Limit: checkoutFormPageSize}
	if gte, ok := parseCheckoutFormCursor(cursor); ok {
		params.UpdatedAtGte = &gte
	} else {
		lookback := time.Now().UTC().Add(-checkoutFormLookback)
		params.UpdatedAtGte = &lookback
	}

	var orders []integration.MarketplaceOrder
	newest := cursor
	var newestTime time.Time
	if params.UpdatedAtGte != nil {
		newestTime = *params.UpdatedAtGte
	}

	for page := range maxCheckoutFormPages {
		params.Offset = page * checkoutFormPageSize
		list, err := p.client.Orders.List(ctx, params)
		if err != nil {
			return nil, cursor, fmt.Errorf("allegro: list checkout-forms: %w", err)
		}
		if list == nil || len(list.CheckoutForms) == 0 {
			break
		}

		for i := range list.CheckoutForms {
			form := list.CheckoutForms[i]
			orders = append(orders, p.mapAllegroOrder(&form))
			if form.UpdatedAt.After(newestTime) {
				newestTime = form.UpdatedAt
				newest = form.UpdatedAt.UTC().Format(time.RFC3339)
			}
		}

		if len(list.CheckoutForms) < checkoutFormPageSize {
			break
		}
		if list.TotalCount > 0 && params.Offset+len(list.CheckoutForms) >= list.TotalCount {
			break
		}
	}

	return orders, newest, nil
}

func parseCheckoutFormCursor(cursor string) (time.Time, bool) {
	if cursor == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, cursor)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// GetOrder retrieves a single order from Allegro by external ID.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	order, err := p.client.Orders.Get(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("allegro: get order %s: %w", externalID, err)
	}
	mo := p.mapAllegroOrder(order)
	return &mo, nil
}

// PushOffer creates an Allegro offer from a product and listing data.
func (p *Provider) PushOffer(ctx context.Context, _ *model.Product, listingData map[string]any) (string, error) {
	offer, err := p.client.Offers.Create(ctx, listingData)
	if err != nil {
		return "", fmt.Errorf("allegro: create offer: %w", err)
	}
	return offer.ID, nil
}

// UpdateStock updates the stock quantity for an Allegro offer.
func (p *Provider) UpdateStock(ctx context.Context, externalOfferID string, quantity int) error {
	return p.client.Offers.UpdateStock(ctx, externalOfferID, quantity)
}

// UpdatePrice updates the price for an Allegro offer.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, price float64) error {
	return p.client.Offers.UpdatePrice(ctx, externalOfferID, price, "PLN")
}

// ActivateOffer implements integration.ListingActivator using Allegro publication commands.
func (p *Provider) ActivateOffer(ctx context.Context, externalOfferID string) error {
	return p.client.Offers.Activate(ctx, externalOfferID)
}

// DeactivateOffer implements integration.ListingDeactivator using Allegro publication commands.
func (p *Provider) DeactivateOffer(ctx context.Context, externalOfferID string) error {
	return p.client.Offers.Deactivate(ctx, externalOfferID)
}

// BulkUpdateStock implements integration.BulkStockUpdater using Allegro command API.
func (p *Provider) BulkUpdateStock(ctx context.Context, updates []integration.StockUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	sdkUpdates := make([]allegrosdk.StockUpdate, len(updates))
	for i, u := range updates {
		sdkUpdates[i] = allegrosdk.StockUpdate{
			OfferID:  u.ExternalOfferID,
			Quantity: u.Quantity,
		}
	}
	return p.client.Offers.BulkUpdateStock(ctx, sdkUpdates)
}

// BulkUpdatePrice implements integration.BulkPriceUpdater using Allegro command API.
func (p *Provider) BulkUpdatePrice(ctx context.Context, updates []integration.PriceUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	sdkUpdates := make([]allegrosdk.PriceUpdate, len(updates))
	for i, u := range updates {
		sdkUpdates[i] = allegrosdk.PriceUpdate{
			OfferID:  u.ExternalOfferID,
			Amount:   u.Amount,
			Currency: u.Currency,
		}
	}
	return p.client.Offers.BulkUpdatePrice(ctx, sdkUpdates)
}

// UpdateFulfillment updates the fulfillment status of an Allegro order.
func (p *Provider) UpdateFulfillment(ctx context.Context, externalOrderID, status string) error {
	if err := p.client.Fulfillment.UpdateStatus(ctx, externalOrderID, status); err != nil {
		return fmt.Errorf("allegro: update fulfillment %s: %w", externalOrderID, err)
	}
	p.logger.Info("fulfillment updated", "order_id", externalOrderID, "status", status)
	return nil
}

// AddTracking adds a shipment with tracking information to an Allegro order.
func (p *Provider) AddTracking(ctx context.Context, externalOrderID, carrierID, waybill string) error {
	shipment := allegrosdk.ShipmentInput{
		CarrierID: carrierID,
		Waybill:   waybill,
	}
	if err := p.client.Fulfillment.AddShipment(ctx, externalOrderID, shipment); err != nil {
		return fmt.Errorf("allegro: add tracking %s: %w", externalOrderID, err)
	}
	p.logger.Info("tracking added", "order_id", externalOrderID, "carrier", carrierID, "waybill", waybill)
	return nil
}

// ListCarriers returns the list of available Allegro shipping carriers.
func (p *Provider) ListCarriers(ctx context.Context) ([]allegrosdk.Carrier, error) {
	carriers, err := p.client.Fulfillment.ListCarriers(ctx)
	if err != nil {
		return nil, fmt.Errorf("allegro: list carriers: %w", err)
	}
	return carriers, nil
}

// --- Shipment Management ("Wysyłam z Allegro") ---

// ListDeliveryServices returns available delivery services for Allegro shipment management.
func (p *Provider) ListDeliveryServices(ctx context.Context) ([]allegrosdk.DeliveryService, error) {
	services, err := p.client.ShipmentManagement.ListDeliveryServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("allegro: list delivery services: %w", err)
	}
	return services, nil
}

// CreateShipment creates a managed shipment via Allegro shipment management.
func (p *Provider) CreateShipment(ctx context.Context, cmd allegrosdk.CreateShipmentCommand) (*allegrosdk.CreateShipmentResponse, error) {
	resp, err := p.client.ShipmentManagement.CreateShipment(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("allegro: create shipment: %w", err)
	}
	p.logger.Info("managed shipment created", "command_id", cmd.CommandID, "shipment_id", resp.ShipmentID)
	return resp, nil
}

// GetShipment retrieves a managed shipment by ID.
func (p *Provider) GetShipment(ctx context.Context, shipmentID string) (*allegrosdk.ManagedShipment, error) {
	shipment, err := p.client.ShipmentManagement.GetShipment(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("allegro: get shipment %s: %w", shipmentID, err)
	}
	return shipment, nil
}

// GetLabel generates a shipping label PDF for the given shipment IDs.
func (p *Provider) GetLabel(ctx context.Context, shipmentIDs []string) ([]byte, error) {
	data, err := p.client.ShipmentManagement.GetLabel(ctx, shipmentIDs)
	if err != nil {
		return nil, fmt.Errorf("allegro: get label: %w", err)
	}
	return data, nil
}

// CancelShipment cancels managed shipments by their IDs.
func (p *Provider) CancelShipment(ctx context.Context, shipmentIDs []string) error {
	if err := p.client.ShipmentManagement.CancelShipment(ctx, shipmentIDs); err != nil {
		return fmt.Errorf("allegro: cancel shipment: %w", err)
	}
	p.logger.Info("managed shipments cancelled", "shipment_ids", shipmentIDs)
	return nil
}

// GetPickupProposals retrieves pickup proposals for managed shipments.
func (p *Provider) GetPickupProposals(ctx context.Context, req allegrosdk.PickupProposalRequest) ([]allegrosdk.PickupProposal, error) {
	proposals, err := p.client.ShipmentManagement.GetPickupProposals(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("allegro: get pickup proposals: %w", err)
	}
	return proposals, nil
}

// SchedulePickup schedules a courier pickup for managed shipments.
func (p *Provider) SchedulePickup(ctx context.Context, cmd allegrosdk.SchedulePickupCommand) error {
	if err := p.client.ShipmentManagement.SchedulePickup(ctx, cmd); err != nil {
		return fmt.Errorf("allegro: schedule pickup: %w", err)
	}
	p.logger.Info("pickup scheduled", "command_id", cmd.CommandID, "date", cmd.PickupDate)
	return nil
}

// GenerateProtocol generates a dispatch protocol PDF for the given shipment IDs.
func (p *Provider) GenerateProtocol(ctx context.Context, shipmentIDs []string) ([]byte, error) {
	data, err := p.client.ShipmentManagement.GenerateProtocol(ctx, shipmentIDs)
	if err != nil {
		return nil, fmt.Errorf("allegro: generate protocol: %w", err)
	}
	return data, nil
}

// mapAllegroOrder converts an Allegro SDK Order to the normalized MarketplaceOrder.
func (p *Provider) mapAllegroOrder(o *allegrosdk.Order) integration.MarketplaceOrder {
	customerName := strings.TrimSpace(fmt.Sprintf("%s %s", o.Delivery.Address.FirstName, o.Delivery.Address.LastName))
	if customerName == "" {
		customerName = o.Buyer.Login
	}
	if customerName == "" {
		customerName = o.Buyer.Email
	}
	if customerName == "" {
		customerName = o.ID
	}

	mo := integration.MarketplaceOrder{
		ExternalID:     o.ID,
		ExternalStatus: o.Status,
		CustomerName:   customerName,
		CustomerEmail:  o.Buyer.Email,
		ShippingAddress: model.ShippingAddress{
			Name:       customerName,
			Street:     o.Delivery.Address.Street,
			City:       o.Delivery.Address.City,
			PostalCode: o.Delivery.Address.ZipCode,
			Country:    o.Delivery.Address.CountryCode,
			Phone:      o.Delivery.Address.Phone,
			Email:      o.Buyer.Email,
		},
		Currency:      "PLN",
		PaymentMethod: o.Payment.Type,
		OrderedAt:     o.UpdatedAt,
	}

	// Buyer phone
	if o.Buyer.Phone != nil {
		mo.CustomerPhone = o.Buyer.Phone.Number
	}

	// Delivery method
	if o.Delivery.Method.Name != "" {
		mo.RawData = map[string]any{
			"delivery_method_id":   o.Delivery.Method.ID,
			"delivery_method_name": o.Delivery.Method.Name,
		}
	}

	// Pickup point
	if o.Delivery.PickupPoint != nil {
		if mo.RawData == nil {
			mo.RawData = map[string]any{}
		}
		mo.RawData["pickup_point_id"] = o.Delivery.PickupPoint.ID
		mo.RawData["pickup_point_name"] = o.Delivery.PickupPoint.Name
	}

	// Payment status
	paidAmount, _ := strconv.ParseFloat(o.Payment.PaidAmount.Amount, 64)
	if paidAmount > 0 {
		mo.PaymentStatus = "paid"
	} else {
		mo.PaymentStatus = "pending"
	}
	mo.TotalAmount = paidAmount
	if o.Payment.PaidAmount.Currency != "" {
		mo.Currency = o.Payment.PaidAmount.Currency
	}

	// Line items
	for _, li := range o.LineItems {
		unitPrice, _ := strconv.ParseFloat(li.Price.Amount, 64)
		sku := ""
		if li.Offer.External != nil {
			sku = li.Offer.External.ID
		}
		mo.Items = append(mo.Items, integration.MarketplaceOrderItem{
			ExternalID: li.Offer.ID,
			Name:       li.Offer.Name,
			SKU:        sku,
			Quantity:   li.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: unitPrice * float64(li.Quantity),
		})
	}

	// If paidAmount was set, use that as total; otherwise sum line items
	if paidAmount <= 0 {
		var total float64
		for _, item := range mo.Items {
			total += item.TotalPrice
		}
		mo.TotalAmount = total
	}

	return mo
}
