package ebay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFulfillmentPolicies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/account/v1/fulfillment_policy" {
			t.Errorf("path = %q, want /sell/account/v1/fulfillment_policy", r.URL.Path)
		}
		if r.URL.Query().Get("marketplace_id") != "EBAY_PL" {
			t.Errorf("marketplace_id = %q, want EBAY_PL", r.URL.Query().Get("marketplace_id"))
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		resp := FulfillmentPoliciesResponse{
			FulfillmentPolicies: []FulfillmentPolicy{
				{FulfillmentPolicyID: "FP-1", Name: "Standard Shipping", MarketplaceID: "EBAY_PL"},
				{FulfillmentPolicyID: "FP-2", Name: "Express Shipping", MarketplaceID: "EBAY_PL", Description: "1-2 days"},
			},
			Total: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	policies, err := c.Account.GetFulfillmentPolicies(context.Background(), "EBAY_PL")
	if err != nil {
		t.Fatalf("GetFulfillmentPolicies error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("len(policies) = %d, want 2", len(policies))
	}
	if policies[0].FulfillmentPolicyID != "FP-1" {
		t.Errorf("policies[0].FulfillmentPolicyID = %q, want FP-1", policies[0].FulfillmentPolicyID)
	}
	if policies[0].Name != "Standard Shipping" {
		t.Errorf("policies[0].Name = %q, want Standard Shipping", policies[0].Name)
	}
	if policies[1].FulfillmentPolicyID != "FP-2" {
		t.Errorf("policies[1].FulfillmentPolicyID = %q, want FP-2", policies[1].FulfillmentPolicyID)
	}
	if policies[1].Description != "1-2 days" {
		t.Errorf("policies[1].Description = %q, want 1-2 days", policies[1].Description)
	}
}

func TestGetReturnPolicies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/account/v1/return_policy" {
			t.Errorf("path = %q, want /sell/account/v1/return_policy", r.URL.Path)
		}
		if r.URL.Query().Get("marketplace_id") != "EBAY_PL" {
			t.Errorf("marketplace_id = %q, want EBAY_PL", r.URL.Query().Get("marketplace_id"))
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		resp := ReturnPoliciesResponse{
			ReturnPolicies: []ReturnPolicy{
				{ReturnPolicyID: "RP-1", Name: "30 Day Returns", MarketplaceID: "EBAY_PL"},
				{ReturnPolicyID: "RP-2", Name: "No Returns", MarketplaceID: "EBAY_PL", Description: "All sales final"},
			},
			Total: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	policies, err := c.Account.GetReturnPolicies(context.Background(), "EBAY_PL")
	if err != nil {
		t.Fatalf("GetReturnPolicies error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("len(policies) = %d, want 2", len(policies))
	}
	if policies[0].ReturnPolicyID != "RP-1" {
		t.Errorf("policies[0].ReturnPolicyID = %q, want RP-1", policies[0].ReturnPolicyID)
	}
	if policies[0].Name != "30 Day Returns" {
		t.Errorf("policies[0].Name = %q, want 30 Day Returns", policies[0].Name)
	}
	if policies[1].ReturnPolicyID != "RP-2" {
		t.Errorf("policies[1].ReturnPolicyID = %q, want RP-2", policies[1].ReturnPolicyID)
	}
	if policies[1].Description != "All sales final" {
		t.Errorf("policies[1].Description = %q, want All sales final", policies[1].Description)
	}
}

func TestGetPaymentPolicies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/account/v1/payment_policy" {
			t.Errorf("path = %q, want /sell/account/v1/payment_policy", r.URL.Path)
		}
		if r.URL.Query().Get("marketplace_id") != "EBAY_PL" {
			t.Errorf("marketplace_id = %q, want EBAY_PL", r.URL.Query().Get("marketplace_id"))
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		resp := PaymentPoliciesResponse{
			PaymentPolicies: []PaymentPolicy{
				{PaymentPolicyID: "PP-1", Name: "PayPal", MarketplaceID: "EBAY_PL"},
				{PaymentPolicyID: "PP-2", Name: "Managed Payments", MarketplaceID: "EBAY_PL", Description: "eBay managed"},
			},
			Total: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	policies, err := c.Account.GetPaymentPolicies(context.Background(), "EBAY_PL")
	if err != nil {
		t.Fatalf("GetPaymentPolicies error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("len(policies) = %d, want 2", len(policies))
	}
	if policies[0].PaymentPolicyID != "PP-1" {
		t.Errorf("policies[0].PaymentPolicyID = %q, want PP-1", policies[0].PaymentPolicyID)
	}
	if policies[0].Name != "PayPal" {
		t.Errorf("policies[0].Name = %q, want PayPal", policies[0].Name)
	}
	if policies[1].PaymentPolicyID != "PP-2" {
		t.Errorf("policies[1].PaymentPolicyID = %q, want PP-2", policies[1].PaymentPolicyID)
	}
	if policies[1].Description != "eBay managed" {
		t.Errorf("policies[1].Description = %q, want eBay managed", policies[1].Description)
	}
}

func TestGetFulfillmentPolicies_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"errorId": 1001, "domain": "account", "category": "REQUEST", "message": "Invalid access token"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("bad_token"),
	)

	_, err := c.Account.GetFulfillmentPolicies(context.Background(), "EBAY_PL")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if len(apiErr.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(apiErr.Errors))
	}
	if apiErr.Errors[0].Message != "Invalid access token" {
		t.Errorf("Message = %q, want Invalid access token", apiErr.Errors[0].Message)
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected error to wrap ErrUnauthorized")
	}
}
