package model

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Capability support states (OPE-420 Task 5).
const (
	SupportStatusSupported      = "supported"
	SupportStatusConfigured     = "configured"
	SupportStatusUnsupported    = "unsupported"
	SupportStatusRequiresManual = "requires_manual"
	SupportStatusDegraded       = "degraded"
	SupportStatusUnknown        = "unknown"
)

// Status mapping domains (spec §170: order/shipment/line are separate).
const (
	StatusDomainOrder    = "order"
	StatusDomainShipment = "shipment"
	StatusDomainLine     = "line"
)

// Status mapping confidence.
const (
	MappingConfidenceHigh   = "high"
	MappingConfidenceMedium = "medium"
	MappingConfidenceLow    = "low"
)

// Integration gap types (spec §223).
const (
	GapTypeMissingSourceDocs      = "missing_source_docs"
	GapTypeMissingCredentialField = "missing_credential_field"
	GapTypeMissingStatusMapping   = "missing_status_mapping"
	GapTypeUnsupportedCapability  = "unsupported_capability"
	GapTypeStaleDataRisk          = "stale_data_risk"
	GapTypeMissingTracking        = "missing_tracking"
	GapTypeMissingOrderPreflight  = "missing_order_preflight"
	GapTypeAmbiguousProductID     = "ambiguous_product_identity"
	GapTypeAuthFailure            = "auth_failure"
	GapTypeProviderBusinessError  = "provider_business_error"
	GapTypeParserFailure          = "parser_failure"
	GapTypeManualFallbackRequired = "manual_fallback_required"
)

// Gap severities (OPE-420 Task 5).
const (
	GapSeverityInfo           = "info"
	GapSeverityWarning        = "warning"
	GapSeverityActionRequired = "action_required"
	GapSeveritySystemError    = "system_error"
)

// Gap lifecycle statuses.
const (
	GapStatusOpen         = "open"
	GapStatusAcknowledged = "acknowledged"
	GapStatusResolved     = "resolved"
)

var validSupportStatuses = []string{
	SupportStatusSupported, SupportStatusConfigured, SupportStatusUnsupported,
	SupportStatusRequiresManual, SupportStatusDegraded, SupportStatusUnknown,
}
var validStatusDomains = []string{StatusDomainOrder, StatusDomainShipment, StatusDomainLine}
var validMappingConfidences = []string{MappingConfidenceHigh, MappingConfidenceMedium, MappingConfidenceLow}
var validGapTypes = []string{
	GapTypeMissingSourceDocs, GapTypeMissingCredentialField, GapTypeMissingStatusMapping,
	GapTypeUnsupportedCapability, GapTypeStaleDataRisk, GapTypeMissingTracking,
	GapTypeMissingOrderPreflight, GapTypeAmbiguousProductID, GapTypeAuthFailure,
	GapTypeProviderBusinessError, GapTypeParserFailure, GapTypeManualFallbackRequired,
}
var validGapSeverities = []string{GapSeverityInfo, GapSeverityWarning, GapSeverityActionRequired, GapSeveritySystemError}
var validGapStatuses = []string{GapStatusOpen, GapStatusAcknowledged, GapStatusResolved}

// IsValidSupportStatus reports whether s is a known capability support status.
func IsValidSupportStatus(s string) bool { return slices.Contains(validSupportStatuses, s) }

// IsValidStatusDomain reports whether d is a known status-mapping domain.
func IsValidStatusDomain(d string) bool { return slices.Contains(validStatusDomains, d) }

// IsValidMappingConfidence reports whether c is a known mapping confidence.
func IsValidMappingConfidence(c string) bool { return slices.Contains(validMappingConfidences, c) }

// IsValidGapType reports whether t is a known integration-gap type.
func IsValidGapType(t string) bool { return slices.Contains(validGapTypes, t) }

