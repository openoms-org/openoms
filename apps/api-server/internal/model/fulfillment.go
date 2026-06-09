package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Fulfillment process aggregate status (ADR-424, operator-visible progress).
const (
	ProcessStatusNew             = "new"
	ProcessStatusValidating      = "validating"
	ProcessStatusReady           = "ready"
	ProcessStatusInProgress      = "in_progress"
	ProcessStatusWaitingExternal = "waiting_external"
	ProcessStatusBlocked         = "blocked"
	ProcessStatusCompleted       = "completed"
	ProcessStatusCancelled       = "cancelled"
)

// Fulfillment process health status (independent from progress).
const (
	ProcessHealthOK             = "ok"
	ProcessHealthWarning        = "warning"
	ProcessHealthActionRequired = "action_required"
	ProcessHealthSystemError    = "system_error"
)

// Fulfillment unit types.
const (
	UnitTypeWarehouse  = "warehouse"
	UnitTypeDropship   = "dropship"
	UnitTypeBackorder  = "backorder"
	UnitTypeMixedChild = "mixed_child"
	UnitTypeManual     = "manual"
)

// Fulfillment unit/step status (shared lifecycle values).
const (
	FulfillmentStatusPending         = "pending"
	FulfillmentStatusReady           = "ready"
	FulfillmentStatusRunning         = "running"
	FulfillmentStatusWaitingExternal = "waiting_external"
	FulfillmentStatusBlocked         = "blocked"
	FulfillmentStatusSucceeded       = "succeeded"
	FulfillmentStatusFailed          = "failed"
	FulfillmentStatusCancelled       = "cancelled"
	FulfillmentStatusSkipped         = "skipped"
)

// Canonical, provider-agnostic step keys (ADR-424).
const (
	StepValidateOrder           = "validate_order"
	StepSelectFulfillmentPolicy = "select_fulfillment_policy"
	StepAllocateUnits           = "allocate_units"
	StepReserveStock            = "reserve_stock"
	StepReleaseStock            = "release_stock"
	StepRecalculateAvailability = "recalculate_availability"
	StepPropagateStock          = "propagate_stock_to_channels"
	StepCreatePurchaseOrder     = "create_purchase_order"
	StepReceivePurchaseOrder    = "receive_purchase_order"
	StepCreateDropshipOrder     = "create_dropship_order"
	StepPreflightSupplierOrder  = "preflight_supplier_order"
	StepConfirmSupplierOrder    = "confirm_supplier_order"
	StepPickItems               = "pick_items"
	StepPackItems               = "pack_items"
	StepSelectCarrierService    = "select_carrier_service"
	StepCreateShipment          = "create_shipment"
	StepGenerateLabel           = "generate_label"
	StepCreateDispatchOrder     = "create_dispatch_order"
	StepSyncTrackingToMkt       = "sync_tracking_to_marketplace"
	StepNotifyCustomer          = "notify_customer"
	StepAwaitTracking           = "await_tracking"
	StepCloseProcess            = "close_process"
)

// Blocker codes (ADR-424).
const (
	BlockerStockSyncFailed                  = "stock_sync_failed"
	BlockerChannelStockStale                = "channel_stock_stale"
	BlockerSupplierAvailabilityStale        = "supplier_availability_stale"
	BlockerSupplierAvailabilityUnknown      = "supplier_availability_unknown"
	BlockerSupplierPreflightRequired        = "supplier_preflight_required"
	BlockerSupplierAvailabilityInsufficient = "supplier_availability_insufficient"
	BlockerManualStockReviewRequired        = "manual_stock_review_required"
	BlockerStockWriteUnsupported            = "stock_write_unsupported"
	BlockerStockAckMissing                  = "stock_ack_missing"
	BlockerExternalStatusUnmapped           = "external_status_unmapped"
	BlockerIntegrationCapabilityMissing     = "integration_capability_missing"
	BlockerIntegrationCapabilityDegraded    = "integration_capability_degraded"
	// BlockerAutomationActionFailed marks a permanently failed orchestrated
	// automation action (e.g. set_status with an unknown target status). Opened by
	// the orchestration worker so operators see why an automation never applied
	// (OPE-421). fulfillment_blockers.code has no DB CHECK constraint — codes are
	// app-validated via IsValidBlockerCode — so no migration is required.
	BlockerAutomationActionFailed = "automation_action_failed"
)

// Blocker lifecycle status.
const (
	BlockerStatusOpen         = "open"
	BlockerStatusAcknowledged = "acknowledged"
	BlockerStatusResolved     = "resolved"
)

