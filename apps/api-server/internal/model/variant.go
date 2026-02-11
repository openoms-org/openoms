package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProductVariant represents a single variant (size, color, etc.) of a product.
type ProductVariant struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	ProductID     uuid.UUID       `json:"product_id"`
	SKU           *string         `json:"sku,omitempty"`
	EAN           *string         `json:"ean,omitempty"`
	Name          string          `json:"name"`
	Attributes    json.RawMessage `json:"attributes"`
	PriceOverride *float64        `json:"price_override,omitempty"`
	StockQuantity int             `json:"stock_quantity"`
	Weight        *float64        `json:"weight,omitempty"`
	ImageURL      *string         `json:"image_url,omitempty"`
	Position      int             `json:"position"`
	Active        bool            `json:"active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// CreateVariantRequest is the payload for creating a new variant.
type CreateVariantRequest struct {
	SKU           *string         `json:"sku,omitempty"`
	EAN           *string         `json:"ean,omitempty"`
	Name          string          `json:"name"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	PriceOverride *float64        `json:"price_override,omitempty"`
	StockQuantity int             `json:"stock_quantity"`
	Weight        *float64        `json:"weight,omitempty"`
	ImageURL      *string         `json:"image_url,omitempty"`
	Position      *int            `json:"position,omitempty"`
	Active        *bool           `json:"active,omitempty"`
}

// Validate validates the create variant request.
func (r *CreateVariantRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if r.StockQuantity < 0 {
		return errors.New("stock_quantity must not be negative")
	}
	if r.PriceOverride != nil && *r.PriceOverride < 0 {
		return errors.New("price_override must not be negative")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("sku", r.SKU, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("ean", r.EAN, 50); err != nil {
		return err
	}
	return nil
}

// UpdateVariantRequest is the payload for updating an existing variant.
type UpdateVariantRequest struct {
	SKU           *string          `json:"sku,omitempty"`
	EAN           *string          `json:"ean,omitempty"`
	Name          *string          `json:"name,omitempty"`
	Attributes    *json.RawMessage `json:"attributes,omitempty"`
	PriceOverride *float64         `json:"price_override,omitempty"`
	StockQuantity *int             `json:"stock_quantity,omitempty"`
	Weight        *float64         `json:"weight,omitempty"`
	ImageURL      *string          `json:"image_url,omitempty"`
	Position      *int             `json:"position,omitempty"`
	Active        *bool            `json:"active,omitempty"`
}

// Validate validates the update variant request.
func (r *UpdateVariantRequest) Validate() error {
	if r.SKU == nil && r.EAN == nil && r.Name == nil && r.Attributes == nil &&
		r.PriceOverride == nil && r.StockQuantity == nil && r.Weight == nil &&
		r.ImageURL == nil && r.Position == nil && r.Active == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name must not be empty")
	}
	if r.StockQuantity != nil && *r.StockQuantity < 0 {
		return errors.New("stock_quantity must not be negative")
	}
	if r.PriceOverride != nil && *r.PriceOverride < 0 {
		return errors.New("price_override must not be negative")
	}
	if err := validateMaxLengthPtr("name", r.Name, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("sku", r.SKU, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("ean", r.EAN, 50); err != nil {
		return err
	}
	return nil
}

// VariantListFilter holds the filtering/pagination for listing variants.
type VariantListFilter struct {
	ProductID uuid.UUID
	Active    *bool
	PaginationParams
}
