package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Supplier struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Name         string          `json:"name"`
	Code         *string         `json:"code,omitempty"`
	FeedURL      *string         `json:"feed_url,omitempty"`
	FeedFormat   string          `json:"feed_format"`
	Status       string          `json:"status"`
	Settings     json.RawMessage `json:"settings"`
	LastSyncAt   *time.Time      `json:"last_sync_at,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SupplierProduct struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	ProductID     *uuid.UUID      `json:"product_id,omitempty"`
	ExternalID    string          `json:"external_id"`
	Name          string          `json:"name"`
	EAN           *string         `json:"ean,omitempty"`
	SKU           *string         `json:"sku,omitempty"`
	Price         *float64        `json:"price,omitempty"`
	StockQuantity int             `json:"stock_quantity"`
	Metadata      json.RawMessage `json:"metadata"`
	LastSyncedAt  *time.Time      `json:"last_synced_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type CreateSupplierRequest struct {
	Name       string          `json:"name"`
	Code       *string         `json:"code,omitempty"`
	FeedURL    *string         `json:"feed_url,omitempty"`
	FeedFormat string          `json:"feed_format"`
	Settings   json.RawMessage `json:"settings,omitempty"`
}

func (r *CreateSupplierRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if r.FeedFormat == "" {
		r.FeedFormat = "iof"
	}
	switch r.FeedFormat {
	case "iof", "csv", "custom":
		// valid
	default:
		return errors.New("feed_format must be one of: iof, csv, custom")
	}
	return nil
}

type UpdateSupplierRequest struct {
	Name         *string          `json:"name,omitempty"`
	Code         *string          `json:"code,omitempty"`
	FeedURL      *string          `json:"feed_url,omitempty"`
	FeedFormat   *string          `json:"feed_format,omitempty"`
	Status       *string          `json:"status,omitempty"`
	Settings     *json.RawMessage `json:"settings,omitempty"`
	ErrorMessage *string          `json:"error_message,omitempty"`
}

func (r *UpdateSupplierRequest) Validate() error {
	if r.Name == nil && r.Code == nil && r.FeedURL == nil &&
		r.FeedFormat == nil && r.Status == nil && r.Settings == nil &&
		r.ErrorMessage == nil {
		return errors.New("at least one field must be provided")
	}
	if r.FeedFormat != nil {
		switch *r.FeedFormat {
		case "iof", "csv", "custom":
			// valid
		default:
			return errors.New("feed_format must be one of: iof, csv, custom")
		}
	}
	if r.Status != nil {
		switch *r.Status {
		case "active", "inactive":
			// valid
		default:
			return errors.New("status must be one of: active, inactive")
		}
	}
	return nil
}

type SupplierListFilter struct {
	Status *string
	PaginationParams
}

type SupplierProductListFilter struct {
	SupplierID *uuid.UUID
	EAN        *string
	Linked     *bool
	PaginationParams
}

type LinkProductRequest struct {
	ProductID uuid.UUID `json:"product_id"`
}

func (r *LinkProductRequest) Validate() error {
	if r.ProductID == uuid.Nil {
		return errors.New("product_id is required")
	}
	return nil
}
