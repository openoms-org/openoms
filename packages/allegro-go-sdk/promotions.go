package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// PromotionService handles Allegro promotions/campaign management.
type PromotionService struct {
	client *Client
}

// List lists the seller's promotions.
func (s *PromotionService) List(ctx context.Context, params *ListPromotionsParams) (*PromotionList, error) {
	path := "/sale/loyalty/promotions"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result PromotionList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single promotion by ID.
func (s *PromotionService) Get(ctx context.Context, promotionID string) (*Promotion, error) {
	var result Promotion
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/loyalty/promotions/%s", promotionID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new promotion campaign.
func (s *PromotionService) Create(ctx context.Context, promotion CreatePromotionRequest) (*Promotion, error) {
	var result Promotion
	if err := s.client.do(ctx, "POST", "/sale/loyalty/promotions", promotion, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an existing promotion.
func (s *PromotionService) Update(ctx context.Context, promotionID string, promotion CreatePromotionRequest) (*Promotion, error) {
	var result Promotion
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/sale/loyalty/promotions/%s", promotionID), promotion, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a promotion.
func (s *PromotionService) Delete(ctx context.Context, promotionID string) error {
	return s.client.do(ctx, "DELETE", fmt.Sprintf("/sale/loyalty/promotions/%s", promotionID), nil, nil)
}

// ListBadges lists available offer promotion badges (Bold, Highlight, etc.)
func (s *PromotionService) ListBadges(ctx context.Context) (*BadgeList, error) {
	var result BadgeList
	if err := s.client.do(ctx, "GET", "/sale/offer-promotion-packages", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
