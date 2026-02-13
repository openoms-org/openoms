package allegro

import (
	"context"
	"fmt"
)

// AfterSalesService handles Allegro after-sales conditions: return policies, implied warranties.
type AfterSalesService struct {
	client *Client
}

// --- Return Policies ---

// ListReturnPolicies lists the seller's return policies.
// GET /after-sales-service-conditions/return-policies
func (s *AfterSalesService) ListReturnPolicies(ctx context.Context) (*ReturnPolicyList, error) {
	var result ReturnPolicyList
	if err := s.client.do(ctx, "GET", "/after-sales-service-conditions/return-policies", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetReturnPolicy gets a single return policy.
// GET /after-sales-service-conditions/return-policies/{id}
func (s *AfterSalesService) GetReturnPolicy(ctx context.Context, policyID string) (*ReturnPolicy, error) {
	var result ReturnPolicy
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/after-sales-service-conditions/return-policies/%s", policyID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateReturnPolicy creates a return policy.
// POST /after-sales-service-conditions/return-policies
func (s *AfterSalesService) CreateReturnPolicy(ctx context.Context, policy CreateReturnPolicyRequest) (*ReturnPolicy, error) {
	var result ReturnPolicy
	if err := s.client.do(ctx, "POST", "/after-sales-service-conditions/return-policies", policy, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateReturnPolicy updates a return policy.
// PUT /after-sales-service-conditions/return-policies/{id}
func (s *AfterSalesService) UpdateReturnPolicy(ctx context.Context, policyID string, policy CreateReturnPolicyRequest) (*ReturnPolicy, error) {
	var result ReturnPolicy
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/after-sales-service-conditions/return-policies/%s", policyID), policy, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Implied Warranties (Rekojmia) ---

// ListWarranties lists the seller's implied warranty policies.
// GET /after-sales-service-conditions/implied-warranties
func (s *AfterSalesService) ListWarranties(ctx context.Context) (*WarrantyList, error) {
	var result WarrantyList
	if err := s.client.do(ctx, "GET", "/after-sales-service-conditions/implied-warranties", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetWarranty gets a single implied warranty.
// GET /after-sales-service-conditions/implied-warranties/{id}
func (s *AfterSalesService) GetWarranty(ctx context.Context, warrantyID string) (*ImpliedWarranty, error) {
	var result ImpliedWarranty
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/after-sales-service-conditions/implied-warranties/%s", warrantyID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateWarranty creates an implied warranty.
// POST /after-sales-service-conditions/implied-warranties
func (s *AfterSalesService) CreateWarranty(ctx context.Context, warranty CreateWarrantyRequest) (*ImpliedWarranty, error) {
	var result ImpliedWarranty
	if err := s.client.do(ctx, "POST", "/after-sales-service-conditions/implied-warranties", warranty, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateWarranty updates an implied warranty.
// PUT /after-sales-service-conditions/implied-warranties/{id}
func (s *AfterSalesService) UpdateWarranty(ctx context.Context, warrantyID string, warranty CreateWarrantyRequest) (*ImpliedWarranty, error) {
	var result ImpliedWarranty
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/after-sales-service-conditions/implied-warranties/%s", warrantyID), warranty, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
