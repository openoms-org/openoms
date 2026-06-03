package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Validation probe types (spec §182).
const (
	ProbeAuthCheck          = "auth_check"
	ProbeEndpointReach      = "endpoint_reachability"
	ProbeFeedFetch          = "feed_fetch"
	ProbeFeedParse          = "feed_parse"
	ProbeSampleCatalogRead  = "sample_catalog_read"
	ProbeSampleStockRead    = "sample_stock_read"
	ProbeSamplePriceRead    = "sample_price_read"
	ProbeOrderPreflight     = "order_preflight"
	ProbeSandboxOrderCreate = "sandbox_order_create"
	ProbeOrderStatusRead    = "order_status_read"
	ProbeShipmentTracking   = "shipment_tracking_read"
	ProbeInvoiceRead        = "invoice_read"
	ProbeWebhookSignature   = "webhook_signature_verification"
	ProbeMalformedPayload   = "malformed_payload_test"
	ProbeRateLimitBehavior  = "rate_limit_behavior"
)

// Validation environments.
const (
	ValidationEnvSandbox    = "sandbox"
	ValidationEnvProduction = "production"
)

// Validation run verdicts.
const (
	RunVerdictPending = "pending"
	RunVerdictPassed  = "passed"
	RunVerdictFailed  = "failed"
	RunVerdictError   = "error"
)

// Validation result statuses.
const (
	ResultStatusPassed  = "passed"
	ResultStatusFailed  = "failed"
	ResultStatusSkipped = "skipped"
	ResultStatusError   = "error"
)

var validProbeTypes = []string{
	ProbeAuthCheck, ProbeEndpointReach, ProbeFeedFetch, ProbeFeedParse, ProbeSampleCatalogRead,
	ProbeSampleStockRead, ProbeSamplePriceRead, ProbeOrderPreflight, ProbeSandboxOrderCreate,
	ProbeOrderStatusRead, ProbeShipmentTracking, ProbeInvoiceRead, ProbeWebhookSignature,
	ProbeMalformedPayload, ProbeRateLimitBehavior,
}
var validValidationEnvs = []string{ValidationEnvSandbox, ValidationEnvProduction}
var validResultStatuses = []string{ResultStatusPassed, ResultStatusFailed, ResultStatusSkipped, ResultStatusError}

// IsValidProbeType reports whether t is a known validation probe type.
func IsValidProbeType(t string) bool { return slices.Contains(validProbeTypes, t) }

// IsValidValidationEnv reports whether e is a known validation environment.
func IsValidValidationEnv(e string) bool { return slices.Contains(validValidationEnvs, e) }

// IsValidResultStatus reports whether s is a known probe result status.
func IsValidResultStatus(s string) bool { return slices.Contains(validResultStatuses, s) }

// ProviderValidationProbe is a probe declared for a provider version.
type ProviderValidationProbe struct {
	ID                uuid.UUID `json:"id"`
	ProviderVersionID uuid.UUID `json:"provider_version_id"`
	ProbeType         string    `json:"probe_type"`
	Label             string    `json:"label"`
	Destructive       bool      `json:"destructive"`
	Required          bool      `json:"required"`
	Config            any       `json:"config,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ProviderValidationRun is one immutable validation attempt against a version.
type ProviderValidationRun struct {
	ID                uuid.UUID                  `json:"id"`
	ProviderVersionID uuid.UUID                  `json:"provider_version_id"`
	Environment       string                     `json:"environment"`
	Verdict           string                     `json:"verdict"`
	StartedBy         *uuid.UUID                 `json:"started_by,omitempty"`
	StartedAt         time.Time                  `json:"started_at"`
	FinishedAt        *time.Time                 `json:"finished_at,omitempty"`
	Notes             string                     `json:"notes"`
	Results           []ProviderValidationResult `json:"results,omitempty"`
}

// ProviderValidationResult is one probe-level result within a run. Observation
// is a redacted safe summary; PayloadHash correlates without storing raw data.
type ProviderValidationResult struct {
	ID          uuid.UUID `json:"id"`
	RunID       uuid.UUID `json:"run_id"`
	ProbeType   string    `json:"probe_type"`
	Label       string    `json:"label"`
	Status      string    `json:"status"`
	Observation string    `json:"observation"`
	PayloadHash string    `json:"payload_hash"`
	Findings    string    `json:"findings"`
	CreatedAt   time.Time `json:"created_at"`
}

// ValidateProbes checks a probe set: known types and non-empty unique labels.
func ValidateProbes(probes []ProviderValidationProbe) error {
	seen := map[string]bool{}
	for _, p := range probes {
		if p.Label == "" {
			return fmt.Errorf("probe has an empty label")
		}
		if seen[p.Label] {
			return fmt.Errorf("duplicate probe label %q", p.Label)
		}
		seen[p.Label] = true
		if !IsValidProbeType(p.ProbeType) {
			return fmt.Errorf("probe %q has unknown probe_type %q", p.Label, p.ProbeType)
		}
	}
	return nil
}

// VerdictFromResults computes a run verdict: any errored probe -> error, any
// failed probe -> failed, no results -> error, otherwise passed.
func VerdictFromResults(results []ProviderValidationResult) string {
	if len(results) == 0 {
		return RunVerdictError
	}
	verdict := RunVerdictPassed
	for _, r := range results {
		switch r.Status {
		case ResultStatusError:
			return RunVerdictError
		case ResultStatusFailed:
			verdict = RunVerdictFailed
		}
	}
	return verdict
}

// GapForFailedProbe maps a failed/errored probe to an integration gap type and
// severity. Returns ("", "") when the result status is not a failure.
func GapForFailedProbe(probeType, status string) (gapType, severity string) {
	if status != ResultStatusFailed && status != ResultStatusError {
		return "", ""
	}
	switch probeType {
	case ProbeAuthCheck:
		gapType = GapTypeAuthFailure
	case ProbeFeedParse, ProbeMalformedPayload:
		gapType = GapTypeParserFailure
	case ProbeOrderPreflight:
		gapType = GapTypeMissingOrderPreflight
	case ProbeShipmentTracking:
		gapType = GapTypeMissingTracking
	default:
		gapType = GapTypeProviderBusinessError
	}
	if status == ResultStatusError {
		severity = GapSeveritySystemError
	} else {
		severity = GapSeverityActionRequired
	}
	return gapType, severity
}

// HashPayload returns a hex SHA-256 of a payload, for evidence correlation
// without storing the raw (possibly sensitive) bytes.
func HashPayload(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
