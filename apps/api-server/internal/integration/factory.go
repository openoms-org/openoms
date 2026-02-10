package integration

import (
	"encoding/json"
	"fmt"
	"sync"
)

// MarketplaceProviderFactory is a constructor function for marketplace providers.
type MarketplaceProviderFactory func(credentials json.RawMessage, settings json.RawMessage) (MarketplaceProvider, error)

var (
	marketplaceFactories   = map[string]MarketplaceProviderFactory{}
	marketplaceFactoriesMu sync.RWMutex
)

// RegisterMarketplaceProvider registers a factory for the given provider name.
func RegisterMarketplaceProvider(name string, factory MarketplaceProviderFactory) {
	marketplaceFactoriesMu.Lock()
	defer marketplaceFactoriesMu.Unlock()
	marketplaceFactories[name] = factory
}

// NewMarketplaceProvider creates a MarketplaceProvider for the given provider name.
func NewMarketplaceProvider(provider string, credentials json.RawMessage, settings json.RawMessage) (MarketplaceProvider, error) {
	marketplaceFactoriesMu.RLock()
	factory, ok := marketplaceFactories[provider]
	marketplaceFactoriesMu.RUnlock()

	if ok {
		return factory(credentials, settings)
	}

	switch provider {
	case "woocommerce":
		// TODO: return woocommerce.NewProvider(credentials, settings)
		return nil, fmt.Errorf("marketplace provider %q: not implemented", provider)
	default:
		return nil, fmt.Errorf("unknown marketplace provider: %q", provider)
	}
}

// CarrierProviderFactory is a constructor function for carrier providers.
type CarrierProviderFactory func(credentials json.RawMessage, settings json.RawMessage) (CarrierProvider, error)

var (
	carrierFactories   = map[string]CarrierProviderFactory{}
	carrierFactoriesMu sync.RWMutex
)

// RegisterCarrierProvider registers a factory for the given carrier provider name.
func RegisterCarrierProvider(name string, factory CarrierProviderFactory) {
	carrierFactoriesMu.Lock()
	defer carrierFactoriesMu.Unlock()
	carrierFactories[name] = factory
}

// NewCarrierProvider creates a CarrierProvider for the given provider name.
func NewCarrierProvider(provider string, credentials json.RawMessage, settings json.RawMessage) (CarrierProvider, error) {
	carrierFactoriesMu.RLock()
	factory, ok := carrierFactories[provider]
	carrierFactoriesMu.RUnlock()

	if ok {
		return factory(credentials, settings)
	}

	switch provider {
	case "dpd":
		// TODO: return dpd.NewProvider(credentials, settings)
		return nil, fmt.Errorf("carrier provider %q: not implemented", provider)
	default:
		return nil, fmt.Errorf("unknown carrier provider: %q", provider)
	}
}
