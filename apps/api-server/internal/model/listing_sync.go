package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ListingSyncConfig defines sync settings between a local product catalog and a marketplace integration.
type ListingSyncConfig struct {
	ID                   uuid.UUID       `json:"id"`
	TenantID             uuid.UUID       `json:"tenant_id"`
	IntegrationID        uuid.UUID       `json:"integration_id"`
	SyncDirection        string          `json:"sync_direction"`
	AutoSync             bool            `json:"auto_sync"`
	SyncIntervalMinutes  int             `json:"sync_interval_minutes"`
	FieldMapping         json.RawMessage `json:"field_mapping"`
	PriceRule            string          `json:"price_rule"`
	PriceModifier        float64         `json:"price_modifier"`
	StockBuffer          int             `json:"stock_buffer"`
	Status               string          `json:"status"`
	LastSyncAt           *time.Time      `json:"last_sync_at,omitempty"`
	LastError            *string         `json:"last_error,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	IntegrationProvider  string          `json:"integration_provider,omitempty"`
	IntegrationLabel     *string         `json:"integration_label,omitempty"`
}

// ListingSyncLog records the result of an individual sync operation.
type ListingSyncLog struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	ConfigID     uuid.UUID       `json:"config_id"`
	Direction    string          `json:"direction"`
	EntityType   string          `json:"entity_type"`
	ProductID    *uuid.UUID      `json:"product_id,omitempty"`
	ExternalID   *string         `json:"external_id,omitempty"`
	Status       string          `json:"status"`
	Changes      json.RawMessage `json:"changes,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// CreateListingSyncConfigRequest is the request to create a new listing sync config.
type CreateListingSyncConfigRequest struct {
	IntegrationID       uuid.UUID       `json:"integration_id"`
	SyncDirection       string          `json:"sync_direction"`
	AutoSync            *bool           `json:"auto_sync,omitempty"`
	SyncIntervalMinutes *int            `json:"sync_interval_minutes,omitempty"`
	FieldMapping        json.RawMessage `json:"field_mapping,omitempty"`
	PriceRule           string          `json:"price_rule,omitempty"`
	PriceModifier       *float64        `json:"price_modifier,omitempty"`
	StockBuffer         *int            `json:"stock_buffer,omitempty"`
}

func (r *CreateListingSyncConfigRequest) Validate() error {
	if r.IntegrationID == uuid.Nil {
		return errors.New("integration_id is required")
	}
	switch r.SyncDirection {
	case "push", "pull", "bidirectional":
		// valid
	case "":
		r.SyncDirection = "bidirectional"
	default:
		return errors.New("sync_direction must be one of: push, pull, bidirectional")
	}
	switch r.PriceRule {
	case "same", "markup_pct", "markup_fixed", "custom", "":
		// valid
	default:
		return errors.New("price_rule must be one of: same, markup_pct, markup_fixed, custom")
	}
	if r.PriceRule == "" {
		r.PriceRule = "same"
	}
	if r.SyncIntervalMinutes != nil && *r.SyncIntervalMinutes < 5 {
		return errors.New("sync_interval_minutes must be at least 5")
	}
	if r.StockBuffer != nil && *r.StockBuffer < 0 {
		return errors.New("stock_buffer must be non-negative")
	}
	return nil
}

// UpdateListingSyncConfigRequest is the request to update a listing sync config.
type UpdateListingSyncConfigRequest struct {
	SyncDirection       *string          `json:"sync_direction,omitempty"`
	AutoSync            *bool            `json:"auto_sync,omitempty"`
	SyncIntervalMinutes *int             `json:"sync_interval_minutes,omitempty"`
	FieldMapping        *json.RawMessage `json:"field_mapping,omitempty"`
	PriceRule           *string          `json:"price_rule,omitempty"`
	PriceModifier       *float64         `json:"price_modifier,omitempty"`
	StockBuffer         *int             `json:"stock_buffer,omitempty"`
	Status              *string          `json:"status,omitempty"`
}

func (r *UpdateListingSyncConfigRequest) Validate() error {
	if r.SyncDirection == nil && r.AutoSync == nil && r.SyncIntervalMinutes == nil &&
		r.FieldMapping == nil && r.PriceRule == nil && r.PriceModifier == nil &&
		r.StockBuffer == nil && r.Status == nil {
		return errors.New("at least one field must be provided")
	}
	if r.SyncDirection != nil {
		switch *r.SyncDirection {
		case "push", "pull", "bidirectional":
			// valid
		default:
			return errors.New("sync_direction must be one of: push, pull, bidirectional")
		}
	}
	if r.PriceRule != nil {
		switch *r.PriceRule {
		case "same", "markup_pct", "markup_fixed", "custom":
			// valid
		default:
			return errors.New("price_rule must be one of: same, markup_pct, markup_fixed, custom")
		}
	}
	if r.Status != nil {
		switch *r.Status {
		case "active", "paused", "error":
			// valid
		default:
			return errors.New("status must be one of: active, paused, error")
		}
	}
	if r.SyncIntervalMinutes != nil && *r.SyncIntervalMinutes < 5 {
		return errors.New("sync_interval_minutes must be at least 5")
	}
	if r.StockBuffer != nil && *r.StockBuffer < 0 {
		return errors.New("stock_buffer must be non-negative")
	}
	return nil
}

// ListingSyncConfigFilter is the filter for listing sync configs.
type ListingSyncConfigFilter struct {
	IntegrationID *uuid.UUID
	Status        *string
	PaginationParams
}

// ListingSyncLogFilter is the filter for listing sync logs.
type ListingSyncLogFilter struct {
	ConfigID   uuid.UUID
	Direction  *string
	EntityType *string
	Status     *string
	PaginationParams
}
