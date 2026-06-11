package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func ptrStr(s string) *string     { return &s }
func ptrFloat(f float64) *float64 { return &f }
func ptrInt(i int) *int           { return &i }
func ptrBool(b bool) *bool        { return &b }

// ─── CreateRepricingRuleRequest.Validate ────────────────────────────────────

func TestCreateRepricingRuleRequest_Validate(t *testing.T) {
	valid := func() CreateRepricingRuleRequest {
		return CreateRepricingRuleRequest{
			Name:      "Margin rule",
			Strategy:  "margin",
			ScopeType: "all",
			Priority:  10,
			Params:    json.RawMessage(`{}`),
		}
	}

	tests := []struct {
		name    string
		req     CreateRepricingRuleRequest
		wantErr string
	}{
		{name: "valid", req: valid()},
		{
			name: "valid with min/max price",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.MinPrice = ptrFloat(10.0)
				r.MaxPrice = ptrFloat(100.0)
				return r
			}(),
		},
		{
			name:    "missing name",
			req:     CreateRepricingRuleRequest{Strategy: "margin", ScopeType: "all"},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateRepricingRuleRequest{Name: strings.Repeat("x", 201), Strategy: "margin", ScopeType: "all"},
			wantErr: "name",
		},
		{
			name: "name at max length (200)",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Name = strings.Repeat("x", 200)
				return r
			}(),
		},
		{
			name:    "invalid strategy",
			req:     CreateRepricingRuleRequest{Name: "Rule", Strategy: "random", ScopeType: "all"},
			wantErr: "strategy must be one of",
		},
		{
			name: "invalid strategy - competitive (removed)",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Strategy = "competitive"
				return r
			}(),
			wantErr: "strategy must be one of",
		},
		{
			name: "valid strategy - time_based",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Strategy = "time_based"
				return r
			}(),
		},
		{
			name: "valid strategy - stock_based",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Strategy = "stock_based"
				return r
			}(),
		},
		{
			name:    "invalid scope_type",
			req:     CreateRepricingRuleRequest{Name: "Rule", Strategy: "margin", ScopeType: "brand"},
			wantErr: "scope_type must be one of",
		},
		{
			name: "valid scope_type - category",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.ScopeType = "category"
				return r
			}(),
		},
		{
			name: "valid scope_type - tag",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.ScopeType = "tag"
				return r
			}(),
		},
		{
			name: "valid scope_type - product_ids",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.ScopeType = "product_ids"
				return r
			}(),
		},
		{
			name: "negative min_price",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.MinPrice = ptrFloat(-1)
				return r
			}(),
			wantErr: "min_price must be non-negative",
		},
		{
			name: "negative max_price",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.MaxPrice = ptrFloat(-1)
				return r
			}(),
			wantErr: "max_price must be non-negative",
		},
		{
			name: "min_price exceeds max_price",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.MinPrice = ptrFloat(100)
				r.MaxPrice = ptrFloat(50)
				return r
			}(),
			wantErr: "min_price must not exceed max_price",
		},
		{
			name: "priority negative",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Priority = -1
				return r
			}(),
			wantErr: "priority must be between 0 and 1000",
		},
		{
			name: "priority too high",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Priority = 1001
				return r
			}(),
			wantErr: "priority must be between 0 and 1000",
		},
		{
			name: "priority at max (1000)",
			req: func() CreateRepricingRuleRequest {
				r := valid()
				r.Priority = 1000
				return r
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateRepricingRuleRequest.Validate ────────────────────────────────────

func TestUpdateRepricingRuleRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateRepricingRuleRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateRepricingRuleRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req:  UpdateRepricingRuleRequest{Name: ptrStr("Updated")},
		},
		{
			name:    "empty name",
			req:     UpdateRepricingRuleRequest{Name: ptrStr("")},
			wantErr: "name must not be empty",
		},
		{
			name:    "invalid status",
			req:     UpdateRepricingRuleRequest{Status: ptrStr("deleted")},
			wantErr: "status must be one of",
		},
		{
			name: "valid status - active",
			req:  UpdateRepricingRuleRequest{Status: ptrStr("active")},
		},
		{
			name: "valid status - paused",
			req:  UpdateRepricingRuleRequest{Status: ptrStr("paused")},
		},
		{
			name: "valid status - archived",
			req:  UpdateRepricingRuleRequest{Status: ptrStr("archived")},
		},
		{
			name:    "invalid strategy",
			req:     UpdateRepricingRuleRequest{Strategy: ptrStr("unknown")},
			wantErr: "strategy must be one of",
		},
		{
			name: "valid strategy update",
			req:  UpdateRepricingRuleRequest{Strategy: ptrStr("stock_based")},
		},
		{
			name:    "invalid strategy update - competitive (removed)",
			req:     UpdateRepricingRuleRequest{Strategy: ptrStr("competitive")},
			wantErr: "strategy must be one of",
		},
		{
			name:    "invalid scope_type",
			req:     UpdateRepricingRuleRequest{ScopeType: ptrStr("brand")},
			wantErr: "scope_type must be one of",
		},
		{
			name: "valid scope_type update",
			req:  UpdateRepricingRuleRequest{ScopeType: ptrStr("tag")},
		},
		{
			name:    "priority negative",
			req:     UpdateRepricingRuleRequest{Priority: ptrInt(-1)},
			wantErr: "priority must be between 0 and 1000",
		},
		{
			name:    "priority too high",
			req:     UpdateRepricingRuleRequest{Priority: ptrInt(1001)},
			wantErr: "priority must be between 0 and 1000",
		},
		{
			name: "valid priority update",
			req:  UpdateRepricingRuleRequest{Priority: ptrInt(500)},
		},
		{
			name: "valid min_price update",
			req:  UpdateRepricingRuleRequest{MinPrice: ptrFloat(10)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreatePriceListRequest.Validate ────────────────────────────────────────

func TestCreatePriceListRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePriceListRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreatePriceListRequest{Name: "Wholesale"},
		},
		{
			name: "valid with all fields",
			req: CreatePriceListRequest{
				Name:         "Premium",
				Currency:     "EUR",
				DiscountType: "fixed",
				Active:       ptrBool(true),
			},
		},
		{
			name:    "missing name",
			req:     CreatePriceListRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreatePriceListRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreatePriceListRequest{Name: strings.Repeat("x", 501)},
			wantErr: "name",
		},
		{
			name: "name at max length (500)",
			req:  CreatePriceListRequest{Name: strings.Repeat("x", 500)},
		},
		{
			name:    "currency too long",
			req:     CreatePriceListRequest{Name: "List", Currency: strings.Repeat("X", 11)},
			wantErr: "currency",
		},
		{
			name:    "invalid discount_type",
			req:     CreatePriceListRequest{Name: "List", DiscountType: "bogus"},
			wantErr: "discount_type must be one of",
		},
		{
			name: "valid discount_type - percentage",
			req:  CreatePriceListRequest{Name: "List", DiscountType: "percentage"},
		},
		{
			name: "valid discount_type - fixed",
			req:  CreatePriceListRequest{Name: "List", DiscountType: "fixed"},
		},
		{
			name: "valid discount_type - override",
			req:  CreatePriceListRequest{Name: "List", DiscountType: "override"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePriceListRequest_Validate_Defaults(t *testing.T) {
	req := CreatePriceListRequest{Name: "List"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "PLN", req.Currency, "currency should default to PLN")
	assert.Equal(t, "percentage", req.DiscountType, "discount_type should default to percentage")
}

// ─── UpdatePriceListRequest.Validate ────────────────────────────────────────

func TestUpdatePriceListRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdatePriceListRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdatePriceListRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req:  UpdatePriceListRequest{Name: ptrStr("Updated")},
		},
		{
			name:    "empty name",
			req:     UpdatePriceListRequest{Name: ptrStr("")},
			wantErr: "name must not be empty",
		},
		{
			name:    "whitespace-only name",
			req:     UpdatePriceListRequest{Name: ptrStr("   ")},
			wantErr: "name must not be empty",
		},
		{
			name:    "name too long",
			req:     UpdatePriceListRequest{Name: ptrStr(strings.Repeat("x", 501))},
			wantErr: "name",
		},
		{
			name:    "invalid discount_type",
			req:     UpdatePriceListRequest{DiscountType: ptrStr("bogus")},
			wantErr: "discount_type must be one of",
		},
		{
			name: "valid discount_type update",
			req:  UpdatePriceListRequest{DiscountType: ptrStr("override")},
		},
		{
			name: "valid active update",
			req:  UpdatePriceListRequest{Active: ptrBool(false)},
		},
		{
			name: "valid description update",
			req:  UpdatePriceListRequest{Description: ptrStr("Updated desc")},
		},
		{
			name: "valid currency update",
			req:  UpdatePriceListRequest{Currency: ptrStr("EUR")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreatePriceListItemRequest.Validate ────────────────────────────────────

func TestCreatePriceListItemRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePriceListItemRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  CreatePriceListItemRequest{ProductID: uuid.New(), MinQuantity: 5},
		},
		{
			name: "valid with price and discount",
			req: CreatePriceListItemRequest{
				ProductID:   uuid.New(),
				Price:       ptrFloat(99.99),
				Discount:    ptrFloat(10.0),
				MinQuantity: 1,
			},
		},
		{
			name:    "missing product_id",
			req:     CreatePriceListItemRequest{MinQuantity: 1},
			wantErr: "product_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePriceListItemRequest_Validate_DefaultsMinQuantity(t *testing.T) {
	req := CreatePriceListItemRequest{ProductID: uuid.New(), MinQuantity: 0}
	require.NoError(t, req.Validate())
	assert.Equal(t, 1, req.MinQuantity, "min_quantity <= 0 should default to 1")
}

// ─── CreateSegmentRequest.Validate ──────────────────────────────────────────

func TestCreateSegmentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateSegmentRequest
		wantErr string
	}{
		{
			name: "valid manual",
			req:  CreateSegmentRequest{Name: "VIP", SegmentType: "manual", Color: "#ff0000"},
		},
		{
			name: "valid rfm_auto",
			req:  CreateSegmentRequest{Name: "High Value", SegmentType: "rfm_auto"},
		},
		{
			name: "valid rule_based",
			req:  CreateSegmentRequest{Name: "Frequent", SegmentType: "rule_based"},
		},
		{
			name:    "missing name",
			req:     CreateSegmentRequest{SegmentType: "manual"},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateSegmentRequest{Name: "   ", SegmentType: "manual"},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateSegmentRequest{Name: strings.Repeat("x", 256), SegmentType: "manual"},
			wantErr: "name",
		},
		{
			name: "name at max length (255)",
			req:  CreateSegmentRequest{Name: strings.Repeat("x", 255), SegmentType: "manual"},
		},
		{
			name:    "invalid segment_type",
			req:     CreateSegmentRequest{Name: "Seg", SegmentType: "dynamic"},
			wantErr: "segment_type must be",
		},
		{
			name:    "missing segment_type",
			req:     CreateSegmentRequest{Name: "Seg"},
			wantErr: "segment_type must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateSegmentRequest_Validate_DefaultColor(t *testing.T) {
	req := CreateSegmentRequest{Name: "Seg", SegmentType: "manual"}
	require.NoError(t, req.Validate())
	assert.Equal(t, "#6366f1", req.Color, "color should default to #6366f1")
}

// ─── UpdateSegmentRequest.Validate ──────────────────────────────────────────

func TestUpdateSegmentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateSegmentRequest
		wantErr string
	}{
		{
			name: "valid name update",
			req:  UpdateSegmentRequest{Name: ptrStr("Updated Segment")},
		},
		{
			name:    "empty name",
			req:     UpdateSegmentRequest{Name: ptrStr("")},
			wantErr: "name cannot be empty",
		},
		{
			name:    "whitespace-only name",
			req:     UpdateSegmentRequest{Name: ptrStr("   ")},
			wantErr: "name cannot be empty",
		},
		{
			name:    "name too long",
			req:     UpdateSegmentRequest{Name: ptrStr(strings.Repeat("x", 256))},
			wantErr: "name",
		},
		{
			name: "valid color update",
			req:  UpdateSegmentRequest{Color: ptrStr("#ff0000")},
		},
		{
			name: "valid description update",
			req:  UpdateSegmentRequest{Description: ptrStr("Desc")},
		},
		{
			name: "nil fields is ok (no required-at-least-one check here)",
			req:  UpdateSegmentRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── AddSegmentMemberRequest.Validate ───────────────────────────────────────

func TestAddSegmentMemberRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     AddSegmentMemberRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  AddSegmentMemberRequest{CustomerID: uuid.New()},
		},
		{
			name:    "missing customer_id",
			req:     AddSegmentMemberRequest{},
			wantErr: "customer_id is required",
		},
		{
			name:    "nil customer_id",
			req:     AddSegmentMemberRequest{CustomerID: uuid.Nil},
			wantErr: "customer_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateLoyaltyProgramRequest.Validate ───────────────────────────────────

func TestCreateLoyaltyProgramRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateLoyaltyProgramRequest
		wantErr string
	}{
		{
			name: "valid points program",
			req:  CreateLoyaltyProgramRequest{Name: "Rewards", ProgramType: "points", Config: json.RawMessage(`{"points_per_pln":1}`)},
		},
		{
			name: "valid tier program",
			req:  CreateLoyaltyProgramRequest{Name: "Tiers", ProgramType: "tier"},
		},
		{
			name: "valid discount_after_n program",
			req:  CreateLoyaltyProgramRequest{Name: "Buy 10 get 1", ProgramType: "discount_after_n"},
		},
		{
			name:    "missing name",
			req:     CreateLoyaltyProgramRequest{ProgramType: "points"},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateLoyaltyProgramRequest{Name: "   ", ProgramType: "points"},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateLoyaltyProgramRequest{Name: strings.Repeat("x", 256), ProgramType: "points"},
			wantErr: "name",
		},
		{
			name:    "invalid program_type",
			req:     CreateLoyaltyProgramRequest{Name: "Prog", ProgramType: "cashback"},
			wantErr: "program_type must be",
		},
		{
			name:    "missing program_type",
			req:     CreateLoyaltyProgramRequest{Name: "Prog"},
			wantErr: "program_type must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateLoyaltyProgramRequest_Validate_DefaultConfig(t *testing.T) {
	req := CreateLoyaltyProgramRequest{Name: "Prog", ProgramType: "points"}
	require.NoError(t, req.Validate())
	assert.Equal(t, json.RawMessage(`{}`), req.Config, "nil config should default to {}")
}

// ─── UpdateLoyaltyProgramRequest.Validate ───────────────────────────────────

func TestUpdateLoyaltyProgramRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateLoyaltyProgramRequest
		wantErr string
	}{
		{
			name: "valid name update",
			req:  UpdateLoyaltyProgramRequest{Name: ptrStr("Updated")},
		},
		{
			name:    "empty name",
			req:     UpdateLoyaltyProgramRequest{Name: ptrStr("")},
			wantErr: "name cannot be empty",
		},
		{
			name:    "whitespace-only name",
			req:     UpdateLoyaltyProgramRequest{Name: ptrStr("   ")},
			wantErr: "name cannot be empty",
		},
		{
			name: "valid status - active",
			req:  UpdateLoyaltyProgramRequest{Status: ptrStr("active")},
		},
		{
			name: "valid status - paused",
			req:  UpdateLoyaltyProgramRequest{Status: ptrStr("paused")},
		},
		{
			name: "valid status - ended",
			req:  UpdateLoyaltyProgramRequest{Status: ptrStr("ended")},
		},
		{
			name:    "invalid status",
			req:     UpdateLoyaltyProgramRequest{Status: ptrStr("deleted")},
			wantErr: "status must be",
		},
		{
			name: "empty update is ok (no at-least-one check)",
			req:  UpdateLoyaltyProgramRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── AwardPointsRequest.Validate ────────────────────────────────────────────

func TestAwardPointsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     AwardPointsRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  AwardPointsRequest{CustomerID: uuid.New(), Points: 100, Reason: "bonus"},
		},
		{
			name:    "missing customer_id",
			req:     AwardPointsRequest{Points: 100, Reason: "bonus"},
			wantErr: "customer_id is required",
		},
		{
			name:    "zero points",
			req:     AwardPointsRequest{CustomerID: uuid.New(), Points: 0, Reason: "bonus"},
			wantErr: "points must be greater than 0",
		},
		{
			name:    "negative points",
			req:     AwardPointsRequest{CustomerID: uuid.New(), Points: -5, Reason: "bonus"},
			wantErr: "points must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── RedeemPointsRequest.Validate ───────────────────────────────────────────

func TestRedeemPointsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RedeemPointsRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  RedeemPointsRequest{CustomerID: uuid.New(), Points: 50},
		},
		{
			name:    "missing customer_id",
			req:     RedeemPointsRequest{Points: 50},
			wantErr: "customer_id is required",
		},
		{
			name:    "zero points",
			req:     RedeemPointsRequest{CustomerID: uuid.New(), Points: 0},
			wantErr: "points must be greater than 0",
		},
		{
			name:    "negative points",
			req:     RedeemPointsRequest{CustomerID: uuid.New(), Points: -10},
			wantErr: "points must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── MergeOrdersRequest.Validate ────────────────────────────────────────────

func TestMergeOrdersRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     MergeOrdersRequest
		wantErr string
	}{
		{
			name: "valid with 2 orders",
			req:  MergeOrdersRequest{OrderIDs: []uuid.UUID{uuid.New(), uuid.New()}},
		},
		{
			name: "valid with 50 orders",
			req: func() MergeOrdersRequest {
				ids := make([]uuid.UUID, 50)
				for i := range ids {
					ids[i] = uuid.New()
				}
				return MergeOrdersRequest{OrderIDs: ids}
			}(),
		},
		{
			name:    "empty order_ids",
			req:     MergeOrdersRequest{},
			wantErr: "at least 2 order_ids are required",
		},
		{
			name:    "only 1 order_id",
			req:     MergeOrdersRequest{OrderIDs: []uuid.UUID{uuid.New()}},
			wantErr: "at least 2 order_ids are required",
		},
		{
			name: "too many order_ids (51)",
			req: func() MergeOrdersRequest {
				ids := make([]uuid.UUID, 51)
				for i := range ids {
					ids[i] = uuid.New()
				}
				return MergeOrdersRequest{OrderIDs: ids}
			}(),
			wantErr: "maximum 50 orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── SplitOrderRequest.Validate ─────────────────────────────────────────────

func TestSplitOrderRequest_Validate(t *testing.T) {
	validSplits := []SplitSpec{
		{Items: json.RawMessage(`[{"sku":"A","qty":1}]`)},
		{Items: json.RawMessage(`[{"sku":"B","qty":2}]`)},
	}

	tests := []struct {
		name    string
		req     SplitOrderRequest
		wantErr string
	}{
		{
			name: "valid with 2 splits",
			req:  SplitOrderRequest{Splits: validSplits},
		},
		{
			name: "valid with 20 splits",
			req: func() SplitOrderRequest {
				splits := make([]SplitSpec, 20)
				for i := range splits {
					splits[i] = SplitSpec{Items: json.RawMessage(`[{"sku":"X","qty":1}]`)}
				}
				return SplitOrderRequest{Splits: splits}
			}(),
		},
		{
			name:    "empty splits",
			req:     SplitOrderRequest{},
			wantErr: "at least 2 splits are required",
		},
		{
			name:    "only 1 split",
			req:     SplitOrderRequest{Splits: []SplitSpec{{Items: json.RawMessage(`[{"sku":"A","qty":1}]`)}}},
			wantErr: "at least 2 splits are required",
		},
		{
			name: "too many splits (21)",
			req: func() SplitOrderRequest {
				splits := make([]SplitSpec, 21)
				for i := range splits {
					splits[i] = SplitSpec{Items: json.RawMessage(`[{"sku":"X","qty":1}]`)}
				}
				return SplitOrderRequest{Splits: splits}
			}(),
			wantErr: "maximum 20 splits",
		},
		{
			name: "split with empty items",
			req: SplitOrderRequest{Splits: []SplitSpec{
				{Items: json.RawMessage(`[{"sku":"A","qty":1}]`)},
				{Items: json.RawMessage{}},
			}},
			wantErr: "split 2 must have items",
		},
		{
			name: "split with null items",
			req: SplitOrderRequest{Splits: []SplitSpec{
				{Items: json.RawMessage(`[{"sku":"A","qty":1}]`)},
				{Items: json.RawMessage(`null`)},
			}},
			wantErr: "split 2 must have items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateStockSyncChannelRequest.Validate ─────────────────────────────────

func TestCreateStockSyncChannelRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateStockSyncChannelRequest
		wantErr string
	}{
		{
			name: "valid allegro",
			req:  CreateStockSyncChannelRequest{ChannelType: "allegro"},
		},
		{
			name: "valid amazon with options",
			req: CreateStockSyncChannelRequest{
				ChannelType: "amazon",
				Enabled:     ptrBool(true),
				StockBuffer: ptrInt(5),
				SyncMode:    ptrStr("realtime"),
				Priority:    ptrInt(1),
			},
		},
		{
			name: "valid all channel types",
			req:  CreateStockSyncChannelRequest{ChannelType: "woocommerce"},
		},
		{
			name: "valid ebay",
			req:  CreateStockSyncChannelRequest{ChannelType: "ebay"},
		},
		{
			name: "valid shopify",
			req:  CreateStockSyncChannelRequest{ChannelType: "shopify"},
		},
		{
			name: "valid manual",
			req:  CreateStockSyncChannelRequest{ChannelType: "manual"},
		},
		{
			name:    "missing channel_type",
			req:     CreateStockSyncChannelRequest{},
			wantErr: "channel_type is required",
		},
		{
			name:    "whitespace-only channel_type",
			req:     CreateStockSyncChannelRequest{ChannelType: "   "},
			wantErr: "channel_type is required",
		},
		{
			name:    "invalid channel_type",
			req:     CreateStockSyncChannelRequest{ChannelType: "tiktok"},
			wantErr: "invalid channel_type",
		},
		{
			name:    "invalid sync_mode",
			req:     CreateStockSyncChannelRequest{ChannelType: "allegro", SyncMode: ptrStr("hourly")},
			wantErr: "invalid sync_mode",
		},
		{
			name: "valid sync_mode - scheduled",
			req:  CreateStockSyncChannelRequest{ChannelType: "allegro", SyncMode: ptrStr("scheduled")},
		},
		{
			name: "valid sync_mode - manual",
			req:  CreateStockSyncChannelRequest{ChannelType: "allegro", SyncMode: ptrStr("manual")},
		},
		{
			name:    "negative stock_buffer",
			req:     CreateStockSyncChannelRequest{ChannelType: "allegro", StockBuffer: ptrInt(-1)},
			wantErr: "stock_buffer must not be negative",
		},
		{
			name: "zero stock_buffer is valid",
			req:  CreateStockSyncChannelRequest{ChannelType: "allegro", StockBuffer: ptrInt(0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateStockSyncChannelRequest_Validate_NormalizesCase(t *testing.T) {
	req := CreateStockSyncChannelRequest{ChannelType: "  ALLEGRO  ", SyncMode: ptrStr(" ReAlTiMe ")}
	require.NoError(t, req.Validate())
	assert.Equal(t, "allegro", req.ChannelType)
	assert.Equal(t, "realtime", *req.SyncMode)
}

// ─── UpdateStockSyncChannelRequest.Validate ─────────────────────────────────

func TestUpdateStockSyncChannelRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateStockSyncChannelRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateStockSyncChannelRequest{},
			wantErr: "at least one field is required",
		},
		{
			name: "valid enabled update",
			req:  UpdateStockSyncChannelRequest{Enabled: ptrBool(false)},
		},
		{
			name: "valid stock_buffer update",
			req:  UpdateStockSyncChannelRequest{StockBuffer: ptrInt(10)},
		},
		{
			name:    "negative stock_buffer",
			req:     UpdateStockSyncChannelRequest{StockBuffer: ptrInt(-1)},
			wantErr: "stock_buffer must not be negative",
		},
		{
			name: "valid sync_mode update",
			req:  UpdateStockSyncChannelRequest{SyncMode: ptrStr("scheduled")},
		},
		{
			name:    "invalid sync_mode",
			req:     UpdateStockSyncChannelRequest{SyncMode: ptrStr("hourly")},
			wantErr: "invalid sync_mode",
		},
		{
			name: "valid priority update",
			req:  UpdateStockSyncChannelRequest{Priority: ptrInt(5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreatePickPackSessionRequest.Validate ──────────────────────────────────

func TestCreatePickPackSessionRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePickPackSessionRequest
		wantErr string
	}{
		{
			name: "valid with one order",
			req:  CreatePickPackSessionRequest{OrderIDs: []uuid.UUID{uuid.New()}},
		},
		{
			name: "valid with multiple orders",
			req:  CreatePickPackSessionRequest{OrderIDs: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}},
		},
		{
			name:    "empty order_ids",
			req:     CreatePickPackSessionRequest{},
			wantErr: "at least one order_id is required",
		},
		{
			name:    "contains nil UUID",
			req:     CreatePickPackSessionRequest{OrderIDs: []uuid.UUID{uuid.New(), uuid.Nil}},
			wantErr: "order_ids must not contain empty UUIDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── ScanItemRequest.Validate ───────────────────────────────────────────────

func TestScanItemRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ScanItemRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  ScanItemRequest{Barcode: "5901234123457"},
		},
		{
			name:    "empty barcode",
			req:     ScanItemRequest{},
			wantErr: "barcode is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── MarkPackedRequest.Validate ─────────────────────────────────────────────

func TestMarkPackedRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     MarkPackedRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  MarkPackedRequest{Quantity: 5},
		},
		{
			name:    "zero quantity",
			req:     MarkPackedRequest{Quantity: 0},
			wantErr: "quantity must be positive",
		},
		{
			name:    "negative quantity",
			req:     MarkPackedRequest{Quantity: -1},
			wantErr: "quantity must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreatePurchaseOrderRequest.Validate ────────────────────────────────────

func TestCreatePurchaseOrderRequest_Validate(t *testing.T) {
	validItem := CreatePurchaseOrderItemReq{
		ProductName: "Widget",
		Quantity:    10,
		UnitCost:    5.99,
		SKU:         "WDG-001",
	}

	tests := []struct {
		name    string
		req     CreatePurchaseOrderRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreatePurchaseOrderRequest{SupplierName: "ACME", Items: []CreatePurchaseOrderItemReq{validItem}},
		},
		{
			name: "valid with all fields",
			req: func() CreatePurchaseOrderRequest {
				sid := uuid.New()
				wid := uuid.New()
				return CreatePurchaseOrderRequest{
					SupplierID:   &sid,
					SupplierName: "ACME Corp",
					WarehouseID:  &wid,
					Notes:        "Urgent order",
					Currency:     "EUR",
					Items:        []CreatePurchaseOrderItemReq{validItem},
				}
			}(),
		},
		{
			name:    "missing supplier_name",
			req:     CreatePurchaseOrderRequest{Items: []CreatePurchaseOrderItemReq{validItem}},
			wantErr: "supplier_name is required",
		},
		{
			name:    "whitespace-only supplier_name",
			req:     CreatePurchaseOrderRequest{SupplierName: "   ", Items: []CreatePurchaseOrderItemReq{validItem}},
			wantErr: "supplier_name is required",
		},
		{
			name:    "supplier_name too long",
			req:     CreatePurchaseOrderRequest{SupplierName: strings.Repeat("x", 501), Items: []CreatePurchaseOrderItemReq{validItem}},
			wantErr: "supplier_name",
		},
		{
			name:    "no items",
			req:     CreatePurchaseOrderRequest{SupplierName: "ACME"},
			wantErr: "at least one item is required",
		},
		{
			name: "item with missing product_name",
			req: CreatePurchaseOrderRequest{
				SupplierName: "ACME",
				Items:        []CreatePurchaseOrderItemReq{{Quantity: 5, UnitCost: 1.0}},
			},
			wantErr: "product_name is required",
		},
		{
			name: "item with zero quantity",
			req: CreatePurchaseOrderRequest{
				SupplierName: "ACME",
				Items:        []CreatePurchaseOrderItemReq{{ProductName: "Widget", Quantity: 0, UnitCost: 1.0}},
			},
			wantErr: "quantity must be greater than 0",
		},
		{
			name: "item with negative unit_cost",
			req: CreatePurchaseOrderRequest{
				SupplierName: "ACME",
				Items:        []CreatePurchaseOrderItemReq{{ProductName: "Widget", Quantity: 5, UnitCost: -1.0}},
			},
			wantErr: "unit_cost must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePurchaseOrderRequest_Validate_DefaultCurrency(t *testing.T) {
	req := CreatePurchaseOrderRequest{
		SupplierName: "ACME",
		Items:        []CreatePurchaseOrderItemReq{{ProductName: "Widget", Quantity: 1, UnitCost: 0}},
	}
	require.NoError(t, req.Validate())
	assert.Equal(t, "PLN", req.Currency, "currency should default to PLN")
}

// ─── CreatePurchaseOrderItemReq.Validate ────────────────────────────────────

func TestCreatePurchaseOrderItemReq_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePurchaseOrderItemReq
		wantErr string
	}{
		{
			name: "valid",
			req:  CreatePurchaseOrderItemReq{ProductName: "Widget", Quantity: 10, UnitCost: 5.99},
		},
		{
			name: "valid with zero unit_cost",
			req:  CreatePurchaseOrderItemReq{ProductName: "Free Sample", Quantity: 1, UnitCost: 0},
		},
		{
			name:    "missing product_name",
			req:     CreatePurchaseOrderItemReq{Quantity: 5, UnitCost: 1.0},
			wantErr: "product_name is required",
		},
		{
			name:    "whitespace-only product_name",
			req:     CreatePurchaseOrderItemReq{ProductName: "   ", Quantity: 5, UnitCost: 1.0},
			wantErr: "product_name is required",
		},
		{
			name:    "zero quantity",
			req:     CreatePurchaseOrderItemReq{ProductName: "W", Quantity: 0, UnitCost: 1.0},
			wantErr: "quantity must be greater than 0",
		},
		{
			name:    "negative quantity",
			req:     CreatePurchaseOrderItemReq{ProductName: "W", Quantity: -1, UnitCost: 1.0},
			wantErr: "quantity must be greater than 0",
		},
		{
			name:    "negative unit_cost",
			req:     CreatePurchaseOrderItemReq{ProductName: "W", Quantity: 1, UnitCost: -0.01},
			wantErr: "unit_cost must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdatePurchaseOrderRequest.Validate ────────────────────────────────────

func TestUpdatePurchaseOrderRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdatePurchaseOrderRequest
		wantErr string
	}{
		{
			name: "valid supplier_name update",
			req:  UpdatePurchaseOrderRequest{SupplierName: ptrStr("Updated Supplier")},
		},
		{
			name:    "empty supplier_name",
			req:     UpdatePurchaseOrderRequest{SupplierName: ptrStr("")},
			wantErr: "supplier_name must not be empty",
		},
		{
			name:    "whitespace-only supplier_name",
			req:     UpdatePurchaseOrderRequest{SupplierName: ptrStr("   ")},
			wantErr: "supplier_name must not be empty",
		},
		{
			name: "valid items update",
			req: UpdatePurchaseOrderRequest{
				Items: []CreatePurchaseOrderItemReq{{ProductName: "New Widget", Quantity: 5, UnitCost: 3.0}},
			},
		},
		{
			name: "invalid item in update",
			req: UpdatePurchaseOrderRequest{
				Items: []CreatePurchaseOrderItemReq{{ProductName: "", Quantity: 5, UnitCost: 3.0}},
			},
			wantErr: "product_name is required",
		},
		{
			name: "empty update is ok (no at-least-one check)",
			req:  UpdatePurchaseOrderRequest{},
		},
		{
			name: "valid notes update",
			req:  UpdatePurchaseOrderRequest{Notes: ptrStr("Updated notes")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── ReceiveItemsRequest.Validate ───────────────────────────────────────────

func TestReceiveItemsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ReceiveItemsRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  ReceiveItemsRequest{Items: []ReceiveItemEntry{{ItemID: uuid.New(), QuantityReceived: 5}}},
		},
		{
			name: "valid multiple items",
			req: ReceiveItemsRequest{Items: []ReceiveItemEntry{
				{ItemID: uuid.New(), QuantityReceived: 5},
				{ItemID: uuid.New(), QuantityReceived: 10},
			}},
		},
		{
			name:    "empty items",
			req:     ReceiveItemsRequest{},
			wantErr: "at least one item is required",
		},
		{
			name:    "nil item_id",
			req:     ReceiveItemsRequest{Items: []ReceiveItemEntry{{ItemID: uuid.Nil, QuantityReceived: 5}}},
			wantErr: "item_id is required",
		},
		{
			name:    "zero quantity_received",
			req:     ReceiveItemsRequest{Items: []ReceiveItemEntry{{ItemID: uuid.New(), QuantityReceived: 0}}},
			wantErr: "quantity_received must be greater than 0",
		},
		{
			name:    "negative quantity_received",
			req:     ReceiveItemsRequest{Items: []ReceiveItemEntry{{ItemID: uuid.New(), QuantityReceived: -1}}},
			wantErr: "quantity_received must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateRecurringOrderRequest.Validate ───────────────────────────────────

func TestCreateRecurringOrderRequest_Validate(t *testing.T) {
	validItem := CreateRecurringOrderItemRequest{
		SKU:         "WDG-001",
		ProductName: "Widget",
		Quantity:    2,
		UnitPrice:   29.99,
	}

	tests := []struct {
		name    string
		req     CreateRecurringOrderRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan Kowalski",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{validItem},
			},
		},
		{
			name: "valid with all optional fields",
			req: func() CreateRecurringOrderRequest {
				cid := uuid.New()
				email := "jan@example.com"
				endDate := "2026-12-31"
				maxOrders := 12
				notes := "Subscription"
				return CreateRecurringOrderRequest{
					CustomerID:    &cid,
					CustomerName:  "Jan Kowalski",
					CustomerEmail: &email,
					Frequency:     "weekly",
					NextOrderDate: "2026-03-01",
					EndDate:       &endDate,
					MaxOrders:     &maxOrders,
					Notes:         &notes,
					Items:         []CreateRecurringOrderItemRequest{validItem},
				}
			}(),
		},
		{
			name: "missing customer_name",
			req: CreateRecurringOrderRequest{
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{validItem},
			},
			wantErr: "customer_name is required",
		},
		{
			name: "whitespace-only customer_name",
			req: CreateRecurringOrderRequest{
				CustomerName:  "   ",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{validItem},
			},
			wantErr: "customer_name is required",
		},
		{
			name: "customer_name too long",
			req: CreateRecurringOrderRequest{
				CustomerName:  strings.Repeat("x", 501),
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{validItem},
			},
			wantErr: "customer_name",
		},
		{
			name: "customer_email too long",
			req: func() CreateRecurringOrderRequest {
				email := strings.Repeat("x", 256)
				return CreateRecurringOrderRequest{
					CustomerName:  "Jan",
					CustomerEmail: &email,
					Frequency:     "monthly",
					NextOrderDate: "2026-03-01",
					Items:         []CreateRecurringOrderItemRequest{validItem},
				}
			}(),
			wantErr: "customer_email",
		},
		{
			name: "invalid frequency",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "yearly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{validItem},
			},
			wantErr: "frequency must be one of",
		},
		{
			name: "valid frequency - daily",
			req: CreateRecurringOrderRequest{
				CustomerName: "Jan", Frequency: "daily", NextOrderDate: "2026-03-01",
				Items: []CreateRecurringOrderItemRequest{validItem},
			},
		},
		{
			name: "valid frequency - weekly",
			req: CreateRecurringOrderRequest{
				CustomerName: "Jan", Frequency: "weekly", NextOrderDate: "2026-03-01",
				Items: []CreateRecurringOrderItemRequest{validItem},
			},
		},
		{
			name: "valid frequency - biweekly",
			req: CreateRecurringOrderRequest{
				CustomerName: "Jan", Frequency: "biweekly", NextOrderDate: "2026-03-01",
				Items: []CreateRecurringOrderItemRequest{validItem},
			},
		},
		{
			name: "valid frequency - quarterly",
			req: CreateRecurringOrderRequest{
				CustomerName: "Jan", Frequency: "quarterly", NextOrderDate: "2026-03-01",
				Items: []CreateRecurringOrderItemRequest{validItem},
			},
		},
		{
			name: "missing next_order_date",
			req: CreateRecurringOrderRequest{
				CustomerName: "Jan",
				Frequency:    "monthly",
				Items:        []CreateRecurringOrderItemRequest{validItem},
			},
			wantErr: "next_order_date is required",
		},
		{
			name: "invalid next_order_date format",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "monthly",
				NextOrderDate: "01/03/2026",
				Items:         []CreateRecurringOrderItemRequest{validItem},
			},
			wantErr: "next_order_date must be YYYY-MM-DD format",
		},
		{
			name: "invalid end_date format",
			req: func() CreateRecurringOrderRequest {
				endDate := "2026-13-01"
				return CreateRecurringOrderRequest{
					CustomerName:  "Jan",
					Frequency:     "monthly",
					NextOrderDate: "2026-03-01",
					EndDate:       &endDate,
					Items:         []CreateRecurringOrderItemRequest{validItem},
				}
			}(),
			wantErr: "end_date must be YYYY-MM-DD format",
		},
		{
			name: "zero max_orders",
			req: func() CreateRecurringOrderRequest {
				maxOrders := 0
				return CreateRecurringOrderRequest{
					CustomerName:  "Jan",
					Frequency:     "monthly",
					NextOrderDate: "2026-03-01",
					MaxOrders:     &maxOrders,
					Items:         []CreateRecurringOrderItemRequest{validItem},
				}
			}(),
			wantErr: "max_orders must be a positive integer",
		},
		{
			name: "negative max_orders",
			req: func() CreateRecurringOrderRequest {
				maxOrders := -5
				return CreateRecurringOrderRequest{
					CustomerName:  "Jan",
					Frequency:     "monthly",
					NextOrderDate: "2026-03-01",
					MaxOrders:     &maxOrders,
					Items:         []CreateRecurringOrderItemRequest{validItem},
				}
			}(),
			wantErr: "max_orders must be a positive integer",
		},
		{
			name: "empty items",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
			},
			wantErr: "at least one item is required",
		},
		{
			name: "item missing sku",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{{ProductName: "W", Quantity: 1, UnitPrice: 10}},
			},
			wantErr: "items[0].sku is required",
		},
		{
			name: "item missing product_name",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{{SKU: "W-01", Quantity: 1, UnitPrice: 10}},
			},
			wantErr: "items[0].product_name is required",
		},
		{
			name: "item zero quantity",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{{SKU: "W-01", ProductName: "W", Quantity: 0, UnitPrice: 10}},
			},
			wantErr: "items[0].quantity must be positive",
		},
		{
			name: "item negative unit_price",
			req: CreateRecurringOrderRequest{
				CustomerName:  "Jan",
				Frequency:     "monthly",
				NextOrderDate: "2026-03-01",
				Items:         []CreateRecurringOrderItemRequest{{SKU: "W-01", ProductName: "W", Quantity: 1, UnitPrice: -1}},
			},
			wantErr: "items[0].unit_price must be non-negative",
		},
		{
			name: "notes too long",
			req: func() CreateRecurringOrderRequest {
				notes := strings.Repeat("x", 5001)
				return CreateRecurringOrderRequest{
					CustomerName:  "Jan",
					Frequency:     "monthly",
					NextOrderDate: "2026-03-01",
					Notes:         &notes,
					Items:         []CreateRecurringOrderItemRequest{validItem},
				}
			}(),
			wantErr: "notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateRecurringOrderRequest.Validate ───────────────────────────────────

func TestUpdateRecurringOrderRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateRecurringOrderRequest
		wantErr string
	}{
		{
			name: "valid customer_name update",
			req:  UpdateRecurringOrderRequest{CustomerName: ptrStr("Updated Name")},
		},
		{
			name:    "empty customer_name",
			req:     UpdateRecurringOrderRequest{CustomerName: ptrStr("")},
			wantErr: "customer_name cannot be empty",
		},
		{
			name:    "whitespace-only customer_name",
			req:     UpdateRecurringOrderRequest{CustomerName: ptrStr("   ")},
			wantErr: "customer_name cannot be empty",
		},
		{
			name:    "customer_name too long",
			req:     UpdateRecurringOrderRequest{CustomerName: ptrStr(strings.Repeat("x", 501))},
			wantErr: "customer_name",
		},
		{
			name: "customer_email too long",
			req: func() UpdateRecurringOrderRequest {
				email := strings.Repeat("x", 256)
				return UpdateRecurringOrderRequest{CustomerEmail: &email}
			}(),
			wantErr: "customer_email",
		},
		{
			name:    "invalid frequency",
			req:     UpdateRecurringOrderRequest{Frequency: ptrStr("yearly")},
			wantErr: "frequency must be one of",
		},
		{
			name: "valid frequency update",
			req:  UpdateRecurringOrderRequest{Frequency: ptrStr("weekly")},
		},
		{
			name:    "invalid next_order_date format",
			req:     UpdateRecurringOrderRequest{NextOrderDate: ptrStr("not-a-date")},
			wantErr: "next_order_date must be YYYY-MM-DD format",
		},
		{
			name: "valid next_order_date update",
			req:  UpdateRecurringOrderRequest{NextOrderDate: ptrStr("2026-06-15")},
		},
		{
			name:    "invalid end_date format",
			req:     UpdateRecurringOrderRequest{EndDate: ptrStr("2026-13-01")},
			wantErr: "end_date must be YYYY-MM-DD format",
		},
		{
			name: "valid end_date update",
			req:  UpdateRecurringOrderRequest{EndDate: ptrStr("2026-12-31")},
		},
		{
			name:    "zero max_orders",
			req:     UpdateRecurringOrderRequest{MaxOrders: ptrInt(0)},
			wantErr: "max_orders must be a positive integer",
		},
		{
			name:    "negative max_orders",
			req:     UpdateRecurringOrderRequest{MaxOrders: ptrInt(-1)},
			wantErr: "max_orders must be a positive integer",
		},
		{
			name: "valid max_orders update",
			req:  UpdateRecurringOrderRequest{MaxOrders: ptrInt(24)},
		},
		{
			name: "valid items update",
			req: func() UpdateRecurringOrderRequest {
				items := []CreateRecurringOrderItemRequest{
					{SKU: "W-01", ProductName: "Widget", Quantity: 3, UnitPrice: 19.99},
				}
				return UpdateRecurringOrderRequest{Items: &items}
			}(),
		},
		{
			name: "item with empty sku in update",
			req: func() UpdateRecurringOrderRequest {
				items := []CreateRecurringOrderItemRequest{
					{ProductName: "Widget", Quantity: 3, UnitPrice: 19.99},
				}
				return UpdateRecurringOrderRequest{Items: &items}
			}(),
			wantErr: "items[0].sku is required",
		},
		{
			name: "item with negative unit_price in update",
			req: func() UpdateRecurringOrderRequest {
				items := []CreateRecurringOrderItemRequest{
					{SKU: "W-01", ProductName: "Widget", Quantity: 3, UnitPrice: -1},
				}
				return UpdateRecurringOrderRequest{Items: &items}
			}(),
			wantErr: "items[0].unit_price must be non-negative",
		},
		{
			name: "notes too long",
			req: func() UpdateRecurringOrderRequest {
				notes := strings.Repeat("x", 5001)
				return UpdateRecurringOrderRequest{Notes: &notes}
			}(),
			wantErr: "notes",
		},
		{
			name: "empty update is ok",
			req:  UpdateRecurringOrderRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateVariantRequest.Validate ──────────────────────────────────────────

func TestCreateVariantRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateVariantRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateVariantRequest{Name: "Large / Red"},
		},
		{
			name: "valid with all fields",
			req: func() CreateVariantRequest {
				sku := "WDG-L-R"
				ean := "5901234123457"
				return CreateVariantRequest{
					SKU:           &sku,
					EAN:           &ean,
					Name:          "Large / Red",
					PriceOverride: ptrFloat(49.99),
					StockQuantity: 100,
					Weight:        ptrFloat(0.5),
					ImageURL:      ptrStr("https://example.com/img.jpg"),
					Position:      ptrInt(1),
					Active:        ptrBool(true),
				}
			}(),
		},
		{
			name:    "missing name",
			req:     CreateVariantRequest{},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateVariantRequest{Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "negative stock_quantity",
			req:     CreateVariantRequest{Name: "V", StockQuantity: -1},
			wantErr: "stock_quantity must not be negative",
		},
		{
			name: "zero stock_quantity is valid",
			req:  CreateVariantRequest{Name: "V", StockQuantity: 0},
		},
		{
			name:    "negative price_override",
			req:     CreateVariantRequest{Name: "V", PriceOverride: ptrFloat(-1)},
			wantErr: "price_override must not be negative",
		},
		{
			name: "zero price_override is valid",
			req:  CreateVariantRequest{Name: "V", PriceOverride: ptrFloat(0)},
		},
		{
			name:    "name too long",
			req:     CreateVariantRequest{Name: strings.Repeat("x", 501)},
			wantErr: "name",
		},
		{
			name: "name at max length (500)",
			req:  CreateVariantRequest{Name: strings.Repeat("x", 500)},
		},
		{
			name: "sku too long",
			req: func() CreateVariantRequest {
				sku := strings.Repeat("x", 101)
				return CreateVariantRequest{Name: "V", SKU: &sku}
			}(),
			wantErr: "sku",
		},
		{
			name: "sku at max length (100)",
			req: func() CreateVariantRequest {
				sku := strings.Repeat("x", 100)
				return CreateVariantRequest{Name: "V", SKU: &sku}
			}(),
		},
		{
			name: "ean too long",
			req: func() CreateVariantRequest {
				ean := strings.Repeat("0", 51)
				return CreateVariantRequest{Name: "V", EAN: &ean}
			}(),
			wantErr: "ean",
		},
		{
			name: "ean at max length (50)",
			req: func() CreateVariantRequest {
				ean := strings.Repeat("0", 50)
				return CreateVariantRequest{Name: "V", EAN: &ean}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateVariantRequest.Validate ──────────────────────────────────────────

func TestUpdateVariantRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateVariantRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateVariantRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid name update",
			req:  UpdateVariantRequest{Name: ptrStr("Updated Variant")},
		},
		{
			name:    "empty name",
			req:     UpdateVariantRequest{Name: ptrStr("")},
			wantErr: "name must not be empty",
		},
		{
			name:    "whitespace-only name",
			req:     UpdateVariantRequest{Name: ptrStr("   ")},
			wantErr: "name must not be empty",
		},
		{
			name:    "negative stock_quantity",
			req:     UpdateVariantRequest{StockQuantity: ptrInt(-1)},
			wantErr: "stock_quantity must not be negative",
		},
		{
			name: "zero stock_quantity is valid",
			req:  UpdateVariantRequest{StockQuantity: ptrInt(0)},
		},
		{
			name:    "negative price_override",
			req:     UpdateVariantRequest{PriceOverride: ptrFloat(-0.01)},
			wantErr: "price_override must not be negative",
		},
		{
			name: "zero price_override is valid",
			req:  UpdateVariantRequest{PriceOverride: ptrFloat(0)},
		},
		{
			name:    "name too long",
			req:     UpdateVariantRequest{Name: ptrStr(strings.Repeat("x", 501))},
			wantErr: "name",
		},
		{
			name: "sku too long",
			req: func() UpdateVariantRequest {
				sku := strings.Repeat("x", 101)
				return UpdateVariantRequest{SKU: &sku}
			}(),
			wantErr: "sku",
		},
		{
			name: "ean too long",
			req: func() UpdateVariantRequest {
				ean := strings.Repeat("0", 51)
				return UpdateVariantRequest{EAN: &ean}
			}(),
			wantErr: "ean",
		},
		{
			name: "valid active update",
			req:  UpdateVariantRequest{Active: ptrBool(false)},
		},
		{
			name: "valid position update",
			req:  UpdateVariantRequest{Position: ptrInt(5)},
		},
		{
			name: "valid weight update",
			req:  UpdateVariantRequest{Weight: ptrFloat(1.5)},
		},
		{
			name: "valid image_url update",
			req:  UpdateVariantRequest{ImageURL: ptrStr("https://example.com/new.jpg")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateBundleComponentRequest.Validate ──────────────────────────────────

func TestCreateBundleComponentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateBundleComponentRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  CreateBundleComponentRequest{ComponentProductID: uuid.New(), Quantity: 3, Position: 1},
		},
		{
			name: "valid with variant_id",
			req: func() CreateBundleComponentRequest {
				vid := uuid.New()
				return CreateBundleComponentRequest{ComponentProductID: uuid.New(), ComponentVariantID: &vid, Quantity: 1}
			}(),
		},
		{
			name:    "missing component_product_id",
			req:     CreateBundleComponentRequest{Quantity: 1},
			wantErr: "component_product_id is required",
		},
		{
			name:    "nil component_product_id",
			req:     CreateBundleComponentRequest{ComponentProductID: uuid.Nil, Quantity: 1},
			wantErr: "component_product_id is required",
		},
		{
			name:    "zero quantity",
			req:     CreateBundleComponentRequest{ComponentProductID: uuid.New(), Quantity: 0},
			wantErr: "quantity must be at least 1",
		},
		{
			name:    "negative quantity",
			req:     CreateBundleComponentRequest{ComponentProductID: uuid.New(), Quantity: -1},
			wantErr: "quantity must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateBundleComponentRequest.Validate ──────────────────────────────────

func TestUpdateBundleComponentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateBundleComponentRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateBundleComponentRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid quantity update",
			req:  UpdateBundleComponentRequest{Quantity: ptrInt(5)},
		},
		{
			name: "valid position update",
			req:  UpdateBundleComponentRequest{Position: ptrInt(2)},
		},
		{
			name: "valid both fields",
			req:  UpdateBundleComponentRequest{Quantity: ptrInt(3), Position: ptrInt(1)},
		},
		{
			name:    "zero quantity",
			req:     UpdateBundleComponentRequest{Quantity: ptrInt(0)},
			wantErr: "quantity must be at least 1",
		},
		{
			name:    "negative quantity",
			req:     UpdateBundleComponentRequest{Quantity: ptrInt(-1)},
			wantErr: "quantity must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateExchangeRateRequest.Validate ─────────────────────────────────────

func TestCreateExchangeRateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateExchangeRateRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  CreateExchangeRateRequest{BaseCurrency: "PLN", TargetCurrency: "EUR", Rate: 0.22},
		},
		{
			name: "valid with source",
			req:  CreateExchangeRateRequest{BaseCurrency: "PLN", TargetCurrency: "USD", Rate: 0.25, Source: "nbp"},
		},
		{
			name:    "missing base_currency",
			req:     CreateExchangeRateRequest{TargetCurrency: "EUR", Rate: 0.22},
			wantErr: "base_currency is required",
		},
		{
			name:    "missing target_currency",
			req:     CreateExchangeRateRequest{BaseCurrency: "PLN", Rate: 0.22},
			wantErr: "target_currency is required",
		},
		{
			name:    "zero rate",
			req:     CreateExchangeRateRequest{BaseCurrency: "PLN", TargetCurrency: "EUR", Rate: 0},
			wantErr: "rate must be positive",
		},
		{
			name:    "negative rate",
			req:     CreateExchangeRateRequest{BaseCurrency: "PLN", TargetCurrency: "EUR", Rate: -1},
			wantErr: "rate must be positive",
		},
		{
			name:    "same currencies",
			req:     CreateExchangeRateRequest{BaseCurrency: "PLN", TargetCurrency: "PLN", Rate: 1},
			wantErr: "base_currency and target_currency must be different",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateExchangeRateRequest.Validate ─────────────────────────────────────

func TestUpdateExchangeRateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateExchangeRateRequest
		wantErr string
	}{
		{
			name:    "empty update",
			req:     UpdateExchangeRateRequest{},
			wantErr: "at least one field must be provided",
		},
		{
			name: "valid rate update",
			req:  UpdateExchangeRateRequest{Rate: ptrFloat(0.23)},
		},
		{
			name: "valid source update",
			req:  UpdateExchangeRateRequest{Source: ptrStr("manual")},
		},
		{
			name: "valid both fields",
			req:  UpdateExchangeRateRequest{Rate: ptrFloat(0.23), Source: ptrStr("nbp")},
		},
		{
			name:    "zero rate",
			req:     UpdateExchangeRateRequest{Rate: ptrFloat(0)},
			wantErr: "rate must be positive",
		},
		{
			name:    "negative rate",
			req:     UpdateExchangeRateRequest{Rate: ptrFloat(-0.5)},
			wantErr: "rate must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── ConvertAmountRequest.Validate ──────────────────────────────────────────

func TestConvertAmountRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ConvertAmountRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  ConvertAmountRequest{Amount: 100.0, From: "PLN", To: "EUR"},
		},
		{
			name: "valid zero amount",
			req:  ConvertAmountRequest{Amount: 0, From: "PLN", To: "EUR"},
		},
		{
			name:    "negative amount",
			req:     ConvertAmountRequest{Amount: -1, From: "PLN", To: "EUR"},
			wantErr: "amount must be non-negative",
		},
		{
			name:    "missing from",
			req:     ConvertAmountRequest{Amount: 100, To: "EUR"},
			wantErr: "from currency is required",
		},
		{
			name:    "missing to",
			req:     ConvertAmountRequest{Amount: 100, From: "PLN"},
			wantErr: "to currency is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── CreateStocktakeRequest.Validate ────────────────────────────────────────

func TestCreateStocktakeRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateStocktakeRequest
		wantErr string
	}{
		{
			name: "valid minimal",
			req:  CreateStocktakeRequest{WarehouseID: uuid.New(), Name: "January Count"},
		},
		{
			name: "valid with optional fields",
			req: func() CreateStocktakeRequest {
				notes := "Full warehouse count"
				return CreateStocktakeRequest{
					WarehouseID: uuid.New(),
					Name:        "Q1 Stocktake",
					Notes:       &notes,
					ProductIDs:  []uuid.UUID{uuid.New(), uuid.New()},
				}
			}(),
		},
		{
			name:    "missing warehouse_id",
			req:     CreateStocktakeRequest{Name: "Count"},
			wantErr: "warehouse_id is required",
		},
		{
			name:    "nil warehouse_id",
			req:     CreateStocktakeRequest{WarehouseID: uuid.Nil, Name: "Count"},
			wantErr: "warehouse_id is required",
		},
		{
			name:    "missing name",
			req:     CreateStocktakeRequest{WarehouseID: uuid.New()},
			wantErr: "name is required",
		},
		{
			name:    "whitespace-only name",
			req:     CreateStocktakeRequest{WarehouseID: uuid.New(), Name: "   "},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			req:     CreateStocktakeRequest{WarehouseID: uuid.New(), Name: strings.Repeat("x", 501)},
			wantErr: "name",
		},
		{
			name: "name at max length (500)",
			req:  CreateStocktakeRequest{WarehouseID: uuid.New(), Name: strings.Repeat("x", 500)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── UpdateStocktakeItemRequest.Validate ────────────────────────────────────

func TestUpdateStocktakeItemRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateStocktakeItemRequest
		wantErr string
	}{
		{
			name: "valid",
			req:  UpdateStocktakeItemRequest{CountedQuantity: 50},
		},
		{
			name: "valid zero count",
			req:  UpdateStocktakeItemRequest{CountedQuantity: 0},
		},
		{
			name: "valid with notes",
			req:  UpdateStocktakeItemRequest{CountedQuantity: 10, Notes: ptrStr("Recounted twice")},
		},
		{
			name:    "negative counted_quantity",
			req:     UpdateStocktakeItemRequest{CountedQuantity: -1},
			wantErr: "counted_quantity must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── PackOrderRequest.Validate ──────────────────────────────────────────────

func TestPackOrderRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     PackOrderRequest
		wantErr string
	}{
		{
			name: "valid single item",
			req:  PackOrderRequest{ScannedItems: []ScannedItem{{SKU: "WDG-001", Quantity: 2}}},
		},
		{
			name: "valid multiple items",
			req: PackOrderRequest{ScannedItems: []ScannedItem{
				{SKU: "WDG-001", Quantity: 2},
				{SKU: "WDG-002", Quantity: 1},
			}},
		},
		{
			name:    "empty scanned_items",
			req:     PackOrderRequest{},
			wantErr: "scanned_items is required and must not be empty",
		},
		{
			name:    "item missing sku",
			req:     PackOrderRequest{ScannedItems: []ScannedItem{{SKU: "", Quantity: 1}}},
			wantErr: "each scanned_item must have a sku",
		},
		{
			name:    "item zero quantity",
			req:     PackOrderRequest{ScannedItems: []ScannedItem{{SKU: "WDG-001", Quantity: 0}}},
			wantErr: "each scanned_item must have a positive quantity",
		},
		{
			name:    "item negative quantity",
			req:     PackOrderRequest{ScannedItems: []ScannedItem{{SKU: "WDG-001", Quantity: -1}}},
			wantErr: "each scanned_item must have a positive quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