var validAggregateStatuses = []string{
	ProcessStatusNew, ProcessStatusValidating, ProcessStatusReady, ProcessStatusInProgress,
	ProcessStatusWaitingExternal, ProcessStatusBlocked, ProcessStatusCompleted, ProcessStatusCancelled,
}
var validHealthStatuses = []string{ProcessHealthOK, ProcessHealthWarning, ProcessHealthActionRequired, ProcessHealthSystemError}
var validUnitTypes = []string{UnitTypeWarehouse, UnitTypeDropship, UnitTypeBackorder, UnitTypeMixedChild, UnitTypeManual}
var validFulfillmentStatuses = []string{
	FulfillmentStatusPending, FulfillmentStatusReady, FulfillmentStatusRunning, FulfillmentStatusWaitingExternal,
	FulfillmentStatusBlocked, FulfillmentStatusSucceeded, FulfillmentStatusFailed, FulfillmentStatusCancelled, FulfillmentStatusSkipped,
}
var validFulfillmentStepKeys = []string{
	StepValidateOrder, StepSelectFulfillmentPolicy, StepAllocateUnits, StepReserveStock, StepReleaseStock,
	StepRecalculateAvailability, StepPropagateStock, StepCreatePurchaseOrder, StepReceivePurchaseOrder,
	StepCreateDropshipOrder, StepPreflightSupplierOrder, StepConfirmSupplierOrder, StepPickItems, StepPackItems,
	StepSelectCarrierService, StepCreateShipment, StepGenerateLabel, StepCreateDispatchOrder, StepSyncTrackingToMkt,
	StepNotifyCustomer, StepAwaitTracking, StepCloseProcess,
}
var validBlockerStatuses = []string{BlockerStatusOpen, BlockerStatusAcknowledged, BlockerStatusResolved}

// blockerCategories maps each blocker code to its category.
var blockerCategories = map[string]string{
	BlockerStockSyncFailed:                  "integration",
	BlockerChannelStockStale:                "integration",
	BlockerSupplierAvailabilityStale:        "supplier",
	BlockerSupplierAvailabilityUnknown:      "supplier",
	BlockerSupplierPreflightRequired:        "supplier",
	BlockerSupplierAvailabilityInsufficient: "supplier",
	BlockerManualStockReviewRequired:        "operator",
	BlockerStockWriteUnsupported:            "capability",
	BlockerStockAckMissing:                  "capability",
	BlockerExternalStatusUnmapped:           "mapping",
	BlockerIntegrationCapabilityMissing:     "capability",
	BlockerIntegrationCapabilityDegraded:    "capability",
	BlockerAutomationActionFailed:           "automation",
}

// IsValidAggregateStatus reports whether s is a known process aggregate status.
func IsValidAggregateStatus(s string) bool { return slices.Contains(validAggregateStatuses, s) }

// IsValidHealthStatus reports whether s is a known process health status.
func IsValidHealthStatus(s string) bool { return slices.Contains(validHealthStatuses, s) }

// IsValidUnitType reports whether t is a known fulfillment unit type.
func IsValidUnitType(t string) bool { return slices.Contains(validUnitTypes, t) }

// IsValidFulfillmentStatus reports whether s is a known unit/step status.
func IsValidFulfillmentStatus(s string) bool { return slices.Contains(validFulfillmentStatuses, s) }

// IsValidStepKey reports whether k is a canonical fulfillment step key.
func IsValidStepKey(k string) bool { return slices.Contains(validFulfillmentStepKeys, k) }

// IsValidBlockerCode reports whether c is a known blocker code.
func IsValidBlockerCode(c string) bool { _, ok := blockerCategories[c]; return ok }

// IsValidBlockerStatus reports whether s is a known blocker status.
func IsValidBlockerStatus(s string) bool { return slices.Contains(validBlockerStatuses, s) }

// BlockerCategory returns the category for a blocker code, or "" if unknown.
func BlockerCategory(code string) string { return blockerCategories[code] }

// FulfillmentProcess is the per-order fulfillment aggregate (ADR-424).
type FulfillmentProcess struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	OrderID         uuid.UUID `json:"order_id"`
	AggregateStatus string    `json:"aggregate_status"`
	HealthStatus    string    `json:"health_status"`
	Metadata        any       `json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// FulfillmentUnit is a fulfillable slice of an order's fulfillment process.
type FulfillmentUnit struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	ProcessID    uuid.UUID  `json:"process_id"`
	ParentUnitID *uuid.UUID `json:"parent_unit_id,omitempty"`
	UnitType     string     `json:"unit_type"`
	Status       string     `json:"status"`
	Metadata     any        `json:"metadata,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// FulfillmentStep is a canonical step within a unit.
type FulfillmentStep struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UnitID    uuid.UUID `json:"unit_id"`
	StepKey   string    `json:"step_key"`
	Status    string    `json:"status"`
	Attempts  int       `json:"attempts"`
	Metadata  any       `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FulfillmentBlocker is a typed issue preventing a process/unit from advancing.
type FulfillmentBlocker struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ProcessID   uuid.UUID  `json:"process_id"`
	UnitID      *uuid.UUID `json:"unit_id,omitempty"`
	Code        string     `json:"code"`
	Category    string     `json:"category"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}
