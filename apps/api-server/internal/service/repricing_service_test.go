package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// helper: create a minimal RepricingService with only a logger (pure functions need nothing else).
func newTestRepricingService() *RepricingService {
	return &RepricingService{
		logger: slog.Default(),
	}
}

func float64Ptr(v float64) *float64 { return &v }

func makeProduct(price float64, stockQty int, metadata json.RawMessage) model.Product {
	return model.Product{
		Price:         price,
		StockQuantity: stockQty,
		Metadata:      metadata,
	}
}

func makeRule(strategy string, params json.RawMessage, minPrice, maxPrice *float64) model.RepricingRule {
	return model.RepricingRule{
		Strategy: strategy,
		Params:   params,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}
}

// ---------------------------------------------------------------------------
// calculatePrice
// ---------------------------------------------------------------------------

func TestRepricingCalculatePrice_Margin(t *testing.T) {
	svc := newTestRepricingService()
	product := makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 50.0}`))
	rule := makeRule("margin", json.RawMessage(`{"target_margin_pct": 30, "cost_field": "supplier_cost"}`), nil, nil)

	newPrice, reason, skip := svc.calculatePrice(product, rule)
	assert.False(t, skip)
	assert.InDelta(t, 65.0, newPrice, 0.01)
	assert.Contains(t, reason, "Margin strategy")
}

func TestRepricingCalculatePrice_StockBased(t *testing.T) {
	svc := newTestRepricingService()
	product := makeProduct(100, 2, nil)
	rule := makeRule("stock_based", json.RawMessage(`{
		"low_stock_threshold": 5,
		"low_stock_markup_pct": 20,
		"high_stock_threshold": 100,
		"high_stock_discount_pct": 10
	}`), nil, nil)

	newPrice, reason, skip := svc.calculatePrice(product, rule)
	assert.False(t, skip)
	assert.InDelta(t, 120.0, newPrice, 0.01)
	assert.Contains(t, reason, "low stock")
}

func TestRepricingCalculatePrice_UnknownStrategy(t *testing.T) {
	svc := newTestRepricingService()
	product := makeProduct(100, 10, nil)
	rule := makeRule("unknown_strategy", json.RawMessage(`{}`), nil, nil)

	_, _, skip := svc.calculatePrice(product, rule)
	assert.True(t, skip, "unknown strategy should be skipped")
}

func TestRepricingCalculatePrice_EmptyStrategy(t *testing.T) {
	svc := newTestRepricingService()
	product := makeProduct(100, 10, nil)
	rule := makeRule("", json.RawMessage(`{}`), nil, nil)

	_, _, skip := svc.calculatePrice(product, rule)
	assert.True(t, skip, "empty strategy should be skipped")
}

// Create must reject the removed "competitive" strategy at validation time,
// before any repository/DB access — so no rule row is ever inserted.
func TestRepricingCreate_RejectsCompetitive(t *testing.T) {
	// nil pool/repos are safe here: Validate() fails first and Create returns early.
	svc := newTestRepricingService()
	req := model.CreateRepricingRuleRequest{
		Name:      "Konkurencyjna",
		Strategy:  "competitive",
		ScopeType: "all",
		Priority:  10,
	}

	rule, err := svc.Create(context.Background(), uuid.New(), req, uuid.New(), "127.0.0.1")

	require.Nil(t, rule, "no rule should be returned (none inserted) for a rejected strategy")
	require.Error(t, err)
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr, "competitive must be rejected as a validation error (HTTP 400)")
	assert.Contains(t, err.Error(), "strategy must be one of")
}

// ---------------------------------------------------------------------------
// applyMarginStrategy
// ---------------------------------------------------------------------------

func TestApplyMarginStrategy(t *testing.T) {
	svc := newTestRepricingService()

	tests := []struct {
		name      string
		product   model.Product
		params    string
		wantPrice float64
		wantSkip  bool
		checkMsg  string
	}{
		{
			name:      "basic 30% margin on cost 50",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 50.0}`)),
			params:    `{"target_margin_pct": 30, "cost_field": "supplier_cost"}`,
			wantPrice: 65.0, // 50 * (1 + 30/100) = 65
			wantSkip:  false,
			checkMsg:  "cost=50.00",
		},
		{
			name:      "zero margin",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 80.0}`)),
			params:    `{"target_margin_pct": 0, "cost_field": "supplier_cost"}`,
			wantPrice: 80.0, // 80 * (1 + 0) = 80
			wantSkip:  false,
			checkMsg:  "target margin=0.0%",
		},
		{
			name:      "100% margin doubles the cost",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 40.0}`)),
			params:    `{"target_margin_pct": 100, "cost_field": "supplier_cost"}`,
			wantPrice: 80.0, // 40 * (1 + 100/100) = 80
			wantSkip:  false,
		},
		{
			name:      "negative margin (below cost)",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 80.0}`)),
			params:    `{"target_margin_pct": -20, "cost_field": "supplier_cost"}`,
			wantPrice: 64.0, // 80 * (1 + -20/100) = 80 * 0.8 = 64
			wantSkip:  false,
		},
		{
			name:     "zero cost price - skip",
			product:  makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 0}`)),
			params:   `{"target_margin_pct": 30, "cost_field": "supplier_cost"}`,
			wantSkip: true,
		},
		{
			name:     "negative cost price - skip",
			product:  makeProduct(100, 10, json.RawMessage(`{"supplier_cost": -10}`)),
			params:   `{"target_margin_pct": 30, "cost_field": "supplier_cost"}`,
			wantSkip: true,
		},
		{
			name:     "nil metadata - skip",
			product:  makeProduct(100, 10, nil),
			params:   `{"target_margin_pct": 30, "cost_field": "supplier_cost"}`,
			wantSkip: true,
		},
		{
			name:     "missing cost field in metadata - skip",
			product:  makeProduct(100, 10, json.RawMessage(`{"other_field": 50.0}`)),
			params:   `{"target_margin_pct": 30, "cost_field": "supplier_cost"}`,
			wantSkip: true,
		},
		{
			name:      "default cost field (empty string uses supplier_cost)",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 60.0}`)),
			params:    `{"target_margin_pct": 25}`,
			wantPrice: 75.0, // 60 * (1 + 25/100) = 75
			wantSkip:  false,
		},
		{
			name:      "custom cost field name",
			product:   makeProduct(100, 10, json.RawMessage(`{"my_cost": 40.0}`)),
			params:    `{"target_margin_pct": 50, "cost_field": "my_cost"}`,
			wantPrice: 60.0, // 40 * (1 + 50/100) = 60
			wantSkip:  false,
		},
		{
			name:     "invalid params JSON - skip",
			product:  makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 50.0}`)),
			params:   `{invalid json`,
			wantSkip: true,
		},
		{
			name:      "very large margin",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 10.0}`)),
			params:    `{"target_margin_pct": 1000, "cost_field": "supplier_cost"}`,
			wantPrice: 110.0, // 10 * (1 + 1000/100) = 10 * 11 = 110
			wantSkip:  false,
		},
		{
			name:      "fractional margin",
			product:   makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 100.0}`)),
			params:    `{"target_margin_pct": 15.5, "cost_field": "supplier_cost"}`,
			wantPrice: 115.5, // 100 * (1 + 15.5/100) = 115.5
			wantSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := makeRule("margin", json.RawMessage(tt.params), nil, nil)
			newPrice, reason, skip := svc.applyMarginStrategy(tt.product, rule)
			assert.Equal(t, tt.wantSkip, skip)
			if !skip {
				assert.InDelta(t, tt.wantPrice, newPrice, 0.001)
				assert.Contains(t, reason, "Margin strategy")
				if tt.checkMsg != "" {
					assert.Contains(t, reason, tt.checkMsg)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getCostFromMetadata
// ---------------------------------------------------------------------------

func TestGetCostFromMetadata(t *testing.T) {
	svc := newTestRepricingService()

	tests := []struct {
		name      string
		metadata  json.RawMessage
		costField string
		want      float64
	}{
		{
			name:      "float64 value",
			metadata:  json.RawMessage(`{"supplier_cost": 42.5}`),
			costField: "supplier_cost",
			want:      42.5,
		},
		{
			name:      "integer value (unmarshals as float64)",
			metadata:  json.RawMessage(`{"supplier_cost": 100}`),
			costField: "supplier_cost",
			want:      100.0,
		},
		{
			name:      "zero value",
			metadata:  json.RawMessage(`{"supplier_cost": 0}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "negative value",
			metadata:  json.RawMessage(`{"supplier_cost": -15.5}`),
			costField: "supplier_cost",
			want:      -15.5,
		},
		{
			name:      "string value returns 0 (not a number type)",
			metadata:  json.RawMessage(`{"supplier_cost": "50.00"}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "boolean value returns 0",
			metadata:  json.RawMessage(`{"supplier_cost": true}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "null value returns 0",
			metadata:  json.RawMessage(`{"supplier_cost": null}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "nil metadata returns 0",
			metadata:  nil,
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "missing key returns 0",
			metadata:  json.RawMessage(`{"other_field": 42.5}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "empty costField defaults to supplier_cost",
			metadata:  json.RawMessage(`{"supplier_cost": 77.0}`),
			costField: "",
			want:      77.0,
		},
		{
			name:      "custom costField",
			metadata:  json.RawMessage(`{"purchase_price": 33.33}`),
			costField: "purchase_price",
			want:      33.33,
		},
		{
			name:      "invalid JSON returns 0",
			metadata:  json.RawMessage(`{not valid}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "empty JSON object",
			metadata:  json.RawMessage(`{}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "nested object value returns 0",
			metadata:  json.RawMessage(`{"supplier_cost": {"value": 50}}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "array value returns 0",
			metadata:  json.RawMessage(`{"supplier_cost": [50]}`),
			costField: "supplier_cost",
			want:      0,
		},
		{
			name:      "very large float64",
			metadata:  json.RawMessage(`{"supplier_cost": 9999999.99}`),
			costField: "supplier_cost",
			want:      9999999.99,
		},
		{
			name:      "very small positive float64",
			metadata:  json.RawMessage(`{"supplier_cost": 0.01}`),
			costField: "supplier_cost",
			want:      0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := model.Product{Metadata: tt.metadata}
			got := svc.getCostFromMetadata(product, tt.costField)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

// ---------------------------------------------------------------------------
// applyStockStrategy
// ---------------------------------------------------------------------------

func TestApplyStockStrategy(t *testing.T) {
	svc := newTestRepricingService()

	tests := []struct {
		name      string
		price     float64
		stockQty  int
		params    string
		wantPrice float64
		wantSkip  bool
		checkMsg  string
	}{
		{
			name:     "stock below low threshold - markup",
			price:    100.0,
			stockQty: 3,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantPrice: 120.0, // 100 * (1 + 20/100)
			wantSkip:  false,
			checkMsg:  "low stock",
		},
		{
			name:     "stock above high threshold - discount",
			price:    100.0,
			stockQty: 150,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 15
			}`,
			wantPrice: 85.0, // 100 * (1 - 15/100)
			wantSkip:  false,
			checkMsg:  "high stock",
		},
		{
			name:     "stock in normal range - skip",
			price:    100.0,
			stockQty: 50,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantSkip: true,
		},
		{
			name:     "stock exactly at low threshold - normal range (not < threshold)",
			price:    100.0,
			stockQty: 5,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantSkip: true,
		},
		{
			name:     "stock exactly at high threshold - normal range (not > threshold)",
			price:    100.0,
			stockQty: 100,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantSkip: true,
		},
		{
			name:     "stock one below low threshold",
			price:    100.0,
			stockQty: 4,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantPrice: 120.0,
			wantSkip:  false,
		},
		{
			name:     "stock one above high threshold",
			price:    100.0,
			stockQty: 101,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantPrice: 90.0,
			wantSkip:  false,
		},
		{
			name:     "zero stock below threshold",
			price:    100.0,
			stockQty: 0,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 25,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantPrice: 125.0,
			wantSkip:  false,
		},
		{
			name:     "zero thresholds - skip (threshold <= 0 means condition disabled)",
			price:    100.0,
			stockQty: 0,
			params: `{
				"low_stock_threshold": 0,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 0,
				"high_stock_discount_pct": 10
			}`,
			wantSkip: true,
		},
		{
			name:     "zero markup percentage (no actual change)",
			price:    100.0,
			stockQty: 2,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 0,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 10
			}`,
			wantPrice: 100.0, // 100 * (1 + 0) = 100 (no change, but not skipped)
			wantSkip:  false,
		},
		{
			name:     "zero discount percentage (no actual change)",
			price:    100.0,
			stockQty: 200,
			params: `{
				"low_stock_threshold": 5,
				"low_stock_markup_pct": 20,
				"high_stock_threshold": 100,
				"high_stock_discount_pct": 0
			}`,
			wantPrice: 100.0, // 100 * (1 - 0) = 100 (no change, but not skipped)
			wantSkip:  false,
		},
		{
			name:     "invalid params JSON - skip",
			price:    100.0,
			stockQty: 2,
			params:   `{invalid`,
			wantSkip: true,
		},
		{
			name:     "low stock with low-priced product",
			price:    0.50,
			stockQty: 1,
			params: `{
				"low_stock_threshold": 10,
				"low_stock_markup_pct": 100,
				"high_stock_threshold": 200,
				"high_stock_discount_pct": 50
			}`,
			wantPrice: 1.0, // 0.50 * (1 + 100/100) = 1.0
			wantSkip:  false,
		},
		{
			name:     "low stock takes priority over high stock when both match (stock < low && stock > high is impossible, but low < high check)",
			price:    100.0,
			stockQty: 3,
			params: `{
				"low_stock_threshold": 10,
				"low_stock_markup_pct": 10,
				"high_stock_threshold": 2,
				"high_stock_discount_pct": 20
			}`,
			wantPrice: 110.0, // low stock check runs first: 3 < 10 = true
			wantSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := makeProduct(tt.price, tt.stockQty, nil)
			rule := makeRule("stock_based", json.RawMessage(tt.params), nil, nil)
			newPrice, reason, skip := svc.applyStockStrategy(product, rule)
			assert.Equal(t, tt.wantSkip, skip)
			if !skip {
				assert.InDelta(t, tt.wantPrice, newPrice, 0.001)
				if tt.checkMsg != "" {
					assert.Contains(t, reason, tt.checkMsg)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// applyTimeStrategy
// ---------------------------------------------------------------------------

func TestApplyTimeStrategy(t *testing.T) {
	svc := newTestRepricingService()
	now := time.Now()
	currentDay := strings.ToLower(now.Weekday().String()[:3])
	currentHour := now.Hour()

	tests := []struct {
		name      string
		price     float64
		params    model.TimeBasedParams
		wantPrice float64
		wantSkip  bool
		checkMsg  string
	}{
		{
			name:  "matching day, no hours restriction",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, DiscountPct: 10},
				},
			},
			wantPrice: 90.0, // 100 * (1 - 10/100)
			wantSkip:  false,
			checkMsg:  "Time-based",
		},
		{
			name:  "matching day, current hour within range",
			price: 200.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, Hours: fmt.Sprintf("%d-%d", currentHour, currentHour), DiscountPct: 15},
				},
			},
			wantPrice: 170.0, // 200 * (1 - 15/100)
			wantSkip:  false,
		},
		{
			name:  "matching day, current hour outside range - skip",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, Hours: func() string {
						// Pick an hour that is definitely not the current hour
						otherHour := (currentHour + 12) % 24
						return fmt.Sprintf("%d-%d", otherHour, otherHour)
					}(), DiscountPct: 10},
				},
			},
			wantSkip: true,
		},
		{
			name:  "non-matching day - skip",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: func() string {
						// Pick a day that is definitely not today
						days := []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
						for _, d := range days {
							if d != currentDay {
								return d
							}
						}
						return "mon"
					}(), DiscountPct: 10},
				},
			},
			wantSkip: true,
		},
		{
			name:  "empty schedule - skip",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{},
			},
			wantSkip: true,
		},
		{
			name:  "nil schedule - skip",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: nil,
			},
			wantSkip: true,
		},
		{
			name:  "multiple schedules, first matches",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, DiscountPct: 20},
					{Days: currentDay, DiscountPct: 5},
				},
			},
			wantPrice: 80.0, // first match wins: 100 * (1 - 20/100) = 80
			wantSkip:  false,
		},
		{
			name:  "multiple days in comma-separated list",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: "mon,tue,wed,thu,fri,sat,sun", DiscountPct: 12},
				},
			},
			wantPrice: 88.0, // 100 * (1 - 12/100) = 88
			wantSkip:  false,
		},
		{
			name:  "days with spaces around commas",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: fmt.Sprintf(" %s , %s ", currentDay, "xyz"), DiscountPct: 5},
				},
			},
			wantPrice: 95.0,
			wantSkip:  false,
		},
		{
			name:  "zero discount percentage",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, DiscountPct: 0},
				},
			},
			wantPrice: 100.0, // 100 * (1 - 0) = 100
			wantSkip:  false,
		},
		{
			name:  "negative discount (acts as markup)",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, DiscountPct: -10},
				},
			},
			wantPrice: 110.0, // 100 * (1 - (-10/100)) = 100 * 1.10 = 110
			wantSkip:  false,
		},
		{
			name:  "100% discount (price goes to zero)",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, DiscountPct: 100},
				},
			},
			wantPrice: 0.0, // 100 * (1 - 100/100) = 0
			wantSkip:  false,
		},
		{
			name:  "hours range 0-23 (all day)",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, Hours: "0-23", DiscountPct: 10},
				},
			},
			wantPrice: 90.0,
			wantSkip:  false,
		},
		{
			name:  "case-insensitive day matching (uppercase)",
			price: 100.0,
			params: model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: strings.ToUpper(currentDay), DiscountPct: 10},
				},
			},
			wantPrice: 90.0,
			wantSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := makeProduct(tt.price, 10, nil)
			paramsJSON, err := json.Marshal(tt.params)
			assert.NoError(t, err)

			rule := makeRule("time_based", paramsJSON, nil, nil)
			newPrice, reason, skip := svc.applyTimeStrategy(product, rule)
			assert.Equal(t, tt.wantSkip, skip, "skip mismatch")
			if !skip {
				assert.InDelta(t, tt.wantPrice, newPrice, 0.001)
				if tt.checkMsg != "" {
					assert.Contains(t, reason, tt.checkMsg)
				}
			}
		})
	}
}

