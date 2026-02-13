package olx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func init() {
	integration.RegisterMarketplaceProvider("olx", func(credentials json.RawMessage, settings json.RawMessage) (integration.MarketplaceProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// OLXCredentials is the JSON structure stored in encrypted integration credentials.
type OLXCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	Sandbox      bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for OLX.
type Provider struct {
	client *olxsdk.Client
	logger *slog.Logger
}

// NewProvider creates an OLX MarketplaceProvider from encrypted credentials.
func NewProvider(credentials json.RawMessage, settings json.RawMessage) (*Provider, error) {
	var creds OLXCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("olx: parse credentials: %w", err)
	}

	if creds.ClientID == "" {
		return nil, fmt.Errorf("olx: client_id is required")
	}
	if creds.ClientSecret == "" {
		return nil, fmt.Errorf("olx: client_secret is required")
	}

	client := olxsdk.NewClient(creds.ClientID, creds.ClientSecret, creds.AccessToken)

	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "olx"),
	}, nil
}

func (p *Provider) ProviderName() string { return "olx" }

// PollOrders polls OLX transactions (since OLX is classifieds, transactions map to orders).
func (p *Provider) PollOrders(ctx context.Context, cursor string) ([]integration.MarketplaceOrder, string, error) {
	params := olxsdk.TransactionListParams{
		Limit: 50,
	}
	if cursor != "" {
		params.CreatedAfter = cursor
	}

	resp, err := p.client.Transactions.ListTransactions(ctx, params)
	if err != nil {
		return nil, cursor, fmt.Errorf("olx: poll orders: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, cursor, nil
	}

	var orders []integration.MarketplaceOrder
	newCursor := cursor

	for _, tx := range resp.Data {
		mo := p.mapOLXTransaction(&tx)
		orders = append(orders, mo)

		if tx.CreatedAt > newCursor {
			newCursor = tx.CreatedAt
		}
	}

	return orders, newCursor, nil
}

// GetOrder retrieves a single OLX transaction. Since OLX transactions don't have a direct
// single-get endpoint, we search by ID in the transaction list.
func (p *Provider) GetOrder(ctx context.Context, externalID string) (*integration.MarketplaceOrder, error) {
	// OLX Partner API does not have a direct get-transaction-by-id endpoint.
	// We poll recent transactions and search for the matching one.
	resp, err := p.client.Transactions.ListTransactions(ctx, olxsdk.TransactionListParams{
		Limit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("olx: get order %s: %w", externalID, err)
	}

	for _, tx := range resp.Data {
		if tx.ID == externalID {
			mo := p.mapOLXTransaction(&tx)
			return &mo, nil
		}
	}

	return nil, fmt.Errorf("olx: order %s not found", externalID)
}

// PushOffer creates an OLX advert from a product.
func (p *Provider) PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (string, error) {
	return "", fmt.Errorf("olx: PushOffer not yet implemented")
}

// UpdateStock is not applicable for OLX classifieds.
func (p *Provider) UpdateStock(_ context.Context, _ string, _ int) error {
	return fmt.Errorf("olx: UpdateStock not applicable for classifieds")
}

// UpdatePrice is not yet supported for OLX.
func (p *Provider) UpdatePrice(_ context.Context, _ string, _ float64) error {
	return fmt.Errorf("olx: UpdatePrice not yet implemented")
}

// mapOLXTransaction converts an OLX transaction to the normalized MarketplaceOrder.
func (p *Provider) mapOLXTransaction(tx *olxsdk.Transaction) integration.MarketplaceOrder {
	mo := integration.MarketplaceOrder{
		ExternalID:     tx.ID,
		ExternalStatus: tx.Status,
		CustomerName:   tx.BuyerName,
		CustomerEmail:  tx.BuyerEmail,
		CustomerPhone:  tx.BuyerPhone,
		TotalAmount:    tx.Amount,
		Currency:       tx.Currency,
	}

	// Map payment status
	switch tx.Status {
	case "completed", "paid":
		mo.PaymentStatus = "paid"
	default:
		mo.PaymentStatus = "pending"
	}

	// Parse ordered_at
	if t, err := time.Parse(time.RFC3339, tx.CreatedAt); err == nil {
		mo.OrderedAt = t
	}

	// Shipping address
	if tx.ShippingAddr != nil {
		mo.ShippingAddress = model.ShippingAddress{
			Name:       tx.ShippingAddr.Name,
			Street:     tx.ShippingAddr.Street,
			City:       tx.ShippingAddr.City,
			PostalCode: tx.ShippingAddr.PostalCode,
			Country:    tx.ShippingAddr.Country,
			Phone:      tx.ShippingAddr.Phone,
			Email:      tx.BuyerEmail,
		}
	} else {
		mo.ShippingAddress = model.ShippingAddress{
			Name:  tx.BuyerName,
			Email: tx.BuyerEmail,
			Phone: tx.BuyerPhone,
		}
	}

	// Single line item from the transaction
	quantity := tx.Quantity
	if quantity == 0 {
		quantity = 1
	}
	mo.Items = []integration.MarketplaceOrderItem{
		{
			ExternalID: strconv.FormatInt(tx.AdvertID, 10),
			Name:       tx.AdvertTitle,
			Quantity:   quantity,
			UnitPrice:  tx.Amount / float64(quantity),
			TotalPrice: tx.Amount,
		},
	}

	// RawData
	mo.RawData = map[string]any{
		"olx_transaction_id": tx.ID,
		"olx_advert_id":      tx.AdvertID,
		"olx_status":          tx.Status,
	}

	return mo
}
