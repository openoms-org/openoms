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

// === Validation engine (OPE-408) ===

/** Probe types — must match model.validProbeTypes on the backend. */
export type ProbeType =
  | "auth_check"
  | "endpoint_reachability"
  | "feed_fetch"
  | "feed_parse"
  | "sample_catalog_read"
  | "sample_stock_read"
  | "sample_price_read"
  | "order_preflight"
  | "sandbox_order_create"
  | "order_status_read"
  | "shipment_tracking_read"
  | "invoice_read"
  | "webhook_signature_verification"
  | "malformed_payload_test"
  | "rate_limit_behavior";

export const PROBE_TYPES: ProbeType[] = [
  "auth_check",
  "endpoint_reachability",
  "feed_fetch",
  "feed_parse",
  "sample_catalog_read",
  "sample_stock_read",
  "sample_price_read",
  "order_preflight",
  "sandbox_order_create",
  "order_status_read",
  "shipment_tracking_read",
  "invoice_read",
  "webhook_signature_verification",
  "malformed_payload_test",
  "rate_limit_behavior",
];

/** Validation environments — must match model.validValidationEnvs. */
export type ValidationEnvironment = "sandbox" | "production";

export const VALIDATION_ENVIRONMENTS: ValidationEnvironment[] = [
  "sandbox",
  "production",
];

/** Run verdicts — must match model RunVerdict* constants. */
export type RunVerdict = "pending" | "passed" | "failed" | "error";

/** Probe result statuses — must match model.validResultStatuses. */
export type ResultStatus = "passed" | "failed" | "skipped" | "error";

export const RESULT_STATUSES: ResultStatus[] = [
  "passed",
  "failed",
  "skipped",
  "error",
];

/**
 * Safety class of a probe. NOT a persisted backend column: derived UI-side from
 * the probe type plus its `destructive` flag (Screen 8 safety levels). The
 * backend authoritative control is the per-run `allow_destructive` flag.
 */
export type ProbeSafetyClass =
  | "read_only"
  | "sandbox_write"
  | "production_write"
  | "destructive";

export const PROBE_SAFETY_CLASSES: ProbeSafetyClass[] = [
  "read_only",
  "sandbox_write",
  "production_write",
  "destructive",
];

/** A probe declared for a provider version. */
export interface ProviderValidationProbe {
  id: string;
  provider_version_id: string;
  probe_type: ProbeType;
  label: string;
  destructive: boolean;
  required: boolean;
  config?: unknown;
  created_at: string;
  updated_at: string;
}

export interface ProviderProbesResponse {
  probes: ProviderValidationProbe[];
}

/**
 * One probe-level result within a run. `observation` is a redacted safe summary;
 * `payload_hash` correlates evidence without storing raw (possibly sensitive)
 * data — see Evidence/Retention contract §9.
 */
export interface ProviderValidationResult {
  id: string;
  run_id: string;
  probe_type: ProbeType | string;
  label: string;
  status: ResultStatus;
  observation: string;
  payload_hash: string;
  findings: string;
  created_at: string;
}

/** One immutable validation attempt against a version. */
export interface ProviderValidationRun {
  id: string;
  provider_version_id: string;
  environment: ValidationEnvironment | string;
  verdict: RunVerdict;
  started_by?: string | null;
  started_at: string;
  finished_at?: string | null;
  notes: string;
  results?: ProviderValidationResult[];
}

export interface ProviderValidationRunsResponse {
  runs: ProviderValidationRun[];
}

export interface StartValidationRunRequest {
  environment: ValidationEnvironment;
  allow_destructive: boolean;
}

export interface RecordValidationResultRequest {
  probe_type: ProbeType | string;
  label: string;
  status: ResultStatus;
  observation: string;
  payload_hash: string;
  findings: string;
}

// === Lifecycle / publication (OPE-405) ===

/** Publication-event action — must match model ProviderEvent* constants. */
export type PublicationEventAction =
  | "create"
  | "transition"
  | "emergency_disable";

/** Append-only audit record of a lifecycle change. */
export interface ProviderPublicationEvent {
  id: number;
  provider_version_id: string;
  from_state?: string | null;
  to_state: ProviderPublicationState;
  action: PublicationEventAction | string;
  actor_user_id?: string | null;
  reason: string;
  created_at: string;
}

export interface ProviderPublicationEventsResponse {
  events: ProviderPublicationEvent[];
}

/** A private-beta allowlist entry returned by POST .../enable-tenant. */
export interface ProviderTenantEnable {
  id: string;
  provider_version_id: string;
  tenant_id: string;
  enabled_by?: string | null;
  created_at: string;
}

export interface PublishRequest {
  to_state: ProviderPublicationState;
  reason: string;
}

export interface EmergencyDisableRequest {
  reason: string;
}

export interface EnableTenantRequest {
  tenant_id: string;
}

/**
 * Authoritative lifecycle graph, mirrored from model.allowedProviderTransitions.
 * A transition from -> to is legal only when `to` is listed for `from`.
 * `retired` is terminal. Used to derive which publication actions are offered.
 */
export const ALLOWED_PROVIDER_TRANSITIONS: Record<
  ProviderPublicationState,
  ProviderPublicationState[]
> = {
  research: ["designed"],
  designed: ["adapter_in_progress"],
  adapter_in_progress: ["internal_validation"],
  internal_validation: ["private_beta", "designed"],
  private_beta: ["available", "internal_validation"],
  available: ["deprecated", "internal_validation"],
  deprecated: ["retired"],
  retired: [],
};

/** True when a from -> to lifecycle transition is legal (mirror of CanTransition). */
export function canTransition(
  from: ProviderPublicationState,
  to: ProviderPublicationState,
): boolean {
  return ALLOWED_PROVIDER_TRANSITIONS[from]?.includes(to) ?? false;
}

/**
 * States from which an emergency disable is valid. Mirrors the backend rule:
 * EmergencyDisable pulls an available/private_beta version back to
 * internal_validation.
 */
export const EMERGENCY_DISABLE_FROM_STATES: ReadonlySet<ProviderPublicationState> =
  new Set(["private_beta", "available"]);
