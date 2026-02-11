package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Warehouse represents a physical warehouse/stock location.
type Warehouse struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Name      string          `json:"name"`
	Code      *string         `json:"code,omitempty"`
	Address   json.RawMessage `json:"address"`
	IsDefault bool            `json:"is_default"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// WarehouseStock represents the stock level for a product (optionally variant) in a warehouse.
type WarehouseStock struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	WarehouseID uuid.UUID  `json:"warehouse_id"`
	ProductID   uuid.UUID  `json:"product_id"`
	VariantID   *uuid.UUID `json:"variant_id,omitempty"`
	Quantity    int        `json:"quantity"`
	Reserved    int        `json:"reserved"`
	MinStock    int        `json:"min_stock"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateWarehouseRequest is the payload for creating a new warehouse.
type CreateWarehouseRequest struct {
	Name      string          `json:"name"`
	Code      *string         `json:"code,omitempty"`
	Address   json.RawMessage `json:"address,omitempty"`
	IsDefault *bool           `json:"is_default,omitempty"`
	Active    *bool           `json:"active,omitempty"`
}

// Validate validates the create warehouse request.
func (r *CreateWarehouseRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	return nil
}

// UpdateWarehouseRequest is the payload for updating an existing warehouse.
type UpdateWarehouseRequest struct {
	Name      *string          `json:"name,omitempty"`
	Code      *string          `json:"code,omitempty"`
	Address   *json.RawMessage `json:"address,omitempty"`
	IsDefault *bool            `json:"is_default,omitempty"`
	Active    *bool            `json:"active,omitempty"`
}

// Validate validates the update warehouse request.
func (r *UpdateWarehouseRequest) Validate() error {
	if r.Name == nil && r.Code == nil && r.Address == nil && r.IsDefault == nil && r.Active == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name must not be empty")
	}
	if r.Name != nil {
		if err := validateMaxLength("name", *r.Name, 500); err != nil {
			return err
		}
	}
	return nil
}

// WarehouseListFilter holds the filtering/pagination for listing warehouses.
type WarehouseListFilter struct {
	Active *bool
	PaginationParams
}

// UpsertWarehouseStockRequest is the payload for upserting a stock entry.
type UpsertWarehouseStockRequest struct {
	ProductID uuid.UUID  `json:"product_id"`
	VariantID *uuid.UUID `json:"variant_id,omitempty"`
	Quantity  int        `json:"quantity"`
	Reserved  int        `json:"reserved"`
	MinStock  int        `json:"min_stock"`
}

// Validate validates the upsert warehouse stock request.
func (r *UpsertWarehouseStockRequest) Validate() error {
	if r.ProductID == uuid.Nil {
		return errors.New("product_id is required")
	}
	if r.Quantity < 0 {
		return errors.New("quantity must not be negative")
	}
	if r.Reserved < 0 {
		return errors.New("reserved must not be negative")
	}
	if r.MinStock < 0 {
		return errors.New("min_stock must not be negative")
	}
	return nil
}

// WarehouseStockListFilter holds the filtering/pagination for listing warehouse stock.
type WarehouseStockListFilter struct {
	PaginationParams
}
