package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PickPackSession represents a pick-and-pack workflow session.
type PickPackSession struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	SessionType string         `json:"session_type"` // single, batch
	Status      string         `json:"status"`       // picking, packing, completed, cancelled
	AssignedTo  *uuid.UUID     `json:"assigned_to,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Notes       *string        `json:"notes,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Items       []PickPackItem `json:"items,omitempty"`
	Stats       *PickPackStats `json:"stats,omitempty"`
}

// PickPackItem represents a single line item in a pick-pack session.
type PickPackItem struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	SessionID        uuid.UUID  `json:"session_id"`
	OrderID          uuid.UUID  `json:"order_id"`
	ProductID        *uuid.UUID `json:"product_id,omitempty"`
	SKU              string     `json:"sku"`
	ProductName      string     `json:"product_name"`
	QuantityRequired int        `json:"quantity_required"`
	QuantityPicked   int        `json:"quantity_picked"`
	QuantityPacked   int        `json:"quantity_packed"`
	PickLocation     *string    `json:"pick_location,omitempty"`
	PickedAt         *time.Time `json:"picked_at,omitempty"`
	PackedAt         *time.Time `json:"packed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// PickPackStats holds summary statistics for a pick-pack session.
type PickPackStats struct {
	TotalItems      int `json:"total_items"`
	TotalPicked     int `json:"total_picked"`
	TotalPacked     int `json:"total_packed"`
	TotalRequired   int `json:"total_required"`
	OrderCount      int `json:"order_count"`
	AllPicked       bool `json:"all_picked"`
	AllPacked       bool `json:"all_packed"`
}

// CreatePickPackSessionRequest is the payload for creating a new pick-pack session.
type CreatePickPackSessionRequest struct {
	OrderIDs []uuid.UUID `json:"order_ids"`
	Notes    *string     `json:"notes,omitempty"`
}

// Validate validates the create pick-pack session request.
func (r *CreatePickPackSessionRequest) Validate() error {
	if len(r.OrderIDs) == 0 {
		return errors.New("at least one order_id is required")
	}
	for _, id := range r.OrderIDs {
		if id == uuid.Nil {
			return errors.New("order_ids must not contain empty UUIDs")
		}
	}
	return nil
}

// ScanItemRequest is the payload for scanning a barcode during picking.
type ScanItemRequest struct {
	Barcode string `json:"barcode"`
}

// Validate validates the scan item request.
func (r *ScanItemRequest) Validate() error {
	if r.Barcode == "" {
		return errors.New("barcode is required")
	}
	return nil
}

// ScanItemResponse is the response after scanning a barcode.
type ScanItemResponse struct {
	Item      PickPackItem `json:"item"`
	Remaining int          `json:"remaining"` // items still to pick
	Message   string       `json:"message"`
}

// MarkPackedRequest is the payload for marking items as packed.
type MarkPackedRequest struct {
	Quantity int `json:"quantity"`
}

// Validate validates the mark packed request.
func (r *MarkPackedRequest) Validate() error {
	if r.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	return nil
}

// PickPackSessionListFilter holds filtering/pagination for listing sessions.
type PickPackSessionListFilter struct {
	Status     *string
	AssignedTo *uuid.UUID
	PaginationParams
}
