package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	ExternalID    *string         `json:"external_id,omitempty"`
	Source        string          `json:"source"`
	Name          string          `json:"name"`
	SKU           *string         `json:"sku,omitempty"`
	EAN           *string         `json:"ean,omitempty"`
	Price         float64         `json:"price"`
	StockQuantity int             `json:"stock_quantity"`
	Metadata      json.RawMessage `json:"metadata"`
	ImageURL      *string         `json:"image_url,omitempty"`
	Images        json.RawMessage `json:"images"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type CreateProductRequest struct {
	ExternalID *string         `json:"external_id,omitempty"`
	Source     string          `json:"source"`
	Name       string          `json:"name"`
	SKU        *string         `json:"sku,omitempty"`
	EAN        *string         `json:"ean,omitempty"`
	Price      float64         `json:"price"`
	StockQty   int             `json:"stock_quantity"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	ImageURL   *string         `json:"image_url,omitempty"`
	Images     json.RawMessage `json:"images,omitempty"`
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
	return nil
}

type UpdateProductRequest struct {
	ExternalID    *string          `json:"external_id,omitempty"`
	Source        *string          `json:"source,omitempty"`
	Name          *string          `json:"name,omitempty"`
	SKU           *string          `json:"sku,omitempty"`
	EAN           *string          `json:"ean,omitempty"`
	Price         *float64         `json:"price,omitempty"`
	StockQuantity *int             `json:"stock_quantity,omitempty"`
	Metadata      *json.RawMessage `json:"metadata,omitempty"`
	ImageURL      *string          `json:"image_url,omitempty"`
	Images        *json.RawMessage `json:"images,omitempty"`
}

func (r *UpdateProductRequest) Validate() error {
	if r.ExternalID == nil && r.Source == nil && r.Name == nil && r.SKU == nil &&
		r.EAN == nil && r.Price == nil && r.StockQuantity == nil && r.Metadata == nil &&
		r.ImageURL == nil && r.Images == nil {
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
	return nil
}

type ProductListFilter struct {
	Name *string
	SKU  *string
	PaginationParams
}
