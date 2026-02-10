package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID              uuid.UUID        `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	ExternalID      *string          `json:"external_id,omitempty"`
	Source          string           `json:"source"`
	IntegrationID   *uuid.UUID       `json:"integration_id,omitempty"`
	Status          string           `json:"status"`
	CustomerName    string           `json:"customer_name"`
	CustomerEmail   *string          `json:"customer_email,omitempty"`
	CustomerPhone   *string          `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage  `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage  `json:"billing_address,omitempty"`
	Items           json.RawMessage  `json:"items,omitempty"`
	TotalAmount     float64          `json:"total_amount"`
	Currency        string           `json:"currency"`
	Notes           *string          `json:"notes,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
	OrderedAt       *time.Time       `json:"ordered_at,omitempty"`
	ShippedAt       *time.Time       `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time       `json:"delivered_at,omitempty"`
	PaymentStatus   string           `json:"payment_status"`
	PaymentMethod   *string          `json:"payment_method,omitempty"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CreateOrderRequest struct {
	ExternalID      *string          `json:"external_id,omitempty"`
	Source          string           `json:"source"`
	IntegrationID   *uuid.UUID       `json:"integration_id,omitempty"`
	CustomerName    string           `json:"customer_name"`
	CustomerEmail   *string          `json:"customer_email,omitempty"`
	CustomerPhone   *string          `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage  `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage  `json:"billing_address,omitempty"`
	Items           json.RawMessage  `json:"items,omitempty"`
	TotalAmount     float64          `json:"total_amount"`
	Currency        string           `json:"currency"`
	Notes           *string          `json:"notes,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
	OrderedAt       *time.Time       `json:"ordered_at,omitempty"`
	PaymentStatus   *string          `json:"payment_status,omitempty"`
	PaymentMethod   *string          `json:"payment_method,omitempty"`
}

func (r *CreateOrderRequest) Validate() error {
	if strings.TrimSpace(r.CustomerName) == "" {
		return errors.New("customer_name is required")
	}
	switch r.Source {
	case "allegro", "woocommerce", "manual":
		// valid
	default:
		return errors.New("source must be one of: allegro, woocommerce, manual")
	}
	if r.TotalAmount < 0 {
		return errors.New("total_amount must be non-negative")
	}
	if r.Currency == "" {
		r.Currency = "PLN"
	}
	return nil
}

type UpdateOrderRequest struct {
	ExternalID      *string          `json:"external_id,omitempty"`
	CustomerName    *string          `json:"customer_name,omitempty"`
	CustomerEmail   *string          `json:"customer_email,omitempty"`
	CustomerPhone   *string          `json:"customer_phone,omitempty"`
	ShippingAddress json.RawMessage  `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage  `json:"billing_address,omitempty"`
	Items           json.RawMessage  `json:"items,omitempty"`
	TotalAmount     *float64         `json:"total_amount,omitempty"`
	Currency        *string          `json:"currency,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
	PaymentStatus   *string          `json:"payment_status,omitempty"`
	PaymentMethod   *string          `json:"payment_method,omitempty"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
}

func (r *UpdateOrderRequest) Validate() error {
	if r.ExternalID == nil && r.CustomerName == nil && r.CustomerEmail == nil &&
		r.CustomerPhone == nil && r.ShippingAddress == nil && r.BillingAddress == nil &&
		r.Items == nil && r.TotalAmount == nil && r.Currency == nil &&
		r.Notes == nil && r.Metadata == nil &&
		r.PaymentStatus == nil && r.PaymentMethod == nil && r.PaidAt == nil {
		return errors.New("at least one field must be provided")
	}
	if r.TotalAmount != nil && *r.TotalAmount < 0 {
		return errors.New("total_amount must be non-negative")
	}
	return nil
}

type StatusTransitionRequest struct {
	Status string `json:"status"`
	Force  bool   `json:"force,omitempty"`
}

func (r *StatusTransitionRequest) Validate() error {
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

type OrderListFilter struct {
	Status        *string
	Source        *string
	Search        *string
	PaymentStatus *string
	PaginationParams
}

type BulkStatusTransitionRequest struct {
	OrderIDs []uuid.UUID `json:"order_ids"`
	Status   string      `json:"status"`
	Force    bool        `json:"force,omitempty"`
}

func (r *BulkStatusTransitionRequest) Validate() error {
	if len(r.OrderIDs) == 0 {
		return errors.New("at least one order_id is required")
	}
	if len(r.OrderIDs) > 100 {
		return errors.New("maximum 100 orders per bulk operation")
	}
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

type BulkStatusResult struct {
	OrderID uuid.UUID `json:"order_id"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

type BulkStatusTransitionResponse struct {
	Results   []BulkStatusResult `json:"results"`
	Succeeded int                `json:"succeeded"`
	Failed    int                `json:"failed"`
}
