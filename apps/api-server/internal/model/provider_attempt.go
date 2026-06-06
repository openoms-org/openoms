package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Provider attempt operations (OPE-417). These name the carrier/provider side
// effect being recorded; they are intentionally provider-agnostic.
const (
	ProviderOpCreateShipment = "create_shipment"
	ProviderOpGenerateLabel  = "generate_label"
	ProviderOpDownloadLabel  = "download_label"
	ProviderOpSyncTracking   = "sync_tracking"
)

// Provider attempt status lifecycle. Mirrors the DB CHECK constraint.
const (
	ProviderAttemptPending   = "pending"
	ProviderAttemptSucceeded = "succeeded"
	ProviderAttemptFailed    = "failed"
)

var validProviderAttemptOperations = []string{
	ProviderOpCreateShipment, ProviderOpGenerateLabel, ProviderOpDownloadLabel, ProviderOpSyncTracking,
}
var validProviderAttemptStatuses = []string{ProviderAttemptPending, ProviderAttemptSucceeded, ProviderAttemptFailed}

// IsValidProviderAttemptOperation reports whether op is a known provider operation.
func IsValidProviderAttemptOperation(op string) bool {
	return slices.Contains(validProviderAttemptOperations, op)
}

// IsValidProviderAttemptStatus reports whether s is a known provider attempt status.
func IsValidProviderAttemptStatus(s string) bool {
	return slices.Contains(validProviderAttemptStatuses, s)
}

// ProviderAttempt records a single carrier/provider interaction (one operation)
// for the fulfillment audit trail. It NEVER stores raw secrets/tokens: only a
// redacted result summary (ResultRedacted) and an optional payload digest
// (PayloadHash). RawProviderStatus preserves the provider's own status string
// alongside any canonical mapping.
type ProviderAttempt struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	ProcessID         *uuid.UUID `json:"process_id,omitempty"`
	Provider          string     `json:"provider"`
	Operation         string     `json:"operation"`
	Status            string     `json:"status"`
	RequestID         string     `json:"request_id,omitempty"`
	CorrelationID     string     `json:"correlation_id,omitempty"`
	RawProviderStatus string     `json:"raw_provider_status,omitempty"`
	ResultRedacted    any        `json:"result_redacted,omitempty"`
	PayloadHash       string     `json:"payload_hash,omitempty"`
	ErrorCode         string     `json:"error_code,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Carrier failure classification (OPE-417). Carrier/provider failures are mapped
// onto these classes so a single helper can translate them to the existing
// fulfillment blocker codes without leaking provider-specific detail.
const (
	CarrierFailMissingData       = "missing_data"
	CarrierFailProviderRejection = "provider_rejection"
	CarrierFailProviderOutage    = "provider_outage"
	CarrierFailRateLimit         = "rate_limit"
	CarrierFailAuth              = "auth"
)

// TrackingStatusMapping is the result of mapping a carrier's raw tracking status
// onto the canonical OMS shipment status. Raw is ALWAYS preserved (even when the
// carrier status is unknown/unmapped) so the original provider value is never lost.
type TrackingStatusMapping struct {
	Raw       string `json:"raw"`                 // the provider's own status string
	Canonical string `json:"canonical,omitempty"` // mapped OMS status, empty when unmapped
	Mapped    bool   `json:"mapped"`              // whether a canonical mapping exists
}

// NewTrackingStatusMapping builds a TrackingStatusMapping, preserving the raw
// provider status regardless of whether a canonical mapping was found. canonical
// is ignored (forced empty) when mapped is false.
func NewTrackingStatusMapping(raw, canonical string, mapped bool) TrackingStatusMapping {
	if !mapped {
		canonical = ""
	}
	return TrackingStatusMapping{Raw: raw, Canonical: canonical, Mapped: mapped}
}

// CarrierFailureBlockerCode maps a carrier failure class onto an existing
// fulfillment blocker code (ADR-424). Unknown/empty classes fall back to a
// degraded-capability blocker so a typed blocker is always produced. The mapping
// reuses the codes defined in fulfillment.go — it introduces no new codes.
func CarrierFailureBlockerCode(failureClass string) string {
	switch failureClass {
	case CarrierFailMissingData:
		// Operator must supply/repair input data before the provider can proceed.
		return BlockerManualStockReviewRequired
	case CarrierFailProviderRejection:
		// The provider was reachable but rejected the request — capability missing
		// for this request shape.
		return BlockerIntegrationCapabilityMissing
	case CarrierFailProviderOutage:
		// Transient provider/integration unavailability — degraded capability.
		return BlockerIntegrationCapabilityDegraded
	case CarrierFailRateLimit:
		// Throttled by the provider — degraded capability, retry later.
		return BlockerIntegrationCapabilityDegraded
	case CarrierFailAuth:
		// Credentials/permissions are invalid — capability missing until fixed.
		return BlockerIntegrationCapabilityMissing
	default:
		return BlockerIntegrationCapabilityDegraded
	}
}
