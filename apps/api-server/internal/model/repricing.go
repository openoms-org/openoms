package model

import (
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// RepricingRule represents a dynamic pricing rule.
type RepricingRule struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	Name             string          `json:"name"`
	Status           string          `json:"status"`
	Strategy         string          `json:"strategy"`
	Priority         int             `json:"priority"`
	ScopeType        string          `json:"scope_type"`
	ScopeValue       json.RawMessage `json:"scope_value,omitempty"`
	Params           json.RawMessage `json:"params"`
	MinPrice         *float64        `json:"min_price,omitempty"`
	MaxPrice         *float64        `json:"max_price,omitempty"`
	Channels         json.RawMessage `json:"channels"`
	LastAppliedAt    *time.Time      `json:"last_applied_at,omitempty"`
	ProductsAffected int             `json:"products_affected"`
	CreatedBy        *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// RepricingLog represents a single price change entry.
type RepricingLog struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	RuleID    uuid.UUID `json:"rule_id"`
	ProductID uuid.UUID `json:"product_id"`
	OldPrice  float64   `json:"old_price"`
	NewPrice  float64   `json:"new_price"`
	Reason    *string   `json:"reason,omitempty"`
	Channel   string    `json:"channel"`
	AppliedAt time.Time `json:"applied_at"`
}

// Strategy parameter structs.

// MarginParams holds margin-based repricing strategy parameters.
type MarginParams struct {
	MinMarginPct    float64 `json:"min_margin_pct"`
	TargetMarginPct float64 `json:"target_margin_pct"`
	CostField       string  `json:"cost_field"`
}

// TimeBasedParams holds configuration for the time-based repricing strategy.
type TimeBasedParams struct {
	Schedule []TimeScheduleEntry `json:"schedule"`
}

// TimeScheduleEntry defines a discount for a specific time window.
type TimeScheduleEntry struct {
	Days        string  `json:"days"`
	Hours       string  `json:"hours,omitempty"`
	DiscountPct float64 `json:"discount_pct"`
}

// StockBasedParams holds configuration for the stock-based repricing strategy.
type StockBasedParams struct {
	LowStockThreshold    int     `json:"low_stock_threshold"`
	LowStockMarkupPct    float64 `json:"low_stock_markup_pct"`
	HighStockThreshold   int     `json:"high_stock_threshold"`
	HighStockDiscountPct float64 `json:"high_stock_discount_pct"`
}

// Request types.

var validStrategies = []string{"margin", "time_based", "stock_based"}
var validStatuses = []string{"active", "paused", "archived"}
var validScopeTypes = []string{"all", "category", "tag", "product_ids"}

// CreateRepricingRuleRequest is the payload for creating a new repricing rule.
type CreateRepricingRuleRequest struct {
	Name       string          `json:"name"`
	Strategy   string          `json:"strategy"`
	Priority   int             `json:"priority"`
	ScopeType  string          `json:"scope_type"`
	ScopeValue json.RawMessage `json:"scope_value,omitempty"`
	Params     json.RawMessage `json:"params"`
	MinPrice   *float64        `json:"min_price,omitempty"`
	MaxPrice   *float64        `json:"max_price,omitempty"`
	Channels   json.RawMessage `json:"channels,omitempty"`
}

// Validate validates the create repricing rule request.
func (r CreateRepricingRuleRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 200); err != nil {
		return err
	}
	if !slices.Contains(validStrategies, r.Strategy) {
		return errors.New("strategy must be one of: margin, time_based, stock_based")
	}
	if !slices.Contains(validScopeTypes, r.ScopeType) {
		return errors.New("scope_type must be one of: all, category, tag, product_ids")
	}
	if r.MinPrice != nil && *r.MinPrice < 0 {
		return errors.New("min_price must be non-negative")
	}
	if r.MaxPrice != nil && *r.MaxPrice < 0 {
		return errors.New("max_price must be non-negative")
	}
	if r.MinPrice != nil && r.MaxPrice != nil && *r.MinPrice > *r.MaxPrice {
		return errors.New("min_price must not exceed max_price")
	}
	if r.Priority < 0 || r.Priority > 1000 {
		return errors.New("priority must be between 0 and 1000")
	}
	return nil
}

// UpdateRepricingRuleRequest is the payload for updating an existing repricing rule.
type UpdateRepricingRuleRequest struct {
	Name       *string          `json:"name,omitempty"`
	Status     *string          `json:"status,omitempty"`
	Strategy   *string          `json:"strategy,omitempty"`
	Priority   *int             `json:"priority,omitempty"`
	ScopeType  *string          `json:"scope_type,omitempty"`
	ScopeValue *json.RawMessage `json:"scope_value,omitempty"`
	Params     *json.RawMessage `json:"params,omitempty"`
	MinPrice   *float64         `json:"min_price,omitempty"`
	MaxPrice   *float64         `json:"max_price,omitempty"`
	Channels   *json.RawMessage `json:"channels,omitempty"`
}

// Validate validates the update repricing rule request.
func (r UpdateRepricingRuleRequest) Validate() error {
	if r.Name == nil && r.Status == nil && r.Strategy == nil && r.Priority == nil &&
		r.ScopeType == nil && r.ScopeValue == nil && r.Params == nil &&
		r.MinPrice == nil && r.MaxPrice == nil && r.Channels == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil && *r.Name == "" {
		return errors.New("name must not be empty")
	}
	if r.Status != nil && !slices.Contains(validStatuses, *r.Status) {
		return errors.New("status must be one of: active, paused, archived")
	}
	if r.Strategy != nil && !slices.Contains(validStrategies, *r.Strategy) {
		return errors.New("strategy must be one of: margin, time_based, stock_based")
	}
	if r.ScopeType != nil && !slices.Contains(validScopeTypes, *r.ScopeType) {
		return errors.New("scope_type must be one of: all, category, tag, product_ids")
	}
	if r.Priority != nil && (*r.Priority < 0 || *r.Priority > 1000) {
		return errors.New("priority must be between 0 and 1000")
	}
	return nil
}

// RepricingRuleListFilter holds query parameters for listing repricing rules.
type RepricingRuleListFilter struct {
	Status   *string
	Strategy *string
	PaginationParams
}

// RepricingSummary provides dashboard-level stats.
type RepricingSummary struct {
	ActiveRules   int     `json:"active_rules"`
	PausedRules   int     `json:"paused_rules"`
	ChangesToday  int     `json:"changes_today"`
	ChangesWeek   int     `json:"changes_week"`
	AvgChangePct  float64 `json:"avg_change_pct"`
	TotalAffected int     `json:"total_affected"`
}

// SimulationResult shows what a rule would change without applying.
type SimulationResult struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	SKU         *string   `json:"sku,omitempty"`
	OldPrice    float64   `json:"old_price"`
	NewPrice    float64   `json:"new_price"`
	ChangePct   float64   `json:"change_pct"`
	Reason      string    `json:"reason"`
}

// ApplyResult summarizes the outcome of applying repricing rules.
type ApplyResult struct {
	RulesEvaluated   int `json:"rules_evaluated"`
	ProductsAffected int `json:"products_affected"`
	PriceChanges     int `json:"price_changes"`
}
