package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ProductBundle represents a product that bundles multiple component products together.
type ProductBundle struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	BundleProductID    uuid.UUID  `json:"bundle_product_id"`
	ComponentProductID uuid.UUID  `json:"component_product_id"`
	ComponentVariantID *uuid.UUID `json:"component_variant_id,omitempty"`
	Quantity           int        `json:"quantity"`
	Position           int        `json:"position"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Joined fields (not in DB table, populated by queries)
	ComponentName  string  `json:"component_name,omitempty"`
	ComponentSKU   *string `json:"component_sku,omitempty"`
	ComponentStock int     `json:"component_stock"`
}

// CreateBundleComponentRequest is the payload for adding a component to a product bundle.
type CreateBundleComponentRequest struct {
	ComponentProductID uuid.UUID  `json:"component_product_id"`
	ComponentVariantID *uuid.UUID `json:"component_variant_id,omitempty"`
	Quantity           int        `json:"quantity"`
	Position           int        `json:"position"`
}

// Validate validates the create bundle component request.
func (r *CreateBundleComponentRequest) Validate() error {
	if r.ComponentProductID == uuid.Nil {
		return errors.New("component_product_id is required")
	}
	if r.Quantity < 1 {
		return errors.New("quantity must be at least 1")
	}
	return nil
}

// UpdateBundleComponentRequest is the payload for updating a bundle component.
type UpdateBundleComponentRequest struct {
	Quantity *int `json:"quantity,omitempty"`
	Position *int `json:"position,omitempty"`
}

// Validate validates the update bundle component request.
func (r *UpdateBundleComponentRequest) Validate() error {
	if r.Quantity == nil && r.Position == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Quantity != nil && *r.Quantity < 1 {
		return errors.New("quantity must be at least 1")
	}
	return nil
}
