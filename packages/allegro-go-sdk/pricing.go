package allegro

import (
	"context"
	"net/url"
)

// PricingService handles Allegro fee calculation.
type PricingService struct {
	client *Client
}

// GetFeePreview calculates fees for an offer.
func (s *PricingService) GetFeePreview(ctx context.Context, offerID string) (*FeePreview, error) {
	path := "/pricing/offer-fee-preview"

	if offerID != "" {
		v := url.Values{}
		v.Set("offer.id", offerID)
		path += "?" + v.Encode()
	}

	var result FeePreview
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCommissions lists commission rates by category.
func (s *PricingService) GetCommissions(ctx context.Context, categoryID string) (*CommissionList, error) {
	path := "/pricing/offer-commissions"

	if categoryID != "" {
		v := url.Values{}
		v.Set("offer.category.id", categoryID)
		path += "?" + v.Encode()
	}

	var result CommissionList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
