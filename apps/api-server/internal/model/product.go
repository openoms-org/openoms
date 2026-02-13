package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	ExternalID       *string         `json:"external_id,omitempty"`
	Source           string          `json:"source"`
	Name             string          `json:"name"`
	SKU              *string         `json:"sku,omitempty"`
	EAN              *string         `json:"ean,omitempty"`
	Price            float64         `json:"price"`
	StockQuantity    int             `json:"stock_quantity"`
	Metadata         json.RawMessage `json:"metadata"`
	Tags             []string        `json:"tags"`
	DescriptionShort string          `json:"description_short"`
	DescriptionLong  string          `json:"description_long"`
	Weight           *float64        `json:"weight,omitempty"`
	Width            *float64        `json:"width,omitempty"`
	Height           *float64        `json:"height,omitempty"`
	Depth            *float64        `json:"depth,omitempty"`
	Category         *string         `json:"category,omitempty"`
	ImageURL         *string         `json:"image_url,omitempty"`
	Images           json.RawMessage `json:"images"`
	HasVariants      bool            `json:"has_variants"`
	IsBundle         bool            `json:"is_bundle"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type CreateProductRequest struct {
	ExternalID       *string         `json:"external_id,omitempty"`
	Source           string          `json:"source"`
	Name             string          `json:"name"`
	SKU              *string         `json:"sku,omitempty"`
	EAN              *string         `json:"ean,omitempty"`
	Price            float64         `json:"price"`
	StockQty         int             `json:"stock_quantity"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	DescriptionShort string          `json:"description_short,omitempty"`
	DescriptionLong  string          `json:"description_long,omitempty"`
	Weight           *float64        `json:"weight,omitempty"`
	Width            *float64        `json:"width,omitempty"`
	Height           *float64        `json:"height,omitempty"`
	Depth            *float64        `json:"depth,omitempty"`
	Category         *string         `json:"category,omitempty"`
	ImageURL         *string         `json:"image_url,omitempty"`
	Images           json.RawMessage `json:"images,omitempty"`
}

func (r *CreateProductRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	switch r.Source {
	case "":
		r.Source = "manual"
	case "allegro", "woocommerce", "manual":
		// valid
	default:
		return errors.New("source must be one of: allegro, woocommerce, manual")
	}
	if r.Price < 0 {
		return errors.New("price must not be negative")
	}
	if r.StockQty < 0 {
		return errors.New("stock_quantity must not be negative")
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
	if err := validateMaxLength("description_short", r.DescriptionShort, 1000); err != nil {
		return err
	}
	if err := validateMaxLength("description_long", r.DescriptionLong, 10000); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("category", r.Category, 100); err != nil {
		return err
	}
	return nil
}

type UpdateProductRequest struct {
	ExternalID       *string          `json:"external_id,omitempty"`
	Source           *string          `json:"source,omitempty"`
	Name             *string          `json:"name,omitempty"`
	SKU              *string          `json:"sku,omitempty"`
	EAN              *string          `json:"ean,omitempty"`
	Price            *float64         `json:"price,omitempty"`
	StockQuantity    *int             `json:"stock_quantity,omitempty"`
	Metadata         *json.RawMessage `json:"metadata,omitempty"`
	Tags             *[]string        `json:"tags,omitempty"`
	DescriptionShort *string          `json:"description_short,omitempty"`
	DescriptionLong  *string          `json:"description_long,omitempty"`
	Weight           *float64         `json:"weight,omitempty"`
	Width            *float64         `json:"width,omitempty"`
	Height           *float64         `json:"height,omitempty"`
	Depth            *float64         `json:"depth,omitempty"`
	Category         *string          `json:"category,omitempty"`
	ImageURL         *string          `json:"image_url,omitempty"`
	Images           *json.RawMessage `json:"images,omitempty"`
	IsBundle         *bool            `json:"is_bundle,omitempty"`
}

func (r *UpdateProductRequest) Validate() error {
	if r.ExternalID == nil && r.Source == nil && r.Name == nil && r.SKU == nil &&
		r.EAN == nil && r.Price == nil && r.StockQuantity == nil && r.Metadata == nil &&
		r.Tags == nil && r.DescriptionShort == nil && r.DescriptionLong == nil &&
		r.Weight == nil && r.Width == nil && r.Height == nil && r.Depth == nil &&
		r.Category == nil && r.ImageURL == nil && r.Images == nil && r.IsBundle == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Source != nil {
		switch *r.Source {
		case "allegro", "woocommerce", "manual":
			// valid
		default:
			return errors.New("source must be one of: allegro, woocommerce, manual")
		}
	}
	if r.Price != nil && *r.Price < 0 {
		return errors.New("price must not be negative")
	}
	if r.StockQuantity != nil && *r.StockQuantity < 0 {
		return errors.New("stock_quantity must not be negative")
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
	if err := validateMaxLengthPtr("description_short", r.DescriptionShort, 1000); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("description_long", r.DescriptionLong, 10000); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("category", r.Category, 100); err != nil {
		return err
	}
	return nil
}

type ProductListFilter struct {
	Name     *string
	SKU      *string
	Tag      *string
	Category *string
	Search   *string
	PaginationParams
}
