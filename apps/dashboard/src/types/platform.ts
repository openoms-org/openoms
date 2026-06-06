// === Provider Integration Studio (platform admin) ===
// Read-only types mirroring the /v1/platform backend (OPE-404..408).

/** Result of GET /v1/platform/me — identifies a platform administrator. */
export interface PlatformMe {
  user_id: string;
  permissions: string[];
}

/**
 * Lifecycle state of a provider version. Mirrors the backend
 * publication_state enum. Ordered roughly research -> retired.
 */
export type ProviderPublicationState =
  | "research"
  | "designed"
  | "adapter_in_progress"
  | "internal_validation"
  | "private_beta"
  | "available"
  | "deprecated"
  | "retired";

export const PROVIDER_PUBLICATION_STATES: ProviderPublicationState[] = [
  "research",
  "designed",
  "adapter_in_progress",
  "internal_validation",
  "private_beta",
  "available",
  "deprecated",
  "retired",
];

/** A provider definition in the studio registry. */
export interface ProviderDefinition {
  id: string;
  provider_key: string;
  display_name: string;
  provider_type: string;
  regions: string[];
  business_domains: string[];
  owner: string;
  notes: string;
  latest_published_version_id: string | null;
  created_at: string;
  updated_at: string;
}

/** A single version of a provider definition. */
export interface ProviderVersion {
  id: string;
  version: string;
  publication_state: ProviderPublicationState;
  changelog: string;
  created_at: string;
}

export interface ProviderDefinitionsResponse {
  definitions: ProviderDefinition[];
}

export interface ProviderVersionsResponse {
  versions: ProviderVersion[];
}

/**
 * Lifecycle states in which a version is "published" and therefore frozen for
 * config edits. Mirrors model.IsPublishedState on the backend — edits to a
 * frozen version are rejected, and require a new draft version instead.
 */
const FROZEN_PUBLICATION_STATES: ReadonlySet<ProviderPublicationState> = new Set([
  "private_beta",
  "available",
  "deprecated",
  "retired",
]);

/** True when a version's state is published/frozen (read-only in the studio). */
export function isPublishedState(
  state: ProviderPublicationState | null | undefined,
): boolean {
  return !!state && FROZEN_PUBLICATION_STATES.has(state);
}

// === Credential & settings schema (OPE-406) ===

/** Field group keys — must match model.validProviderFieldGroupKeys. */
export type ProviderFieldGroupKey =
  | "secret_credentials"
  | "settings"
  | "environment"
  | "sync"
  | "feature_toggles"
  | "provider_options";

export const PROVIDER_FIELD_GROUP_KEYS: ProviderFieldGroupKey[] = [
  "secret_credentials",
  "settings",
  "environment",
  "sync",
  "feature_toggles",
  "provider_options",
];

/** Field types — must match model.validProviderFieldTypes. */
export type ProviderFieldType =
  | "string"
  | "password"
  | "number"
  | "boolean"
  | "enum"
  | "url"
  | "textarea";

export const PROVIDER_FIELD_TYPES: ProviderFieldType[] = [
  "string",
  "password",
  "number",
  "boolean",
  "enum",
  "url",
  "textarea",
];

/** Environment scope — must match model.validProviderFieldEnvScopes ("" == any). */
export type ProviderFieldEnvScope = "" | "all" | "production" | "sandbox";

export const PROVIDER_FIELD_ENV_SCOPES: ProviderFieldEnvScope[] = [
  "all",
  "production",
  "sandbox",
];

export interface ProviderFieldValidation {
  enum?: string[];
  regex?: string;
  min?: number;
  max?: number;
  min_length?: number;
  max_length?: number;
}

export interface ProviderField {
  key: string;
  label: string;
  type: ProviderFieldType;
  required: boolean;
  secret: boolean;
  environment_scope?: ProviderFieldEnvScope;
  help_text?: string;
  validation: ProviderFieldValidation;
  capability_enabled?: string;
  test_connection_dependency?: boolean;
}

export interface ProviderFieldGroup {
  key: ProviderFieldGroupKey;
  label: string;
  fields: ProviderField[];
}