// IsValidGapSeverity reports whether s is a known gap severity.
func IsValidGapSeverity(s string) bool { return slices.Contains(validGapSeverities, s) }

// IsValidGapStatus reports whether s is a known gap status.
func IsValidGapStatus(s string) bool { return slices.Contains(validGapStatuses, s) }

// ProviderCapability declares one capability of a provider version.
type ProviderCapability struct {
	ID                uuid.UUID `json:"id"`
	ProviderVersionID uuid.UUID `json:"provider_version_id"`
	CapabilityKey     string    `json:"capability_key"`
	SupportStatus     string    `json:"support_status"`
	Channel           string    `json:"channel"`
	Mode              string    `json:"mode"`
	Freshness         string    `json:"freshness"`
	RequiredInputs    []string  `json:"required_inputs"`
	ProvidedOutputs   []string  `json:"provided_outputs"`
	LatencySLASeconds *int      `json:"latency_sla_seconds,omitempty"`
	Notes             string    `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ProviderStatusMapping maps one raw provider status to a canonical status.
type ProviderStatusMapping struct {
	ID                 uuid.UUID `json:"id"`
	ProviderVersionID  uuid.UUID `json:"provider_version_id"`
	StatusDomain       string    `json:"status_domain"`
	RawStatus          string    `json:"raw_status"`
	CanonicalStatus    string    `json:"canonical_status"`
	CanonicalEventType string    `json:"canonical_event_type"`
	CanonicalStepKey   string    `json:"canonical_step_key"`
	Confidence         string    `json:"confidence"`
	IsTerminal         bool      `json:"is_terminal"`
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ProviderIntegrationGap is an issue that may block safe publication or full automation.
type ProviderIntegrationGap struct {
	ID                uuid.UUID  `json:"id"`
	ProviderVersionID uuid.UUID  `json:"provider_version_id"`
	GapType           string     `json:"gap_type"`
	Severity          string     `json:"severity"`
	Status            string     `json:"status"`
	Description       string     `json:"description"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
}

// ValidateCapabilities checks a capability set: known support statuses, non-empty
// unique capability keys, and non-negative latency.
func ValidateCapabilities(caps []ProviderCapability) error {
	seen := map[string]bool{}
	for _, c := range caps {
		if c.CapabilityKey == "" {
			return fmt.Errorf("capability has an empty capability_key")
		}
		if seen[c.CapabilityKey] {
			return fmt.Errorf("duplicate capability_key %q", c.CapabilityKey)
		}
		seen[c.CapabilityKey] = true
		if !IsValidSupportStatus(c.SupportStatus) {
			return fmt.Errorf("capability %q has unknown support_status %q", c.CapabilityKey, c.SupportStatus)
		}
		if c.LatencySLASeconds != nil && *c.LatencySLASeconds < 0 {
			return fmt.Errorf("capability %q has negative latency_sla_seconds", c.CapabilityKey)
		}
	}
	return nil
}

// ValidateStatusMappings checks a mapping set: known domains/confidence, non-empty
// raw statuses, and unique (domain, raw_status) pairs.
func ValidateStatusMappings(mappings []ProviderStatusMapping) error {
	seen := map[string]bool{}
	for _, m := range mappings {
		if m.RawStatus == "" {
			return fmt.Errorf("status mapping has an empty raw_status")
		}
		if !IsValidStatusDomain(m.StatusDomain) {
			return fmt.Errorf("status mapping for %q has unknown status_domain %q", m.RawStatus, m.StatusDomain)
		}
		if !IsValidMappingConfidence(m.Confidence) {
			return fmt.Errorf("status mapping for %q has unknown confidence %q", m.RawStatus, m.Confidence)
		}
		key := m.StatusDomain + "\x00" + m.RawStatus
		if seen[key] {
			return fmt.Errorf("duplicate status mapping for domain %q raw_status %q", m.StatusDomain, m.RawStatus)
		}
		seen[key] = true
	}
	return nil
}
