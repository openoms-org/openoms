package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WarehouseDocument represents a warehouse document (PZ/WZ/MM).
type WarehouseDocument struct {
	ID                uuid.UUID          `json:"id"`
	TenantID          uuid.UUID          `json:"tenant_id"`
	DocumentNumber    string             `json:"document_number"`
	DocumentType      string             `json:"document_type"` // PZ, WZ, MM
	Status            string             `json:"status"`        // draft, confirmed, cancelled
	WarehouseID       uuid.UUID          `json:"warehouse_id"`
	TargetWarehouseID *uuid.UUID         `json:"target_warehouse_id,omitempty"`
	SupplierID        *uuid.UUID         `json:"supplier_id,omitempty"`
	OrderID           *uuid.UUID         `json:"order_id,omitempty"`
	Notes             *string            `json:"notes,omitempty"`
	ConfirmedAt       *time.Time         `json:"confirmed_at,omitempty"`
	ConfirmedBy       *uuid.UUID         `json:"confirmed_by,omitempty"`
	CreatedBy         *uuid.UUID         `json:"created_by,omitempty"`
	Items             []WarehouseDocItem `json:"items,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// WarehouseDocItem represents a line item in a warehouse document.
type WarehouseDocItem struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	DocumentID uuid.UUID  `json:"document_id"`
	ProductID  uuid.UUID  `json:"product_id"`
	VariantID  *uuid.UUID `json:"variant_id,omitempty"`
	Quantity   int        `json:"quantity"`
	UnitPrice  *float64   `json:"unit_price,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateWarehouseDocumentRequest is the payload for creating a warehouse document.
type CreateWarehouseDocumentRequest struct {
	DocumentType      string                          `json:"document_type"`
	WarehouseID       uuid.UUID                       `json:"warehouse_id"`
	TargetWarehouseID *uuid.UUID                      `json:"target_warehouse_id,omitempty"`
	SupplierID        *uuid.UUID                      `json:"supplier_id,omitempty"`
	OrderID           *uuid.UUID                      `json:"order_id,omitempty"`
	Notes             *string                         `json:"notes,omitempty"`
	Items             []CreateWarehouseDocItemRequest `json:"items"`
}

// CreateWarehouseDocItemRequest is a line item in the create document request.
type CreateWarehouseDocItemRequest struct {
	ProductID uuid.UUID  `json:"product_id"`
	VariantID *uuid.UUID `json:"variant_id,omitempty"`
	Quantity  int        `json:"quantity"`
	UnitPrice *float64   `json:"unit_price,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
}

// Validate validates the create warehouse document request.
func (r *CreateWarehouseDocumentRequest) Validate() error {
	r.DocumentType = strings.TrimSpace(strings.ToUpper(r.DocumentType))
	if r.DocumentType != "PZ" && r.DocumentType != "WZ" && r.DocumentType != "MM" {
		return errors.New("document_type must be PZ, WZ, or MM")
	}
	if r.WarehouseID == uuid.Nil {
		return errors.New("warehouse_id is required")
	}
	if r.DocumentType == "MM" && (r.TargetWarehouseID == nil || *r.TargetWarehouseID == uuid.Nil) {
		return errors.New("target_warehouse_id is required for MM documents")
	}
	if r.DocumentType == "MM" && r.TargetWarehouseID != nil && *r.TargetWarehouseID == r.WarehouseID {
		return errors.New("target_warehouse_id must differ from warehouse_id")
	}
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	for i, item := range r.Items {
		if item.ProductID == uuid.Nil {
			return errors.New("product_id is required for each item")
		}
		if item.Quantity <= 0 {
			return errors.New("quantity must be positive for each item")
		}
		_ = i
	}
	return nil
}

// UpdateWarehouseDocumentRequest is the payload for updating a warehouse document.
type UpdateWarehouseDocumentRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// Validate validates the update warehouse document request.
func (r *UpdateWarehouseDocumentRequest) Validate() error {
	if r.Notes == nil {
		return errors.New("at least one field must be provided")
	}
	return nil
}

// WarehouseDocumentListFilter holds filtering/pagination for warehouse documents.
type WarehouseDocumentListFilter struct {
	DocumentType *string
	Status       *string
	WarehouseID  *uuid.UUID
	PaginationParams
}
