package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Stocktake represents an inventory counting session for a warehouse.
type Stocktake struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	WarehouseID uuid.UUID  `json:"warehouse_id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"` // draft, in_progress, completed, cancelled
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Enriched fields (not stored in DB directly)
	Stats *StocktakeStats `json:"stats,omitempty"`
	Items []StocktakeItem `json:"items,omitempty"`
}

// StocktakeItem represents a single product line in a stocktake.
type StocktakeItem struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	StocktakeID      uuid.UUID  `json:"stocktake_id"`
	ProductID        uuid.UUID  `json:"product_id"`
	ExpectedQuantity int        `json:"expected_quantity"`
	CountedQuantity  *int       `json:"counted_quantity"`
	Difference       int        `json:"difference"`
	Notes            *string    `json:"notes,omitempty"`
	CountedAt        *time.Time `json:"counted_at,omitempty"`
	CountedBy        *uuid.UUID `json:"counted_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`

	// Enriched fields
	ProductName *string `json:"product_name,omitempty"`
	ProductSKU  *string `json:"product_sku,omitempty"`
}

// StocktakeStats holds summary statistics for a stocktake.
type StocktakeStats struct {
	TotalItems    int `json:"total_items"`
	CountedItems  int `json:"counted_items"`
	Discrepancies int `json:"discrepancies"`
	SurplusCount  int `json:"surplus_count"`
	ShortageCount int `json:"shortage_count"`
}

// CreateStocktakeRequest is the payload for creating a new stocktake.
type CreateStocktakeRequest struct {
	WarehouseID uuid.UUID   `json:"warehouse_id"`
	Name        string      `json:"name"`
	Notes       *string     `json:"notes,omitempty"`
	ProductIDs  []uuid.UUID `json:"product_ids,omitempty"` // empty = all products in warehouse
}

// Validate validates the create stocktake request.
func (r *CreateStocktakeRequest) Validate() error {
	if r.WarehouseID == uuid.Nil {
		return errors.New("warehouse_id is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	return nil
}

// UpdateStocktakeItemRequest is the payload for recording a count.
type UpdateStocktakeItemRequest struct {
	CountedQuantity int     `json:"counted_quantity"`
	Notes           *string `json:"notes,omitempty"`
}

// Validate validates the update stocktake item request.
func (r *UpdateStocktakeItemRequest) Validate() error {
	if r.CountedQuantity < 0 {
		return errors.New("counted_quantity must not be negative")
	}
	return nil
}

// StocktakeListFilter holds filtering/pagination for listing stocktakes.
type StocktakeListFilter struct {
	WarehouseID *uuid.UUID
	Status      *string
	PaginationParams
}

// StocktakeItemListFilter holds filtering/pagination for listing stocktake items.
type StocktakeItemListFilter struct {
	Filter string // "all", "uncounted", "discrepancies"
	PaginationParams
}
