package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Availability types (how precise the supplier's quantity signal is).
const (
	AvailabilityExactQuantity = "exact_quantity"
	AvailabilityBucket        = "bucket"
	AvailabilityBoolean       = "boolean"
	AvailabilityETAOnly       = "eta_only"
	AvailabilityUnknown       = "unknown"
)

var validAvailabilityTypes = []string{
	AvailabilityExactQuantity, AvailabilityBucket, AvailabilityBoolean, AvailabilityETAOnly, AvailabilityUnknown,
}

// IsValidAvailabilityType reports whether t is a known availability type.
func IsValidAvailabilityType(t string) bool { return slices.Contains(validAvailabilityTypes, t) }

// Policy scopes (precedence: channel > listing > product > supplier).
const (
	PolicyScopeSupplier = "supplier"
	PolicyScopeProduct  = "product"
	PolicyScopeListing  = "listing"
	PolicyScopeChannel  = "channel"
)

var validPolicyScopes = []string{PolicyScopeSupplier, PolicyScopeProduct, PolicyScopeListing, PolicyScopeChannel}

// IsValidPolicyScope reports whether s is a known policy scope.
func IsValidPolicyScope(s string) bool { return slices.Contains(validPolicyScopes, s) }

// Policy modes.
const (
	PolicyModeAuto   = "auto"
	PolicyModeManual = "manual"
	PolicyModePaused = "paused"
)

var validPolicyModes = []string{PolicyModeAuto, PolicyModeManual, PolicyModePaused}

// IsValidPolicyMode reports whether m is a known policy mode.
func IsValidPolicyMode(m string) bool { return slices.Contains(validPolicyModes, m) }

// SupplierAvailability is the raw, observational availability of a supplier product at
// a supplier warehouse (one row per tenant x supplier_product x warehouse_external_id).
type SupplierAvailability struct {
	ID                    uuid.UUID  `json:"id"`
	TenantID              uuid.UUID  `json:"tenant_id"`
	SupplierID            uuid.UUID  `json:"supplier_id"`
	SupplierProductID     uuid.UUID  `json:"supplier_product_id"`
	ProductID             *uuid.UUID `json:"product_id,omitempty"`
	WarehouseExternalID   string     `json:"warehouse_external_id"`
	SourceQuantity        int        `json:"source_quantity"`
	AvailabilityType      string     `json:"availability_type"`
	MinHandlingDays       *int       `json:"min_handling_days,omitempty"`
	MaxHandlingDays       *int       `json:"max_handling_days,omitempty"`
	NextDeliveryDate      *time.Time `json:"next_delivery_date,omitempty"`
	ReservationSupported  bool       `json:"reservation_supported"`
	FreshnessObservedAt   time.Time  `json:"freshness_observed_at"`
	SourceMaxStaleSeconds *int       `json:"source_max_stale_seconds,omitempty"`
	LastSuccessfulSyncID  *uuid.UUID `json:"last_successful_sync_id,omitempty"`
	Raw                   []byte     `json:"raw,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// SupplierAvailabilityPolicy is one tenant rule at one of the four scopes. Only the ref
// column for its scope is set (supplier_id/product_id/listing_id/channel).
type SupplierAvailabilityPolicy struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Scope                string     `json:"scope"`
	SupplierID           *uuid.UUID `json:"supplier_id,omitempty"`
	ProductID            *uuid.UUID `json:"product_id,omitempty"`
	ListingID            *uuid.UUID `json:"listing_id,omitempty"`
	Channel              *string    `json:"channel,omitempty"`
	Mode                 string     `json:"mode"`
	SafetyBuffer         int        `json:"safety_buffer"`
	FreshnessWindowSecs  *int       `json:"freshness_window_seconds,omitempty"`
	MaxLeadTimeDays      *int       `json:"max_lead_time_days,omitempty"`
	OverrideQuantity     *int       `json:"override_quantity,omitempty"`
	AllowChannelIncrease bool       `json:"allow_channel_increase"`
	RequireReservation   bool       `json:"require_reservation"`
	RequirePreflight     bool       `json:"require_preflight"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
