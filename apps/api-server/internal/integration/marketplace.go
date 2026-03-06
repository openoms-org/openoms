package integration

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// MarketplaceOrderItem represents a single item in a marketplace order.
type MarketplaceOrderItem struct {
	ExternalID string  `json:"external_id"`
	Name       string  `json:"name"`
	SKU        string  `json:"sku,omitempty"`
	EAN        string  `json:"ean,omitempty"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
	TaxRate    float64 `json:"tax_rate,omitempty"`
}

// MarketplaceOrder represents an order retrieved from a marketplace provider.
type MarketplaceOrder struct {
	ExternalID      string                 `json:"external_id"`
	ExternalStatus  string                 `json:"external_status"`
	CustomerName    string                 `json:"customer_name"`
	CustomerEmail   string                 `json:"customer_email,omitempty"`
	CustomerPhone   string                 `json:"customer_phone,omitempty"`
	ShippingAddress model.ShippingAddress  `json:"shipping_address"`
	BillingAddress  *model.ShippingAddress `json:"billing_address,omitempty"`
	Items           []MarketplaceOrderItem `json:"items"`
	TotalAmount     float64                `json:"total_amount"`
	Currency        string                 `json:"currency"`
	PaymentStatus   string                 `json:"payment_status,omitempty"`
	PaymentMethod   string                 `json:"payment_method,omitempty"`
	OrderedAt       time.Time              `json:"ordered_at"`
	RawData         map[string]any         `json:"raw_data,omitempty"`
}

// MarketplaceProvider defines the interface for marketplace integrations
// (e.g. Allegro, WooCommerce).
type MarketplaceProvider interface {
	ProviderName() string
	PollOrders(ctx context.Context, cursor string) (orders []MarketplaceOrder, newCursor string, err error)
	GetOrder(ctx context.Context, externalID string) (*MarketplaceOrder, error)
	PushOffer(ctx context.Context, product *model.Product, listingData map[string]any) (externalID string, err error)
	UpdateStock(ctx context.Context, externalOfferID string, quantity int) error
	UpdatePrice(ctx context.Context, externalOfferID string, price float64) error
}

// ListingActivator is optionally implemented by providers that support activating/relisting
// marketplace offers (e.g. Allegro offer publication commands).
type ListingActivator interface {
	ActivateOffer(ctx context.Context, externalOfferID string) error
}

// ListingDeactivator is optionally implemented by providers that support deactivating/ending
// marketplace offers when stock reaches zero.
type ListingDeactivator interface {
	DeactivateOffer(ctx context.Context, externalOfferID string) error
}

// AsyncStockUpdater is a marker interface implemented by providers whose stock updates
// are asynchronous (e.g., Amazon Feeds API submits a feed that takes minutes to process).
// Callers should set sync_status='pending' instead of 'synced' after a successful update.
type AsyncStockUpdater interface {
	IsAsyncStockUpdate()
}

// AsyncPriceUpdater is a marker interface for providers whose price updates are asynchronous.
type AsyncPriceUpdater interface {
	IsAsyncPriceUpdate()
}

// FeedSubmission holds the result of an async feed submission.
type FeedSubmission struct {
	FeedID   string
	FeedType string // "inventory" or "pricing"
}

// AsyncFeedResult is implemented by providers that return feed IDs from async operations.
// After a successful UpdateStock/BulkUpdateStock/UpdatePrice call, the caller can
// retrieve the feed ID for status polling.
type AsyncFeedResult interface {
	FeedResult() *FeedSubmission
}

// BulkStockUpdater is optionally implemented by providers that support batch stock updates.
type BulkStockUpdater interface {
	BulkUpdateStock(ctx context.Context, updates []StockUpdate) error
}

// BulkPriceUpdater is optionally implemented by providers that support batch price updates.
type BulkPriceUpdater interface {
	BulkUpdatePrice(ctx context.Context, updates []PriceUpdate) error
}

// StockUpdate represents a stock change for one offer.
type StockUpdate struct {
	ExternalOfferID string
	Quantity        int
}

// PriceUpdate represents a price change for one offer.
type PriceUpdate struct {
	ExternalOfferID string
	Amount          float64
	Currency        string
}

// MarketplaceOrderToCreateRequest converts a MarketplaceOrder to a model.CreateOrderRequest.
func MarketplaceOrderToCreateRequest(mo MarketplaceOrder, source string, integrationID uuid.UUID) model.CreateOrderRequest {
	req := model.CreateOrderRequest{
		ExternalID:    &mo.ExternalID,
		Source:        source,
		IntegrationID: &integrationID,
		CustomerName:  mo.CustomerName,
		TotalAmount:   mo.TotalAmount,
		Currency:      mo.Currency,
		OrderedAt:     &mo.OrderedAt,
	}
	if mo.CustomerEmail != "" {
		req.CustomerEmail = &mo.CustomerEmail
	}
	if mo.CustomerPhone != "" {
		req.CustomerPhone = &mo.CustomerPhone
	}
	if mo.PaymentStatus != "" {
		req.PaymentStatus = &mo.PaymentStatus
	}
	if mo.PaymentMethod != "" {
		req.PaymentMethod = &mo.PaymentMethod
	}
	return req
}
