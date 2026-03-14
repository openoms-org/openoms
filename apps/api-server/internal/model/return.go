package model

import (
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Return represents a customer return request for an order.
type Return struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	OrderID       uuid.UUID       `json:"order_id"`
	Status        string          `json:"status"`
	Reason        string          `json:"reason"`
	Items         json.RawMessage `json:"items"`
	RefundAmount  float64         `json:"refund_amount"`
	Notes         *string         `json:"notes,omitempty"`
	ReturnToken   *string         `json:"return_token,omitempty"`
	CustomerEmail *string         `json:"customer_email,omitempty"`
	CustomerNotes *string         `json:"customer_notes,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// PublicReturnRequest is the request body for creating a return via the public self-service endpoint.
type PublicReturnRequest struct {
	OrderID string          `json:"order_id"`
	Email   string          `json:"email"`
	Items   json.RawMessage `json:"items,omitempty"`
	Reason  string          `json:"reason"`
	Notes   string          `json:"notes,omitempty"`
}

// Validate validates the public return request.
func (r PublicReturnRequest) Validate() error {
	if r.OrderID == "" {
		return errors.New("order_id is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Reason == "" {
		return errors.New("reason is required")
	}
	if err := validateMaxLength("reason", r.Reason, 2000); err != nil {
		return err
	}
	if err := validateMaxLength("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

// CreateReturnRequest is the payload for creating an internal return.
type CreateReturnRequest struct {
	OrderID      uuid.UUID       `json:"order_id"`
	Reason       string          `json:"reason"`
	Items        json.RawMessage `json:"items,omitempty"`
	RefundAmount float64         `json:"refund_amount"`
	Notes        *string         `json:"notes,omitempty"`
}

// Validate validates the create return request.
func (r CreateReturnRequest) Validate() error {
	if r.OrderID == uuid.Nil {
		return errors.New("order_id is required")
	}
	if r.Reason == "" {
		return errors.New("reason is required")
	}
	if r.RefundAmount < 0 {
		return errors.New("refund_amount must be non-negative")
	}
	if err := validateMaxLength("reason", r.Reason, 2000); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

// UpdateReturnRequest is the payload for updating an existing return.
type UpdateReturnRequest struct {
	Reason       *string          `json:"reason,omitempty"`
	Items        *json.RawMessage `json:"items,omitempty"`
	RefundAmount *float64         `json:"refund_amount,omitempty"`
	Notes        *string          `json:"notes,omitempty"`
}

// Validate validates the update return request.
func (r UpdateReturnRequest) Validate() error {
	if r.Reason == nil && r.Items == nil && r.RefundAmount == nil && r.Notes == nil {
		return errors.New("at least one field must be provided")
	}
	if r.RefundAmount != nil && *r.RefundAmount < 0 {
		return errors.New("refund_amount must be non-negative")
	}
	return nil
}

// ReturnStatusRequest is the payload for transitioning a return status.
type ReturnStatusRequest struct {
	Status string `json:"status"`
}

// ReturnListFilter holds query parameters for listing returns.
type ReturnListFilter struct {
	Status  *string
	OrderID *uuid.UUID
	PaginationParams
}

// Valid return status transitions
var returnTransitions = map[string][]string{
	"requested": {"approved", "rejected", "cancelled"},
	"approved":  {"received", "cancelled"},
	"received":  {"refunded", "cancelled"},
}

// IsValidReturnTransition reports whether transitioning from one return status to another is allowed.
func IsValidReturnTransition(from, to string) bool {
	allowed, ok := returnTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}
