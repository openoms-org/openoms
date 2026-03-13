package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CustomerSegment represents a group of customers defined by manual assignment or rules.
type CustomerSegment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Color         string          `json:"color"`
	SegmentType   string          `json:"segment_type"`
	Rules         json.RawMessage `json:"rules,omitempty"`
	CustomerCount int             `json:"customer_count"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// SegmentMember represents a customer's membership in a segment.
type SegmentMember struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	SegmentID  uuid.UUID `json:"segment_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	AddedAt    time.Time `json:"added_at"`
	// Joined from customers table for listing
	CustomerName  string  `json:"customer_name,omitempty"`
	CustomerEmail *string `json:"customer_email,omitempty"`
	TotalOrders   int     `json:"total_orders,omitempty"`
	TotalSpent    float64 `json:"total_spent,omitempty"`
}

// RFMScores represents a customer's Recency, Frequency, Monetary scores.
type RFMScores struct {
	Recency   int `json:"recency"`   // 1-5
	Frequency int `json:"frequency"` // 1-5
	Monetary  int `json:"monetary"`  // 1-5
}

// CustomerRFM is a customer with their RFM analysis scores and segment label.
type CustomerRFM struct {
	CustomerID    uuid.UUID `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail *string   `json:"customer_email,omitempty"`
	DaysSince     int       `json:"days_since_last_order"`
	OrderCount    int       `json:"order_count"`
	TotalSpent    float64   `json:"total_spent"`
	RFM           RFMScores `json:"rfm"`
	SegmentLabel  string    `json:"segment_label"`
}

// SegmentRules defines rule-based segment criteria stored in JSONB.
type SegmentRules struct {
	MinOrders          *int     `json:"min_orders,omitempty"`
	MinTotalSpent      *float64 `json:"min_total_spent,omitempty"`
	LastOrderWithinDay *int     `json:"last_order_within_days,omitempty"`
	MaxOrders          *int     `json:"max_orders,omitempty"`
	MaxTotalSpent      *float64 `json:"max_total_spent,omitempty"`
}

// CreateSegmentRequest is the payload for creating a segment.
type CreateSegmentRequest struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Color       string          `json:"color"`
	SegmentType string          `json:"segment_type"`
	Rules       json.RawMessage `json:"rules,omitempty"`
}

// Validate validates the create segment request.
func (r *CreateSegmentRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 255); err != nil {
		return err
	}
	validTypes := map[string]bool{"manual": true, "rfm_auto": true, "rule_based": true}
	if !validTypes[r.SegmentType] {
		return errors.New("segment_type must be 'manual', 'rfm_auto', or 'rule_based'")
	}
	if r.Color == "" {
		r.Color = "#6366f1"
	}
	return nil
}

// UpdateSegmentRequest is the payload for updating a segment.
type UpdateSegmentRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Color       *string         `json:"color,omitempty"`
	Rules       json.RawMessage `json:"rules,omitempty"`
}

// Validate validates the update segment request.
func (r *UpdateSegmentRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if r.Name != nil {
		if err := validateMaxLength("name", *r.Name, 255); err != nil {
			return err
		}
	}
	return nil
}

// AddSegmentMemberRequest adds a customer to a segment.
type AddSegmentMemberRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// Validate validates the add segment member request.
func (r *AddSegmentMemberRequest) Validate() error {
	if r.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}
	return nil
}

// SegmentListFilter for listing segments with pagination.
type SegmentListFilter struct {
	PaginationParams
}

// LoyaltyProgram represents a loyalty/discount program.
type LoyaltyProgram struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Name        string          `json:"name"`
	Status      string          `json:"status"`
	ProgramType string          `json:"program_type"`
	Config      json.RawMessage `json:"config"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CustomerLoyalty represents a customer's status in a loyalty program.
type CustomerLoyalty struct {
	TenantID            uuid.UUID  `json:"tenant_id"`
	CustomerID          uuid.UUID  `json:"customer_id"`
	ProgramID           uuid.UUID  `json:"program_id"`
	PointsBalance       int        `json:"points_balance"`
	TotalPointsEarned   int        `json:"total_points_earned"`
	TotalPointsRedeemed int        `json:"total_points_redeemed"`
	CurrentTier         *string    `json:"current_tier,omitempty"`
	TotalSpent          float64    `json:"total_spent"`
	OrderCount          int        `json:"order_count"`
	LastActivityAt      *time.Time `json:"last_activity_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// CustomerLoyaltyWithProgram joins loyalty status with program info.
type CustomerLoyaltyWithProgram struct {
	CustomerLoyalty
	ProgramName string `json:"program_name"`
	ProgramType string `json:"program_type"`
}

// LeaderboardEntry represents a customer's rank in a loyalty program.
type LeaderboardEntry struct {
	CustomerID   uuid.UUID `json:"customer_id"`
	CustomerName string    `json:"customer_name"`
	Points       int       `json:"points"`
	TotalSpent   float64   `json:"total_spent"`
	OrderCount   int       `json:"order_count"`
	CurrentTier  *string   `json:"current_tier,omitempty"`
}

// CreateLoyaltyProgramRequest is the payload for creating a loyalty program.
type CreateLoyaltyProgramRequest struct {
	Name        string          `json:"name"`
	ProgramType string          `json:"program_type"`
	Config      json.RawMessage `json:"config"`
}

// Validate validates the create loyalty program request.
func (r *CreateLoyaltyProgramRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 255); err != nil {
		return err
	}
	validTypes := map[string]bool{"points": true, "tier": true, "discount_after_n": true}
	if !validTypes[r.ProgramType] {
		return errors.New("program_type must be 'points', 'tier', or 'discount_after_n'")
	}
	if r.Config == nil {
		r.Config = json.RawMessage(`{}`)
	}
	return nil
}

// UpdateLoyaltyProgramRequest is the payload for updating a loyalty program.
type UpdateLoyaltyProgramRequest struct {
	Name   *string         `json:"name,omitempty"`
	Status *string         `json:"status,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// Validate validates the update loyalty program request.
func (r *UpdateLoyaltyProgramRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if r.Status != nil {
		validStatuses := map[string]bool{"active": true, "paused": true, "ended": true}
		if !validStatuses[*r.Status] {
			return errors.New("status must be 'active', 'paused', or 'ended'")
		}
	}
	return nil
}

// AwardPointsRequest is the payload for manually awarding points.
type AwardPointsRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Points     int       `json:"points"`
	Reason     string    `json:"reason"`
}

// Validate validates the award points request.
func (r *AwardPointsRequest) Validate() error {
	if r.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}
	if r.Points <= 0 {
		return errors.New("points must be greater than 0")
	}
	return nil
}

// RedeemPointsRequest is the payload for redeeming points.
type RedeemPointsRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Points     int       `json:"points"`
}

// Validate validates the redeem points request.
func (r *RedeemPointsRequest) Validate() error {
	if r.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}
	if r.Points <= 0 {
		return errors.New("points must be greater than 0")
	}
	return nil
}

// LoyaltyProgramListFilter for listing programs with pagination.
type LoyaltyProgramListFilter struct {
	PaginationParams
}
