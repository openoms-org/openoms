package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// SupplierProduct represents a normalized product from a supplier feed or API.
type SupplierProduct struct {
	ExternalID    string            `json:"external_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	EAN           string            `json:"ean,omitempty"`
	SKU           string            `json:"sku,omitempty"`
	Price         float64           `json:"price"`
	RetailPrice   float64           `json:"retail_price,omitempty"`
	StockQuantity int               `json:"stock_quantity"`
	Category      string            `json:"category,omitempty"`
	Brand         string            `json:"brand,omitempty"`
	ImageURL      string            `json:"image_url,omitempty"`
	Images        []string          `json:"images,omitempty"`
	Weight        float64           `json:"weight,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// SupplierOrderRequest contains data needed to place an order with a supplier.
type SupplierOrderRequest struct {
	Lines             []SupplierOrderLine `json:"lines"`
	ClientOrderNumber string              `json:"client_order_number,omitempty"`
	DeliveryPointID   string              `json:"delivery_point_id,omitempty"`
	CarrierID         string              `json:"carrier_id,omitempty"`
	Note              string              `json:"note,omitempty"`
	IsTesting         bool                `json:"is_testing,omitempty"`
}

// SupplierOrderLine represents a single item in a supplier order.
type SupplierOrderLine struct {
	EAN      string  `json:"ean,omitempty"`
	ItemID   string  `json:"item_id,omitempty"`
	Quantity float64 `json:"quantity"`
}

// SupplierOrderResult is returned after placing an order with a supplier.
type SupplierOrderResult struct {
	ExternalOrderID string `json:"external_order_id"`
	OrderNumber     string `json:"order_number"`
	OrderDate       string `json:"order_date"`
}

// SupplierProvider defines the interface for supplier/wholesaler integrations.
type SupplierProvider interface {
	ProviderName() string
	FetchProducts(ctx context.Context) ([]SupplierProduct, error)
	FetchInventory(ctx context.Context) ([]SupplierProduct, error)
	CreateOrder(ctx context.Context, req SupplierOrderRequest) (*SupplierOrderResult, error)
}

// SupplierProviderFactory is a constructor function for supplier providers.
type SupplierProviderFactory func(credentials json.RawMessage, settings json.RawMessage) (SupplierProvider, error)

var (
	supplierFactories   = map[string]SupplierProviderFactory{}
	supplierFactoriesMu sync.RWMutex
)

// RegisterSupplierProvider registers a factory for the given supplier provider name.
func RegisterSupplierProvider(name string, factory SupplierProviderFactory) {
	supplierFactoriesMu.Lock()
	defer supplierFactoriesMu.Unlock()
	supplierFactories[name] = factory
}

// NewSupplierProvider creates a SupplierProvider for the given provider name.
func NewSupplierProvider(provider string, credentials json.RawMessage, settings json.RawMessage) (SupplierProvider, error) {
	supplierFactoriesMu.RLock()
	factory, ok := supplierFactories[provider]
	supplierFactoriesMu.RUnlock()

	if ok {
		return factory(credentials, settings)
	}

	return nil, fmt.Errorf("unknown supplier provider: %q", provider)
}

// HasSupplierProvider checks if a supplier provider is registered.
func HasSupplierProvider(provider string) bool {
	supplierFactoriesMu.RLock()
	defer supplierFactoriesMu.RUnlock()
	_, ok := supplierFactories[provider]
	return ok
}
