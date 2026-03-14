package model

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RecurringOrder represents a subscription/recurring order template.
type RecurringOrder struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	CustomerID         *uuid.UUID      `json:"customer_id,omitempty"`
	CustomerName       string          `json:"customer_name"`
	CustomerEmail      *string         `json:"customer_email,omitempty"`
	Status             string          `json:"status"`
	Frequency          string          `json:"frequency"`
	IntervalDays       int             `json:"interval_days"`
	NextOrderDate      time.Time       `json:"next_order_date"`
	LastOrderDate      *time.Time      `json:"last_order_date,omitempty"`
	EndDate            *time.Time      `json:"end_date,omitempty"`
	TotalOrdersCreated int             `json:"total_orders_created"`
	MaxOrders          *int            `json:"max_orders,omitempty"`
	ShippingAddress    json.RawMessage `json:"shipping_address,omitempty"`
	Notes              *string         `json:"notes,omitempty"`
	CreatedBy          *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`

	// Loaded separately — items for this recurring order
	Items []RecurringOrderItem `json:"items,omitempty"`
}

// RecurringOrderItem represents a line item in a recurring order template.
type RecurringOrderItem struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	RecurringOrderID uuid.UUID  `json:"recurring_order_id"`
	ProductID        *uuid.UUID `json:"product_id,omitempty"`
	SKU              string     `json:"sku"`
	ProductName      string     `json:"product_name"`
	Quantity         int        `json:"quantity"`
	UnitPrice        float64    `json:"unit_price"`
	CreatedAt        time.Time  `json:"created_at"`
}

// --- Valid frequencies ---

var validFrequencies = map[string]bool{
	"daily":     true,
	"weekly":    true,
	"biweekly":  true,
	"monthly":   true,
	"quarterly": true,
}

// IsValidFrequency reports whether f is a recognised recurring order frequency string.
func IsValidFrequency(f string) bool {
	return validFrequencies[f]
}

// FrequencyToIntervalDays maps frequency strings to interval days.
func FrequencyToIntervalDays(f string) int {
	switch f {
	case "daily":
		return 1
	case "weekly":
		return 7
	case "biweekly":
		return 14
	case "monthly":
		return 30
	case "quarterly":
		return 90
	default:
		return 0
	}
}

// --- Request types ---

// CreateRecurringOrderItemRequest is a single item in the create request.
type CreateRecurringOrderItemRequest struct {
	ProductID   *uuid.UUID `json:"product_id,omitempty"`
	SKU         string     `json:"sku"`
	ProductName string     `json:"product_name"`
	Quantity    int        `json:"quantity"`
	UnitPrice   float64    `json:"unit_price"`
}

// CreateRecurringOrderRequest is the request body for creating a recurring order.
type CreateRecurringOrderRequest struct {
	CustomerID      *uuid.UUID                        `json:"customer_id,omitempty"`
	CustomerName    string                            `json:"customer_name"`
	CustomerEmail   *string                           `json:"customer_email,omitempty"`
	Frequency       string                            `json:"frequency"`
	NextOrderDate   string                            `json:"next_order_date"` // YYYY-MM-DD
	EndDate         *string                           `json:"end_date,omitempty"`
	MaxOrders       *int                              `json:"max_orders,omitempty"`
	ShippingAddress json.RawMessage                   `json:"shipping_address,omitempty"`
	Notes           *string                           `json:"notes,omitempty"`
	Items           []CreateRecurringOrderItemRequest `json:"items"`
}

// Validate validates the create recurring order request.
func (r *CreateRecurringOrderRequest) Validate() error {
	if strings.TrimSpace(r.CustomerName) == "" {
		return errors.New("customer_name is required")
	}
	if err := validateMaxLength("customer_name", r.CustomerName, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("customer_email", r.CustomerEmail, 255); err != nil {
		return err
	}
	if !IsValidFrequency(r.Frequency) {
		return errors.New("frequency must be one of: daily, weekly, biweekly, monthly, quarterly")
	}
	if strings.TrimSpace(r.NextOrderDate) == "" {
		return errors.New("next_order_date is required (YYYY-MM-DD)")
	}
	if _, err := time.Parse("2006-01-02", r.NextOrderDate); err != nil {
		return errors.New("next_order_date must be YYYY-MM-DD format")
	}
	if r.EndDate != nil {
		if _, err := time.Parse("2006-01-02", *r.EndDate); err != nil {
			return errors.New("end_date must be YYYY-MM-DD format")
		}
	}
	if r.MaxOrders != nil && *r.MaxOrders <= 0 {
		return errors.New("max_orders must be a positive integer")
	}
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	for i, item := range r.Items {
		idx := strconv.Itoa(i)
		if strings.TrimSpace(item.SKU) == "" {
			return errors.New("items[" + idx + "].sku is required")
		}
		if strings.TrimSpace(item.ProductName) == "" {
			return errors.New("items[" + idx + "].product_name is required")
		}
		if item.Quantity <= 0 {
			return errors.New("items[" + idx + "].quantity must be positive")
		}
		if item.UnitPrice < 0 {
			return errors.New("items[" + idx + "].unit_price must be non-negative")
		}
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

// UpdateRecurringOrderRequest is the request body for updating a recurring order.
type UpdateRecurringOrderRequest struct {
	CustomerID      *uuid.UUID                         `json:"customer_id,omitempty"`
	CustomerName    *string                            `json:"customer_name,omitempty"`
	CustomerEmail   *string                            `json:"customer_email,omitempty"`
	Frequency       *string                            `json:"frequency,omitempty"`
	NextOrderDate   *string                            `json:"next_order_date,omitempty"`
	EndDate         *string                            `json:"end_date,omitempty"`
	MaxOrders       *int                               `json:"max_orders,omitempty"`
	ShippingAddress json.RawMessage                    `json:"shipping_address,omitempty"`
	Notes           *string                            `json:"notes,omitempty"`
	Items           *[]CreateRecurringOrderItemRequest `json:"items,omitempty"`
}

// Validate validates the update recurring order request.
func (r *UpdateRecurringOrderRequest) Validate() error {
	if r.CustomerName != nil && strings.TrimSpace(*r.CustomerName) == "" {
		return errors.New("customer_name cannot be empty")
	}
	if r.CustomerName != nil {
		if err := validateMaxLength("customer_name", *r.CustomerName, 500); err != nil {
			return err
		}
	}
	if err := validateMaxLengthPtr("customer_email", r.CustomerEmail, 255); err != nil {
		return err
	}
	if r.Frequency != nil && !IsValidFrequency(*r.Frequency) {
		return errors.New("frequency must be one of: daily, weekly, biweekly, monthly, quarterly")
	}
	if r.NextOrderDate != nil {
		if _, err := time.Parse("2006-01-02", *r.NextOrderDate); err != nil {
			return errors.New("next_order_date must be YYYY-MM-DD format")
		}
	}
	if r.EndDate != nil {
		if _, err := time.Parse("2006-01-02", *r.EndDate); err != nil {
			return errors.New("end_date must be YYYY-MM-DD format")
		}
	}
	if r.MaxOrders != nil && *r.MaxOrders <= 0 {
		return errors.New("max_orders must be a positive integer")
	}
	if r.Items != nil {
		for i, item := range *r.Items {
			idx := strconv.Itoa(i)
			if strings.TrimSpace(item.SKU) == "" {
				return errors.New("items[" + idx + "].sku is required")
			}
			if strings.TrimSpace(item.ProductName) == "" {
				return errors.New("items[" + idx + "].product_name is required")
			}
			if item.Quantity <= 0 {
				return errors.New("items[" + idx + "].quantity must be positive")
			}
			if item.UnitPrice < 0 {
				return errors.New("items[" + idx + "].unit_price must be non-negative")
			}
		}
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

// RecurringOrderListFilter holds query filters for listing recurring orders.
type RecurringOrderListFilter struct {
	Status         *string
	CustomerID     *string
	NextDateBefore *string
	PaginationParams
}
