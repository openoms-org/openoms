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
