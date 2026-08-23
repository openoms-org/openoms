package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StockSyncChannel represents a sales channel configuration for stock synchronization.
type StockSyncChannel struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	IntegrationID *uuid.UUID `json:"integration_id,omitempty"`
	ChannelType   string     `json:"channel_type"`
	Enabled       bool       `json:"enabled"`
	StockBuffer   int        `json:"stock_buffer"`
	SyncMode      string     `json:"sync_mode"`
	Priority      int        `json:"priority"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// StockSyncEvent represents a recorded stock change event.
type StockSyncEvent struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	ProductID         *uuid.UUID      `json:"product_id,omitempty"`
	SKU               *string         `json:"sku,omitempty"`
	TriggerType       string          `json:"trigger_type"`
	OldQuantity       int             `json:"old_quantity"`
	NewQuantity       int             `json:"new_quantity"`
	AvailableQuantity int             `json:"available_quantity"`
	ChannelsNotified  int             `json:"channels_notified"`
	ChannelsFailed    int             `json:"channels_failed"`
	Details           json.RawMessage `json:"details"`
	CreatedAt         time.Time       `json:"created_at"`
}

// StockAllocation represents the calculated available stock per channel.
type StockAllocation struct {
	ChannelID         uuid.UUID `json:"channel_id"`
	ChannelType       string    `json:"channel_type"`
	TotalStock        int       `json:"total_stock"`
	Reserved          int       `json:"reserved"`
	Buffer            int       `json:"buffer"`
	AvailableQuantity int       `json:"available_quantity"`
}

// StockSyncDashboard holds overview stats for the stock sync dashboard.
type StockSyncDashboard struct {
	TotalProducts    int              `json:"total_products"`
	ActiveChannels   int              `json:"active_channels"`
	RecentErrors     int              `json:"recent_errors"`
	LastSyncAt       *time.Time       `json:"last_sync_at,omitempty"`
	ChannelSummaries []ChannelSummary `json:"channel_summaries"`
}

// ChannelSummary is a brief status overview of a single sync channel.
type ChannelSummary struct {
	ID          uuid.UUID  `json:"id"`
	ChannelType string     `json:"channel_type"`
	Enabled     bool       `json:"enabled"`
	SyncMode    string     `json:"sync_mode"`
	StockBuffer int        `json:"stock_buffer"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	LastError   *string    `json:"last_error,omitempty"`
	Status      string     `json:"status"`       // ok, warning, error
	ErrorCount  int        `json:"error_count"`  // errors in last 24h
	ItemsSynced int        `json:"items_synced"` // items in last sync
}

// Valid channel types.
var validChannelTypes = map[string]bool{
	"allegro":     true,
	"amazon":      true,
	"woocommerce": true,
	"ebay":        true,
	"shopify":     true,
	"manual":      true,
}

// Valid sync modes.
var validSyncModes = map[string]bool{
	"realtime":  true,
	"scheduled": true,
	"manual":    true,
}

// Valid trigger types.
var validTriggerTypes = map[string]bool{
	"order_placed":       true,
	"order_cancelled":    true,
	"order_shipped":      true,
	"stock_adjusted":     true,
	"manual":             true,
	"manual_update":      true,
	"recount":            true,
	"product_created":    true,
	"product_import":     true,
	"warehouse_document": true,
	"supplier_sync":      true,
	"supplier_delinked":  true,
	"allegro_import":     true,
	"return_restocked":   true,
}

// CreateStockSyncChannelRequest is the payload for creating a stock sync channel.
type CreateStockSyncChannelRequest struct {
	IntegrationID *uuid.UUID `json:"integration_id,omitempty"`
	ChannelType   string     `json:"channel_type"`
	Enabled       *bool      `json:"enabled,omitempty"`
	StockBuffer   *int       `json:"stock_buffer,omitempty"`
	SyncMode      *string    `json:"sync_mode,omitempty"`
	Priority      *int       `json:"priority,omitempty"`
}

// Validate validates the create channel request.
func (r *CreateStockSyncChannelRequest) Validate() error {
	r.ChannelType = strings.TrimSpace(strings.ToLower(r.ChannelType))
	if r.ChannelType == "" {
		return errors.New("channel_type is required")
	}
	if !validChannelTypes[r.ChannelType] {
		return errors.New("invalid channel_type")
	}
	if r.SyncMode != nil {
		mode := strings.TrimSpace(strings.ToLower(*r.SyncMode))
		r.SyncMode = &mode
		if !validSyncModes[mode] {
			return errors.New("invalid sync_mode")
		}
	}
	if r.StockBuffer != nil && *r.StockBuffer < 0 {
		return errors.New("stock_buffer must not be negative")
	}
	return nil
}

// UpdateStockSyncChannelRequest is the payload for updating a stock sync channel.
type UpdateStockSyncChannelRequest struct {
	Enabled     *bool   `json:"enabled,omitempty"`
	StockBuffer *int    `json:"stock_buffer,omitempty"`
	SyncMode    *string `json:"sync_mode,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
}

// Validate validates the update channel request.
func (r *UpdateStockSyncChannelRequest) Validate() error {
	if r.Enabled == nil && r.StockBuffer == nil && r.SyncMode == nil && r.Priority == nil {
		return errors.New("at least one field is required")
	}
	if r.SyncMode != nil {
		mode := strings.TrimSpace(strings.ToLower(*r.SyncMode))
		r.SyncMode = &mode
		if !validSyncModes[mode] {
			return errors.New("invalid sync_mode")
		}
	}
	if r.StockBuffer != nil && *r.StockBuffer < 0 {
		return errors.New("stock_buffer must not be negative")
	}
	return nil
}

// StockSyncChannelListFilter holds filtering/pagination for listing channels.
type StockSyncChannelListFilter struct {
	Enabled     *bool
	ChannelType *string
	PaginationParams
}

// StockSyncEventListFilter holds filtering/pagination for listing events.
type StockSyncEventListFilter struct {
	ProductID   *uuid.UUID
	TriggerType *string
	PaginationParams
}

// IsValidTriggerType checks if a trigger type is valid.
func IsValidTriggerType(t string) bool {
	return validTriggerTypes[t]
}