export interface ProviderFieldSchema {
  id: string;
  provider_version_id: string;
  groups: ProviderFieldGroup[];
  created_at: string;
  updated_at: string;
}

export interface ProviderSchemaUpdateRequest {
  groups: ProviderFieldGroup[];
}

// === Capabilities (OPE-407) ===

/** Capability support states — must match model.validSupportStatuses. */
export type SupportStatus =
  | "supported"
  | "configured"
  | "unsupported"
  | "requires_manual"
  | "degraded"
  | "unknown";

export const SUPPORT_STATUSES: SupportStatus[] = [
  "supported",
  "configured",
  "requires_manual",
  "degraded",
  "unsupported",
  "unknown",
];

export interface ProviderCapability {
  id?: string;
  provider_version_id?: string;
  capability_key: string;
  support_status: SupportStatus;
  channel: string;
  mode: string;
  freshness: string;
  required_inputs: string[];
  provided_outputs: string[];
  latency_sla_seconds?: number | null;
  /** Customer-facing evidence link / probe reference. Held in notes. */
  notes: string;
  created_at?: string;
  updated_at?: string;
}

export interface ProviderCapabilitiesResponse {
  capabilities: ProviderCapability[];
}

export interface ProviderCapabilitiesUpdateRequest {
  capabilities: ProviderCapability[];
}

// === Status mappings (OPE-407) ===

/** Status mapping domains — must match model.validStatusDomains. */
export type StatusDomain = "order" | "line" | "shipment";

export const STATUS_DOMAINS: StatusDomain[] = ["order", "line", "shipment"];

export type MappingConfidence = "high" | "medium" | "low";

export const MAPPING_CONFIDENCES: MappingConfidence[] = ["high", "medium", "low"];

export interface ProviderStatusMapping {
  id?: string;
  provider_version_id?: string;
  status_domain: StatusDomain;
  raw_status: string;
  canonical_status: string;
  canonical_event_type: string;
  canonical_step_key: string;
  confidence: MappingConfidence;
  is_terminal: boolean;
  notes: string;
  created_at?: string;
  updated_at?: string;
}

export interface ProviderStatusMappingsResponse {
  status_mappings: ProviderStatusMapping[];
}

export interface ProviderStatusMappingsUpdateRequest {
  status_mappings: ProviderStatusMapping[];
}

// === Integration gaps (OPE-407) ===

/** Gap types — must match model.validGapTypes. */
export type GapType =
  | "missing_source_docs"
  | "missing_credential_field"
  | "missing_status_mapping"
  | "unsupported_capability"
  | "stale_data_risk"
  | "missing_tracking"
  | "missing_order_preflight"
  | "ambiguous_product_identity"
  | "auth_failure"
  | "provider_business_error"
  | "parser_failure"
  | "manual_fallback_required";

export const GAP_TYPES: GapType[] = [
  "missing_source_docs",
  "missing_credential_field",
  "missing_status_mapping",
  "unsupported_capability",
  "stale_data_risk",
  "missing_tracking",
  "missing_order_preflight",
  "ambiguous_product_identity",
  "auth_failure",
  "provider_business_error",
  "parser_failure",
  "manual_fallback_required",
];

export type GapSeverity = "info" | "warning" | "action_required" | "system_error";

export const GAP_SEVERITIES: GapSeverity[] = [
  "info",
  "warning",
  "action_required",
  "system_error",
];

export type GapStatus = "open" | "acknowledged" | "resolved";

export const GAP_STATUSES: GapStatus[] = ["open", "acknowledged", "resolved"];

export interface ProviderIntegrationGap {
  id: string;
  provider_version_id: string;
  gap_type: GapType;
  severity: GapSeverity;
  status: GapStatus;
  description: string;
  created_at: string;
  updated_at: string;
  resolved_at?: string | null;
}

export interface ProviderGapsResponse {
  gaps: ProviderIntegrationGap[];
}

export interface ProviderCreateGapRequest {
  gap_type: GapType;
  severity: GapSeverity;
  description: string;
}

export interface ProviderUpdateGapRequest {
  status: GapStatus;
}
