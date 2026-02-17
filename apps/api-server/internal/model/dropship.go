package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DropshipOrder represents a forwarded order to a supplier for dropship fulfillment.
type DropshipOrder struct {
	ID                uuid.UUID           `json:"id"`
	TenantID          uuid.UUID           `json:"tenant_id"`
	OrderID           uuid.UUID           `json:"order_id"`
	SupplierID        uuid.UUID           `json:"supplier_id"`
	SupplierName      string              `json:"supplier_name"`
	Status            string              `json:"status"`
	SupplierReference *string             `json:"supplier_reference,omitempty"`
	TrackingNumber    *string             `json:"tracking_number,omitempty"`
	Carrier           *string             `json:"carrier,omitempty"`
	Notes             *string             `json:"notes,omitempty"`
	TotalCost         float64             `json:"total_cost"`
	Currency          string              `json:"currency"`
	SentAt            *time.Time          `json:"sent_at,omitempty"`
	ConfirmedAt       *time.Time          `json:"confirmed_at,omitempty"`
	ShippedAt         *time.Time          `json:"shipped_at,omitempty"`
	DeliveredAt       *time.Time          `json:"delivered_at,omitempty"`
	Items             []DropshipOrderItem `json:"items,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

// DropshipOrderItem represents a line item in a dropship order.
type DropshipOrderItem struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	DropshipOrderID uuid.UUID  `json:"dropship_order_id"`
	ProductID       *uuid.UUID `json:"product_id,omitempty"`
	SKU             string     `json:"sku"`
	ProductName     string     `json:"product_name"`
	Quantity        int        `json:"quantity"`
	UnitCost        float64    `json:"unit_cost"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateDropshipOrderRequest is the payload for manually creating a dropship order.
type CreateDropshipOrderRequest struct {
	OrderID    uuid.UUID                    `json:"order_id"`
	SupplierID uuid.UUID                    `json:"supplier_id"`
	Notes      *string                      `json:"notes,omitempty"`
	Currency   string                       `json:"currency"`
	Items      []CreateDropshipOrderItemReq `json:"items"`
}

// Validate validates the create dropship order request.
func (r *CreateDropshipOrderRequest) Validate() error {
	if r.OrderID == uuid.Nil {
		return errors.New("order_id is required")
	}
	if r.SupplierID == uuid.Nil {
		return errors.New("supplier_id is required")
	}
	if r.Currency == "" {
		r.Currency = "PLN"
	}
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	for _, item := range r.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CreateDropshipOrderItemReq is a single item in the create dropship order request.
type CreateDropshipOrderItemReq struct {
	ProductID   *uuid.UUID `json:"product_id,omitempty"`
	SKU         string     `json:"sku"`
	ProductName string     `json:"product_name"`
	Quantity    int        `json:"quantity"`
	UnitCost    float64    `json:"unit_cost"`
}

// Validate validates a single dropship order item request.
func (r *CreateDropshipOrderItemReq) Validate() error {
	if strings.TrimSpace(r.ProductName) == "" {
		return errors.New("product_name is required")
	}
	if r.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	if r.UnitCost < 0 {
		return errors.New("unit_cost must not be negative")
	}
	return nil
}

// UpdateDropshipStatusRequest is the payload for updating dropship order status.
type UpdateDropshipStatusRequest struct {
	Status            string  `json:"status"`
	TrackingNumber    *string `json:"tracking_number,omitempty"`
	Carrier           *string `json:"carrier,omitempty"`
	SupplierReference *string `json:"supplier_reference,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

// Validate validates the update status request.
func (r *UpdateDropshipStatusRequest) Validate() error {
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	validStatuses := map[string]bool{
		"pending": true, "sent": true, "confirmed": true,
		"shipped": true, "delivered": true, "cancelled": true,
	}
	if !validStatuses[r.Status] {
		return errors.New("status must be one of: pending, sent, confirmed, shipped, delivered, cancelled")
	}
	return nil
}

// DropshipOrderListFilter holds the filtering/pagination for listing dropship orders.
type DropshipOrderListFilter struct {
	Status     *string
	SupplierID *uuid.UUID
	OrderID    *uuid.UUID
	PaginationParams
}