func TestApplyTimeStrategy_InvalidParams(t *testing.T) {
	svc := newTestRepricingService()
	product := makeProduct(100, 10, nil)
	rule := makeRule("time_based", json.RawMessage(`{invalid`), nil, nil)

	_, _, skip := svc.applyTimeStrategy(product, rule)
	assert.True(t, skip, "invalid JSON params should cause skip")
}

func TestApplyTimeStrategy_HourBoundaries(t *testing.T) {
	svc := newTestRepricingService()
	now := time.Now()
	currentDay := strings.ToLower(now.Weekday().String()[:3])
	currentHour := now.Hour()

	tests := []struct {
		name     string
		hours    string
		wantSkip bool
	}{
		{
			name:     "current hour at range start",
			hours:    fmt.Sprintf("%d-%d", currentHour, 23),
			wantSkip: false,
		},
		{
			name:     "current hour at range end",
			hours:    fmt.Sprintf("0-%d", currentHour),
			wantSkip: false,
		},
		{
			name:     "range one hour before current",
			hours:    fmt.Sprintf("0-%d", max(0, currentHour-1)),
			wantSkip: currentHour > 0, // if currentHour == 0, 0-0 range includes 0
		},
		{
			name:  "range one hour after current",
			hours: fmt.Sprintf("%d-23", min(23, currentHour+1)),
			wantSkip: func() bool {
				startHour := min(23, currentHour+1)
				return currentHour < startHour
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: currentDay, Hours: tt.hours, DiscountPct: 10},
				},
			}
			paramsJSON, _ := json.Marshal(params)
			product := makeProduct(100, 10, nil)
			rule := makeRule("time_based", paramsJSON, nil, nil)
			_, _, skip := svc.applyTimeStrategy(product, rule)
			assert.Equal(t, tt.wantSkip, skip)
		})
	}
}

