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
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
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
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenExpiry  string `json:"token_expiry,omitempty"`
	Sandbox      bool   `json:"sandbox,omitempty"`
}

// Provider implements integration.MarketplaceProvider for OLX.
type Provider struct {
	client *olxsdk.Client
	logger *slog.Logger
}

var _ integration.ListingActivator = (*Provider)(nil)

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
	if creds.Sandbox {
		return nil, fmt.Errorf("olx: sandbox mode is not supported — OLX does not provide a sandbox environment")
	}

	opts := []olxsdk.Option{
		olxsdk.WithHTTPClient(netutil.SafeHTTPClient(30 * time.Second)),
	}

	// If OAuth tokens are available (from authorization code flow), use them.
	if creds.RefreshToken != "" && creds.TokenExpiry != "" {
		expiry, _ := time.Parse(time.RFC3339, creds.TokenExpiry)
		opts = append(opts, olxsdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry))
	}

	client := olxsdk.NewClient(creds.ClientID, creds.ClientSecret, creds.AccessToken, opts...)

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
	req := olxsdk.CreateAdvertRequest{
		Title:          product.Name,
		Description:    model.StripHTMLTags(product.DescriptionLong),
		AdvertiserType: "business",
		Price:          &olxsdk.AdvertPrice{Value: product.Price, Currency: "PLN"},
	}

	// Extract image URLs from product.Images JSON array of objects with "url" key.
	if product.Images != nil {
		var imgs []map[string]any
		if err := json.Unmarshal(product.Images, &imgs); err == nil {
			for _, img := range imgs {
				if u, ok := img["url"].(string); ok && u != "" {
					req.Images = append(req.Images, olxsdk.AdvertImage{URL: u})
				}
			}
		}
	}

	// Set external ID from SKU if available.
	if product.SKU != nil && *product.SKU != "" {
		req.ExternalID = *product.SKU
	}

	// Apply listingData overrides.
	if v, ok := listingData["title"].(string); ok {
		req.Title = v
	}
	if v, ok := listingData["description"].(string); ok {
		req.Description = model.StripHTMLTags(v)
	}
	if v, ok := listingData["price"].(float64); ok {
		if req.Price == nil {
			req.Price = &olxsdk.AdvertPrice{Currency: "PLN"}
		}
		req.Price.Value = v
	}
	if v, ok := listingData["currency"].(string); ok {
		if req.Price == nil {
			req.Price = &olxsdk.AdvertPrice{}
		}
		req.Price.Currency = v
	}
	if v, ok := listingData["category_id"].(float64); ok {
		req.CategoryID = int(v)
	}
	if v, ok := listingData["city_id"].(float64); ok {
		req.Location = &olxsdk.Location{CityID: int(v)}
	}
	if v, ok := listingData["contact_name"].(string); ok {
		contact := &olxsdk.Contact{Name: v}
		if phone, ok := listingData["contact_phone"].(string); ok {
			contact.Phone = phone
		}
		req.Contact = contact
	}

	// Validate price is positive.
	if req.Price != nil && req.Price.Value <= 0 {
		return "", fmt.Errorf("olx: price must be positive")
	}

	// Truncate to OLX API limits (70 chars title, 9000 chars description).
	if titleRunes := []rune(req.Title); len(titleRunes) > 70 {
		req.Title = string(titleRunes[:70])
	}
	if descRunes := []rune(req.Description); len(descRunes) > 9000 {
		req.Description = string(descRunes[:9000])
	}

	// Validate required fields.
	if req.CategoryID <= 0 {
		return "", fmt.Errorf("olx: category_id is required in listingData")
	}
	if req.Location == nil || req.Location.CityID <= 0 {
		return "", fmt.Errorf("olx: city_id is required in listingData")
	}
	if req.Contact == nil || req.Contact.Name == "" {
		return "", fmt.Errorf("olx: contact_name is required in listingData")
	}

	advert, err := p.client.Adverts.CreateAdvert(ctx, req)
	if err != nil {
		return "", fmt.Errorf("olx: create advert: %w", err)
	}

	return strconv.FormatInt(advert.ID, 10), nil
}

// UpdateStock is not applicable for OLX classifieds.
func (p *Provider) UpdateStock(_ context.Context, _ string, _ int) error {
	return fmt.Errorf("olx: UpdateStock not applicable for classifieds")
}

// UpdatePrice updates the price of an existing OLX advert.
// OLX PUT requires the full advert body, so we fetch-then-update.
func (p *Provider) UpdatePrice(ctx context.Context, externalOfferID string, newPrice float64) error {
	advertID, err := strconv.ParseInt(externalOfferID, 10, 64)
	if err != nil {
		return fmt.Errorf("olx: invalid advert ID %q: %w", externalOfferID, err)
	}

	advert, err := p.client.Adverts.GetAdvert(ctx, advertID)
	if err != nil {
		return fmt.Errorf("olx: get advert %d: %w", advertID, err)
	}

	req := p.advertToUpdateRequest(advert)
	if req.Price == nil {
		req.Price = &olxsdk.AdvertPrice{Currency: "PLN"}
	}
	req.Price.Value = newPrice

	_, err = p.client.Adverts.UpdateAdvert(ctx, advertID, req)
	if err != nil {
		return fmt.Errorf("olx: update price for advert %d: %w", advertID, err)
	}
	return nil
}

// advertToUpdateRequest converts a fetched Advert to a CreateAdvertRequest suitable for PUT.
func (p *Provider) advertToUpdateRequest(a *olxsdk.Advert) olxsdk.CreateAdvertRequest {
	req := olxsdk.CreateAdvertRequest{
		Title:       a.Title,
		Description: a.Description,
		ExternalID:  a.ExternalID,
	}
	if a.Price != nil {
		req.Price = &olxsdk.AdvertPrice{
			Value:      a.Price.Value,
			Currency:   a.Price.Currency,
			Negotiable: a.Price.Negotiable,
		}
	}
	if a.Category != nil {
		req.CategoryID = a.Category.ID
	}
	if a.Contact != nil {
		req.Contact = a.Contact
	}
	for _, param := range a.Params {
		req.Attributes = append(req.Attributes, olxsdk.AdvertAttribute{
			Code:  param.Key,
			Value: param.Value,
		})
	}
	return req
}

// ActivateOffer activates (or re-lists) an OLX advert.
func (p *Provider) ActivateOffer(ctx context.Context, externalOfferID string) error {
	advertID, err := strconv.ParseInt(externalOfferID, 10, 64)
	if err != nil {
		return fmt.Errorf("olx: invalid advert ID %q: %w", externalOfferID, err)
	}
	return p.client.Adverts.RunCommand(ctx, advertID, olxsdk.AdvertCommandRequest{
		Command: "activate",
	})
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
		"olx_status":         tx.Status,
	}

	if omsStatus, ok := olxsdk.MapTransactionStatus(tx.Status); ok {
		mo.RawData["oms_status"] = omsStatus
	} else {
		p.logger.Warn("olx: unknown transaction status", "status", tx.Status, "transaction_id", tx.ID)
	}

	return mo
}
