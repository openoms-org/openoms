package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ProductListing struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	IntegrationID  uuid.UUID       `json:"integration_id"`
	ExternalID     *string         `json:"external_id,omitempty"`
	Status         string          `json:"status"`
	URL            *string         `json:"url,omitempty"`
	PriceOverride  *float64        `json:"price_override,omitempty"`
	StockOverride  *int            `json:"stock_override,omitempty"`
	SyncStatus     string          `json:"sync_status"`
	LastSyncedAt   *time.Time      `json:"last_synced_at,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateProductListingRequest struct {
	ProductID     uuid.UUID       `json:"product_id"`
	IntegrationID uuid.UUID       `json:"integration_id"`
	ExternalID    *string         `json:"external_id,omitempty"`
	Status        string          `json:"status,omitempty"`
	URL           *string         `json:"url,omitempty"`
	PriceOverride *float64        `json:"price_override,omitempty"`
	StockOverride *int            `json:"stock_override,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

func (r *CreateProductListingRequest) Validate() error {
	if r.ProductID == uuid.Nil {
		return errors.New("product_id is required")
	}
	if r.IntegrationID == uuid.Nil {
		return errors.New("integration_id is required")
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	return nil
}

type UpdateProductListingRequest struct {
	ExternalID    *string          `json:"external_id,omitempty"`
	Status        *string          `json:"status,omitempty"`
	URL           *string          `json:"url,omitempty"`
	PriceOverride *float64         `json:"price_override,omitempty"`
	StockOverride *int             `json:"stock_override,omitempty"`
	SyncStatus    *string          `json:"sync_status,omitempty"`
	ErrorMessage  *string          `json:"error_message,omitempty"`
	Metadata      *json.RawMessage `json:"metadata,omitempty"`
}

func (r *UpdateProductListingRequest) Validate() error {
	if r.ExternalID == nil && r.Status == nil && r.URL == nil &&
		r.PriceOverride == nil && r.StockOverride == nil &&
		r.SyncStatus == nil && r.ErrorMessage == nil && r.Metadata == nil {
		return errors.New("at least one field must be provided")
	}
	return nil
}

type SyncJob struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	IntegrationID  uuid.UUID       `json:"integration_id"`
	JobType        string          `json:"job_type"`
	Status         string          `json:"status"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	ItemsProcessed int             `json:"items_processed"`
	ItemsFailed    int             `json:"items_failed"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
}