// Test each day of the week is recognized.
func TestApplyTimeStrategy_AllDaysOfWeek(t *testing.T) {
	svc := newTestRepricingService()
	now := time.Now()
	currentDay := strings.ToLower(now.Weekday().String()[:3])

	allDays := []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}
	for _, day := range allDays {
		t.Run("day_"+day, func(t *testing.T) {
			params := model.TimeBasedParams{
				Schedule: []model.TimeScheduleEntry{
					{Days: day, DiscountPct: 10},
				},
			}
			paramsJSON, _ := json.Marshal(params)
			product := makeProduct(100, 10, nil)
			rule := makeRule("time_based", paramsJSON, nil, nil)
			_, _, skip := svc.applyTimeStrategy(product, rule)

			if day == currentDay {
				assert.False(t, skip, "today (%s) should match", day)
			} else {
				assert.True(t, skip, "day %s should not match today (%s)", day, currentDay)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// clampPrice
// ---------------------------------------------------------------------------

func TestClampPrice(t *testing.T) {
	svc := newTestRepricingService()

	tests := []struct {
		name     string
		price    float64
		minPrice *float64
		maxPrice *float64
		want     float64
	}{
		{
			name:     "no clamping (nil bounds)",
			price:    100.0,
			minPrice: nil,
			maxPrice: nil,
			want:     100.0,
		},
		{
			name:     "price below min - clamped to min",
			price:    5.0,
			minPrice: float64Ptr(10.0),
			maxPrice: nil,
			want:     10.0,
		},
		{
			name:     "price above max - clamped to max",
			price:    200.0,
			minPrice: nil,
			maxPrice: float64Ptr(150.0),
			want:     150.0,
		},
		{
			name:     "price within range - no change",
			price:    100.0,
			minPrice: float64Ptr(50.0),
			maxPrice: float64Ptr(150.0),
			want:     100.0,
		},
		{
			name:     "price exactly at min",
			price:    50.0,
			minPrice: float64Ptr(50.0),
			maxPrice: float64Ptr(150.0),
			want:     50.0,
		},
		{
			name:     "price exactly at max",
			price:    150.0,
			minPrice: float64Ptr(50.0),
			maxPrice: float64Ptr(150.0),
			want:     150.0,
		},
		{
			name:     "min equals max - price forced to that value (below)",
			price:    50.0,
			minPrice: float64Ptr(100.0),
			maxPrice: float64Ptr(100.0),
			want:     100.0,
		},
		{
			name:     "min equals max - price forced to that value (above)",
			price:    200.0,
			minPrice: float64Ptr(100.0),
			maxPrice: float64Ptr(100.0),
			want:     100.0,
		},
		{
			name:     "min equals max - price already at that value",
			price:    100.0,
			minPrice: float64Ptr(100.0),
			maxPrice: float64Ptr(100.0),
			want:     100.0,
		},
		{
			name:     "min > max - maxPrice wins (max applied after min)",
			price:    120.0,
			minPrice: float64Ptr(200.0),
			maxPrice: float64Ptr(100.0),
			want:     100.0, // first clamped to 200 (min), then clamped to 100 (max)
		},
		{
			name:     "zero price with min bound",
			price:    0.0,
			minPrice: float64Ptr(10.0),
			maxPrice: nil,
			want:     10.0,
		},
		{
			name:     "negative price with min bound",
			price:    -5.0,
			minPrice: float64Ptr(0.0),
			maxPrice: nil,
			want:     0.0,
		},
		{
			name:     "zero min and max",
			price:    50.0,
			minPrice: float64Ptr(0.0),
			maxPrice: float64Ptr(0.0),
			want:     0.0,
		},
		{
			name:     "only min, price above",
			price:    100.0,
			minPrice: float64Ptr(50.0),
			maxPrice: nil,
			want:     100.0,
		},
		{
			name:     "only max, price below",
			price:    50.0,
			minPrice: nil,
			maxPrice: float64Ptr(100.0),
			want:     50.0,
		},
		{
			name:     "very large price clamped to max",
			price:    999999.99,
			minPrice: nil,
			maxPrice: float64Ptr(500.0),
			want:     500.0,
		},
		{
			name:     "very small price clamped to min",
			price:    0.001,
			minPrice: float64Ptr(1.0),
			maxPrice: nil,
			want:     1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.clampPrice(tt.price, tt.minPrice, tt.maxPrice)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: calculatePrice + clampPrice (as used in applyRule)
// ---------------------------------------------------------------------------

func TestRepricingCalculatePriceWithClamp(t *testing.T) {
	svc := newTestRepricingService()

	t.Run("margin price clamped to min", func(t *testing.T) {
		product := makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 10.0}`))
		minP := 20.0
		rule := makeRule("margin", json.RawMessage(`{"target_margin_pct": 10, "cost_field": "supplier_cost"}`), &minP, nil)

		newPrice, _, skip := svc.calculatePrice(product, rule)
		assert.False(t, skip)
		// 10 * 1.10 = 11, then clamp to min 20
		clamped := svc.clampPrice(newPrice, rule.MinPrice, rule.MaxPrice)
		assert.InDelta(t, 20.0, clamped, 0.001)
	})

	t.Run("margin price clamped to max", func(t *testing.T) {
		product := makeProduct(100, 10, json.RawMessage(`{"supplier_cost": 200.0}`))
		maxP := 250.0
		rule := makeRule("margin", json.RawMessage(`{"target_margin_pct": 50, "cost_field": "supplier_cost"}`), nil, &maxP)

		newPrice, _, skip := svc.calculatePrice(product, rule)
		assert.False(t, skip)
		// 200 * 1.50 = 300, then clamp to max 250
		clamped := svc.clampPrice(newPrice, rule.MinPrice, rule.MaxPrice)
		assert.InDelta(t, 250.0, clamped, 0.001)
	})

	t.Run("stock markup clamped between min and max", func(t *testing.T) {
		product := makeProduct(100, 1, nil)
		minP := 90.0
		maxP := 110.0
		rule := makeRule("stock_based", json.RawMessage(`{
			"low_stock_threshold": 5,
			"low_stock_markup_pct": 50,
			"high_stock_threshold": 100,
			"high_stock_discount_pct": 10
		}`), &minP, &maxP)

		newPrice, _, skip := svc.calculatePrice(product, rule)
		assert.False(t, skip)
		// 100 * (1 + 50/100) = 150, clamp to max 110
		clamped := svc.clampPrice(newPrice, rule.MinPrice, rule.MaxPrice)
		assert.InDelta(t, 110.0, clamped, 0.001)
	})
}
