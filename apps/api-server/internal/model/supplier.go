package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Supplier struct {
	ID                  uuid.UUID       `json:"id"`
	TenantID            uuid.UUID       `json:"tenant_id"`
	Name                string          `json:"name"`
	Code                *string         `json:"code,omitempty"`
	FeedURL             *string         `json:"feed_url,omitempty"`
	FeedFormat          string          `json:"feed_format"`
	Status              string          `json:"status"`
	Settings            json.RawMessage `json:"settings"`
	SyncIntervalMinutes int             `json:"sync_interval_minutes"`
	LastSyncAt          *time.Time      `json:"last_sync_at,omitempty"`
	ErrorMessage        *string         `json:"error_message,omitempty"`
	PortalEnabled       bool            `json:"portal_enabled"`
	IntegrationID       *uuid.UUID      `json:"integration_id,omitempty"`
	DefaultCategoryID   *uuid.UUID      `json:"default_category_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type SupplierProduct struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	SupplierID     uuid.UUID       `json:"supplier_id"`
	ProductID      *uuid.UUID      `json:"product_id,omitempty"`
	ExternalID     string          `json:"external_id"`
	Name           string          `json:"name"`
	EAN            *string         `json:"ean,omitempty"`
	SKU            *string         `json:"sku,omitempty"`
	Price          *float64        `json:"price,omitempty"`
	StockQuantity  int             `json:"stock_quantity"`
	SourceCategory *string         `json:"source_category,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
	LastSyncedAt   *time.Time      `json:"last_synced_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateSupplierRequest struct {
	Name                string          `json:"name"`
	Code                *string         `json:"code,omitempty"`
	FeedURL             *string         `json:"feed_url,omitempty"`
	FeedFormat          string          `json:"feed_format"`
	SyncIntervalMinutes *int            `json:"sync_interval_minutes,omitempty"`
	Settings            json.RawMessage `json:"settings,omitempty"`
	IntegrationID       *uuid.UUID      `json:"integration_id,omitempty"`
}

func (r *CreateSupplierRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if r.FeedFormat == "" {
		r.FeedFormat = "iof"
	}
	switch r.FeedFormat {
	case "iof", "csv", "custom", "btp", "xml":
		// valid
	default:
		return errors.New("feed_format must be one of: iof, csv, custom, btp, xml")
	}
	if r.SyncIntervalMinutes != nil && (*r.SyncIntervalMinutes < 5 || *r.SyncIntervalMinutes > 1440) {
		return errors.New("sync_interval_minutes must be between 5 and 1440")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	return nil
}

type UpdateSupplierRequest struct {
	Name                *string          `json:"name,omitempty"`
	Code                *string          `json:"code,omitempty"`
	FeedURL             *string          `json:"feed_url,omitempty"`
	FeedFormat          *string          `json:"feed_format,omitempty"`
	Status              *string          `json:"status,omitempty"`
	Settings            *json.RawMessage `json:"settings,omitempty"`
	SyncIntervalMinutes *int             `json:"sync_interval_minutes,omitempty"`
	ErrorMessage        *string          `json:"error_message,omitempty"`
	PortalEnabled       *bool            `json:"portal_enabled,omitempty"`
	IntegrationID       *uuid.UUID       `json:"integration_id,omitempty"`
	DefaultCategoryID   *uuid.UUID       `json:"default_category_id,omitempty"`
}

func (r *UpdateSupplierRequest) Validate() error {
	if r.Name == nil && r.Code == nil && r.FeedURL == nil &&
		r.FeedFormat == nil && r.Status == nil && r.Settings == nil &&
		r.SyncIntervalMinutes == nil && r.ErrorMessage == nil && r.PortalEnabled == nil &&
		r.IntegrationID == nil && r.DefaultCategoryID == nil {
		return errors.New("at least one field must be provided")
	}
	if r.FeedFormat != nil {
		switch *r.FeedFormat {
		case "iof", "csv", "custom", "btp", "xml":
			// valid
		default:
			return errors.New("feed_format must be one of: iof, csv, custom, btp, xml")
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
	if r.SyncIntervalMinutes != nil && (*r.SyncIntervalMinutes < 5 || *r.SyncIntervalMinutes > 1440) {
		return errors.New("sync_interval_minutes must be between 5 and 1440")
	}
	return nil
}

type SupplierListFilter struct {
	Status *string
	PaginationParams
}

type SupplierProductListFilter struct {
	SupplierID     *uuid.UUID
	EAN            *string
	Linked         *bool
	Search         *string
	SourceCategory *string
	PaginationParams
}

// ImportSupplierProductsRequest is the request body for bulk importing supplier products into the OMS catalog.
type ImportSupplierProductsRequest struct {
	SupplierProductIDs []uuid.UUID `json:"supplier_product_ids"`
}

// Validate checks that the request contains between 1 and 500 supplier product IDs.
func (r *ImportSupplierProductsRequest) Validate() error {
	if len(r.SupplierProductIDs) == 0 {
		return errors.New("supplier_product_ids is required")
	}
	if len(r.SupplierProductIDs) > 500 {
		return errors.New("max 500 products per import")
	}
	return nil
}

// ImportSupplierProductsResponse is the response body for a bulk import operation.
type ImportSupplierProductsResponse struct {
	Imported int                          `json:"imported"`
	Skipped  int                          `json:"skipped"`
	Errors   []ImportSupplierProductError `json:"errors,omitempty"`
}

// ImportSupplierProductError describes a single product that failed to import.
type ImportSupplierProductError struct {
	SupplierProductID string `json:"supplier_product_id"`
	Reason            string `json:"reason"`
}

type SupplierCategoryMapping struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	SupplierID     uuid.UUID  `json:"supplier_id"`
	SourceCategory string     `json:"source_category"`
	CategoryID     *uuid.UUID `json:"category_id,omitempty"`
	AutoMatched    bool       `json:"auto_matched"`
	Confirmed      bool       `json:"confirmed"`
	CreatedAt      time.Time  `json:"created_at"`
}

type UpsertCategoryMappingRequest struct {
	SourceCategory string     `json:"source_category"`
	CategoryID     *uuid.UUID `json:"category_id,omitempty"`
	Confirmed      bool       `json:"confirmed"`
}

func (r *UpsertCategoryMappingRequest) Validate() error {
	if strings.TrimSpace(r.SourceCategory) == "" {
		return errors.New("source_category is required")
	}
	return nil
}

// BulkDeleteSupplierProductsRequest is the request body for bulk deleting supplier products.
type BulkDeleteSupplierProductsRequest struct {
	SupplierProductIDs []uuid.UUID `json:"supplier_product_ids"`
}

// Validate checks that the request contains between 1 and 500 supplier product IDs.
func (r *BulkDeleteSupplierProductsRequest) Validate() error {
	if len(r.SupplierProductIDs) == 0 {
		return errors.New("supplier_product_ids is required")
	}
	if len(r.SupplierProductIDs) > 500 {
		return errors.New("max 500 products per bulk delete")
	}
	return nil
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

// AllegroParameterMapping maps a supplier's data field to an Allegro category parameter.
type AllegroParameterMapping struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	SupplierID        uuid.UUID       `json:"supplier_id"`
	AllegroCategoryID string          `json:"allegro_category_id"`
	AllegroParamID    string          `json:"allegro_param_id"`
	AllegroParamName  string          `json:"allegro_param_name"`
	SourceType        string          `json:"source_type"`
	SourceKey         string          `json:"source_key"`
	ValueMapping      json.RawMessage `json:"value_mapping"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// UpsertAllegroParameterMappingRequest represents a single mapping entry in a bulk upsert.
type UpsertAllegroParameterMappingRequest struct {
	AllegroParamID   string          `json:"allegro_param_id"`
	AllegroParamName string          `json:"allegro_param_name"`
	SourceType       string          `json:"source_type"`
	SourceKey        string          `json:"source_key"`
	ValueMapping     json.RawMessage `json:"value_mapping,omitempty"`
}

// BulkUpsertAllegroMappingsRequest is the payload for creating/updating Allegro parameter mappings.
type BulkUpsertAllegroMappingsRequest struct {
	AllegroCategoryID string                                 `json:"allegro_category_id"`
	Mappings          []UpsertAllegroParameterMappingRequest `json:"mappings"`
}

// Validate checks required fields and limits.
func (r *BulkUpsertAllegroMappingsRequest) Validate() error {
	if strings.TrimSpace(r.AllegroCategoryID) == "" {
		return errors.New("allegro_category_id is required")
	}
	if len(r.Mappings) == 0 {
		return errors.New("at least one mapping is required")
	}
	if len(r.Mappings) > 200 {
		return errors.New("max 200 mappings per request")
	}
	for _, m := range r.Mappings {
		if strings.TrimSpace(m.AllegroParamID) == "" {
			return errors.New("allegro_param_id is required for each mapping")
		}
		if strings.TrimSpace(m.SourceType) == "" {
			return errors.New("source_type is required")
		}
		switch m.SourceType {
		case "attribute", "field", "static":
			// valid
		default:
			return errors.New("source_type must be one of: attribute, field, static")
		}
		if strings.TrimSpace(m.SourceKey) == "" {
			return errors.New("source_key is required")
		}
	}
	return nil
}
