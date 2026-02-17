package model

// ProductFeedConfig stores feed configuration in tenant settings JSONB under key "product_feeds".
type ProductFeedConfig struct {
	CeneoEnabled        bool     `json:"ceneo_enabled"`
	CeneoFeedToken      string   `json:"ceneo_feed_token"`
	GoogleEnabled       bool     `json:"google_enabled"`
	GoogleFeedToken     string   `json:"google_feed_token"`
	DefaultCurrency     string   `json:"default_currency"`
	DefaultShippingCost string   `json:"default_shipping_cost"`
	ExcludedCategories  []string `json:"excluded_categories,omitempty"`
	ExcludeOutOfStock   bool     `json:"exclude_out_of_stock"`
}

// DefaultProductFeedConfig returns the default (empty/disabled) feed configuration.
func DefaultProductFeedConfig() ProductFeedConfig {
	return ProductFeedConfig{
		DefaultCurrency:     "PLN",
		DefaultShippingCost: "12.99",
		ExcludedCategories:  []string{},
	}
}
