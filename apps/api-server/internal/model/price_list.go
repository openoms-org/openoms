package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PriceList represents a B2B price list.
type PriceList struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Name         string     `json:"name"`
	Description  *string    `json:"description,omitempty"`
	Currency     string     `json:"currency"`
	IsDefault    bool       `json:"is_default"`
	DiscountType string     `json:"discount_type"`
	Active       bool       `json:"active"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidTo      *time.Time `json:"valid_to,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// PriceListItem represents a single item/rule within a price list.
type PriceListItem struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	PriceListID uuid.UUID  `json:"price_list_id"`
	ProductID   uuid.UUID  `json:"product_id"`
	VariantID   *uuid.UUID `json:"variant_id,omitempty"`
	Price       *float64   `json:"price,omitempty"`
	Discount    *float64   `json:"discount,omitempty"`
	MinQuantity int        `json:"min_quantity"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreatePriceListRequest is the payload to create a new price list.
type CreatePriceListRequest struct {
	Name         string     `json:"name"`
	Description  *string    `json:"description,omitempty"`
	Currency     string     `json:"currency"`
	IsDefault    bool       `json:"is_default"`
	DiscountType string     `json:"discount_type"`
	Active       *bool      `json:"active,omitempty"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidTo      *time.Time `json:"valid_to,omitempty"`
}

func (r *CreatePriceListRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	if r.Currency == "" {
		r.Currency = "PLN"
	}
	if err := validateMaxLength("currency", r.Currency, 10); err != nil {
		return err
	}
	switch r.DiscountType {
	case "":
		r.DiscountType = "percentage"
	case "percentage", "fixed", "override":
		// valid
	default:
		return errors.New("discount_type must be one of: percentage, fixed, override")
	}
	return nil
}

// UpdatePriceListRequest is the payload to update an existing price list.
type UpdatePriceListRequest struct {
	Name         *string    `json:"name,omitempty"`
	Description  *string    `json:"description,omitempty"`
	Currency     *string    `json:"currency,omitempty"`
	IsDefault    *bool      `json:"is_default,omitempty"`
	DiscountType *string    `json:"discount_type,omitempty"`
	Active       *bool      `json:"active,omitempty"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidTo      *time.Time `json:"valid_to,omitempty"`
}

func (r *UpdatePriceListRequest) Validate() error {
	if r.Name == nil && r.Description == nil && r.Currency == nil &&
		r.IsDefault == nil && r.DiscountType == nil && r.Active == nil &&
		r.ValidFrom == nil && r.ValidTo == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil {
		if strings.TrimSpace(*r.Name) == "" {
			return errors.New("name must not be empty")
		}
		if err := validateMaxLength("name", *r.Name, 500); err != nil {
			return err
		}
	}
	if r.DiscountType != nil {
		switch *r.DiscountType {
		case "percentage", "fixed", "override":
			// valid
		default:
			return errors.New("discount_type must be one of: percentage, fixed, override")
		}
	}
	return nil
}

// CreatePriceListItemRequest is the payload to add an item to a price list.
type CreatePriceListItemRequest struct {
	ProductID   uuid.UUID  `json:"product_id"`
	VariantID   *uuid.UUID `json:"variant_id,omitempty"`
	Price       *float64   `json:"price,omitempty"`
	Discount    *float64   `json:"discount,omitempty"`
	MinQuantity int        `json:"min_quantity"`
}

func (r *CreatePriceListItemRequest) Validate() error {
	if r.ProductID == uuid.Nil {
		return errors.New("product_id is required")
	}
	if r.MinQuantity <= 0 {
		r.MinQuantity = 1
	}
	return nil
}

// PriceListListFilter holds filtering/pagination for listing price lists.
type PriceListListFilter struct {
	Active *bool
	PaginationParams
}

// CalculatePriceResponse is the response for a price calculation.
type CalculatePriceResponse struct {
	OriginalPrice  float64 `json:"original_price"`
	EffectivePrice float64 `json:"effective_price"`
	DiscountType   string  `json:"discount_type"`
	DiscountValue  float64 `json:"discount_value"`
}
