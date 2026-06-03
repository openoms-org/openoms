package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Provider publication lifecycle states (design spec §570).
const (
	ProviderStateResearch           = "research"
	ProviderStateDesigned           = "designed"
	ProviderStateAdapterInProgress  = "adapter_in_progress"
	ProviderStateInternalValidation = "internal_validation"
	ProviderStatePrivateBeta        = "private_beta"
	ProviderStateAvailable          = "available"
	ProviderStateDeprecated         = "deprecated"
	ProviderStateRetired            = "retired"
)

// Provider family types.
const (
	ProviderTypeMarketplace = "marketplace"
	ProviderTypeCarrier     = "carrier"
	ProviderTypeSupplier    = "supplier"
	ProviderTypeShop        = "shop"
	ProviderTypeInvoicing   = "invoicing"
	ProviderTypeEDI         = "edi"
	ProviderTypeOther       = "other"
)

// Publication event actions.
const (
	ProviderEventCreate           = "create"
	ProviderEventTransition       = "transition"
	ProviderEventEmergencyDisable = "emergency_disable"
)

var validProviderTypes = []string{
	ProviderTypeMarketplace, ProviderTypeCarrier, ProviderTypeSupplier,
	ProviderTypeShop, ProviderTypeInvoicing, ProviderTypeEDI, ProviderTypeOther,
}

var validProviderStates = []string{
	ProviderStateResearch, ProviderStateDesigned, ProviderStateAdapterInProgress,
	ProviderStateInternalValidation, ProviderStatePrivateBeta, ProviderStateAvailable,
	ProviderStateDeprecated, ProviderStateRetired,
}

// allowedProviderTransitions is the authoritative lifecycle graph. A transition
// is legal only if `to` is listed for `from`. `retired` is terminal.
var allowedProviderTransitions = map[string][]string{
	ProviderStateResearch:           {ProviderStateDesigned},
	ProviderStateDesigned:           {ProviderStateAdapterInProgress},
	ProviderStateAdapterInProgress:  {ProviderStateInternalValidation},
	ProviderStateInternalValidation: {ProviderStatePrivateBeta, ProviderStateDesigned},
	ProviderStatePrivateBeta:        {ProviderStateAvailable, ProviderStateInternalValidation},
	ProviderStateAvailable:          {ProviderStateDeprecated, ProviderStateInternalValidation},
	ProviderStateDeprecated:         {ProviderStateRetired},
	ProviderStateRetired:            {},
}

// IsValidProviderType reports whether t is a known provider type.
func IsValidProviderType(t string) bool { return slices.Contains(validProviderTypes, t) }

// IsValidPublicationState reports whether s is a known lifecycle state.
func IsValidPublicationState(s string) bool { return slices.Contains(validProviderStates, s) }

// CanTransition reports whether moving a version from -> to is a legal
// lifecycle transition.
func CanTransition(from, to string) bool {
	return slices.Contains(allowedProviderTransitions[from], to)
}

// IsPublishedState reports whether a version in state s is "published" and thus
// frozen for metadata edits (only controlled publication transitions allowed).
func IsPublishedState(s string) bool {
	switch s {
	case ProviderStatePrivateBeta, ProviderStateAvailable, ProviderStateDeprecated, ProviderStateRetired:
		return true
	default:
		return false
	}
}

// ProviderDefinition is a provider family (e.g. "bigbuy", "inpost", "allegro").
type ProviderDefinition struct {
	ID                       uuid.UUID  `json:"id"`
	ProviderKey              string     `json:"provider_key"`
	DisplayName              string     `json:"display_name"`
	ProviderType             string     `json:"provider_type"`
	Regions                  []string   `json:"regions"`
	BusinessDomains          []string   `json:"business_domains"`
	Owner                    string     `json:"owner"`
	SourceLinks              []string   `json:"source_links"`
	Notes                    string     `json:"notes"`
	LatestPublishedVersionID *uuid.UUID `json:"latest_published_version_id,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// ProviderVersion is a versioned snapshot of a provider's configuration.
type ProviderVersion struct {
	ID                   uuid.UUID  `json:"id"`
	ProviderDefinitionID uuid.UUID  `json:"provider_definition_id"`
	Version              string     `json:"version"`
	PublicationState     string     `json:"publication_state"`
	Changelog            string     `json:"changelog"`
	CompatibilityNotes   string     `json:"compatibility_notes"`
	CreatedBy            *uuid.UUID `json:"created_by,omitempty"`
	PublishedBy          *uuid.UUID `json:"published_by,omitempty"`
	PublishedAt          *time.Time `json:"published_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ProviderPublicationEvent is an append-only audit record of a lifecycle change.
type ProviderPublicationEvent struct {
	ID                int64      `json:"id"`
	ProviderVersionID uuid.UUID  `json:"provider_version_id"`
	FromState         *string    `json:"from_state,omitempty"`
	ToState           string     `json:"to_state"`
	Action            string     `json:"action"`
	ActorUserID       *uuid.UUID `json:"actor_user_id,omitempty"`
	Reason            string     `json:"reason"`
	CreatedAt         time.Time  `json:"created_at"`
}

// ProviderTenantEnable is a private-beta allowlist entry (version visible to a tenant).
type ProviderTenantEnable struct {
	ID                uuid.UUID  `json:"id"`
	ProviderVersionID uuid.UUID  `json:"provider_version_id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	EnabledBy         *uuid.UUID `json:"enabled_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
