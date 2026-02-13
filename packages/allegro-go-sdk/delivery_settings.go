package allegro

import (
	"context"
	"fmt"
)

// DeliverySettingsService handles Allegro delivery/shipping configuration.
type DeliverySettingsService struct {
	client *Client
}

// Get retrieves the seller's delivery settings.
func (s *DeliverySettingsService) Get(ctx context.Context) (*DeliverySettings, error) {
	var result DeliverySettings
	if err := s.client.do(ctx, "GET", "/sale/delivery-settings", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates the seller's delivery settings.
func (s *DeliverySettingsService) Update(ctx context.Context, settings DeliverySettings) error {
	return s.client.do(ctx, "PUT", "/sale/delivery-settings", settings, nil)
}

// ListShippingRates lists the seller's shipping rate tables.
func (s *DeliverySettingsService) ListShippingRates(ctx context.Context) (*ShippingRateList, error) {
	var result ShippingRateList
	if err := s.client.do(ctx, "GET", "/sale/shipping-rates?seller.id=me", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetShippingRate retrieves a single shipping rate table by ID.
func (s *DeliverySettingsService) GetShippingRate(ctx context.Context, rateID string) (*ShippingRateSet, error) {
	var result ShippingRateSet
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/shipping-rates/%s", rateID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateShippingRate creates a new shipping rate table.
func (s *DeliverySettingsService) CreateShippingRate(ctx context.Context, rate CreateShippingRateRequest) (*ShippingRateSet, error) {
	var result ShippingRateSet
	if err := s.client.do(ctx, "POST", "/sale/shipping-rates", rate, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateShippingRate updates an existing shipping rate table.
func (s *DeliverySettingsService) UpdateShippingRate(ctx context.Context, rateID string, rate CreateShippingRateRequest) (*ShippingRateSet, error) {
	var result ShippingRateSet
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/sale/shipping-rates/%s", rateID), rate, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListDeliveryMethods lists all available Allegro delivery methods.
func (s *DeliverySettingsService) ListDeliveryMethods(ctx context.Context) (*DeliveryMethodList, error) {
	var result DeliveryMethodList
	if err := s.client.do(ctx, "GET", "/sale/delivery-methods", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
