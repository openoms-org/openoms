package allegro

import (
	"context"
	"net/url"
	"strconv"
)

// AccountService handles communication with the seller account endpoints.
type AccountService struct {
	client *Client
}

// GetMe retrieves the authenticated user's account information.
func (s *AccountService) GetMe(ctx context.Context) (*User, error) {
	var result User
	if err := s.client.do(ctx, "GET", "/me", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetQuality retrieves the seller's quality metrics.
func (s *AccountService) GetQuality(ctx context.Context) (*SellerQuality, error) {
	var result SellerQuality
	if err := s.client.do(ctx, "GET", "/sale/quality", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSmartStatus retrieves the seller's Smart! program status.
func (s *AccountService) GetSmartStatus(ctx context.Context) (*SmartStatus, error) {
	var result SmartStatus
	if err := s.client.do(ctx, "GET", "/sale/smart", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRatings retrieves a paginated list of user ratings.
func (s *AccountService) ListRatings(ctx context.Context, params *RatingsParams) (*RatingList, error) {
	path := "/sale/user-ratings"

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

	var result RatingList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListBilling retrieves a paginated list of billing entries.
func (s *AccountService) ListBilling(ctx context.Context, params *BillingParams) (*BillingList, error) {
	path := "/billing/billing-entries"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.TypeGroup != "" {
			v.Set("type.group", params.TypeGroup)
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result BillingList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
