package ebay

import (
	"context"
	"fmt"
)

// AccountService handles eBay Sell Account API endpoints.
type AccountService struct {
	client *Client
}

// GetFulfillmentPolicies returns the seller's fulfillment policies for a marketplace.
// GET /sell/account/v1/fulfillment_policy?marketplace_id=X
func (s *AccountService) GetFulfillmentPolicies(ctx context.Context, marketplaceID string) ([]FulfillmentPolicy, error) {
	path := fmt.Sprintf("/sell/account/v1/fulfillment_policy?marketplace_id=%s", marketplaceID)
	var resp FulfillmentPoliciesResponse
	if err := s.client.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("ebay: get fulfillment policies: %w", err)
	}
	return resp.FulfillmentPolicies, nil
}

// GetReturnPolicies returns the seller's return policies for a marketplace.
// GET /sell/account/v1/return_policy?marketplace_id=X
func (s *AccountService) GetReturnPolicies(ctx context.Context, marketplaceID string) ([]ReturnPolicy, error) {
	path := fmt.Sprintf("/sell/account/v1/return_policy?marketplace_id=%s", marketplaceID)
	var resp ReturnPoliciesResponse
	if err := s.client.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("ebay: get return policies: %w", err)
	}
	return resp.ReturnPolicies, nil
}

// GetPaymentPolicies returns the seller's payment policies for a marketplace.
// GET /sell/account/v1/payment_policy?marketplace_id=X
func (s *AccountService) GetPaymentPolicies(ctx context.Context, marketplaceID string) ([]PaymentPolicy, error) {
	path := fmt.Sprintf("/sell/account/v1/payment_policy?marketplace_id=%s", marketplaceID)
	var resp PaymentPoliciesResponse
	if err := s.client.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("ebay: get payment policies: %w", err)
	}
	return resp.PaymentPolicies, nil
}
