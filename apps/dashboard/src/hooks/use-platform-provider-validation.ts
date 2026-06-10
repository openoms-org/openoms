// NOTE: There are intentionally no mutations for recording probe results or
// completing validation runs. Run results and completion are backend/API
// driven (probes execute server-side after POST .../validate); the Validation
// Lab UI only starts runs and reads their outcome.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type {
  EmergencyDisableRequest,
  EnableTenantRequest,
  ProviderProbesResponse,
  ProviderPublicationEvent,
  ProviderPublicationEventsResponse,
  ProviderTenantEnable,
  ProviderValidationProbe,
  ProviderValidationRun,
  ProviderValidationRunsResponse,
  ProviderVersion,
  PublishRequest,
  StartValidationRunRequest,
} from "@/types/platform";

const BASE = "/v1/platform/providers";

function versionBase(providerId: string, versionId: string): string {
  return `${BASE}/${providerId}/versions/${versionId}`;
}

// Query keys are scoped by provider + version so a single run only invalidates
// the affected surface, not the whole studio.
const keys = {
  probes: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "probes"] as const,
  runs: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "validation-runs"] as const,
  run: (p: string, v: string, runId: string) =>
    ["platform", "providers", p, "versions", v, "validation-runs", runId] as const,
  publicationEvents: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "publication-events"] as const,
  version: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v] as const,
};

// ---- Probes ----

export function useProviderProbes(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderValidationProbe[]>({
    queryKey: keys.probes(providerId, versionId),
    queryFn: async () => {
      const data = await apiClient<ProviderProbesResponse>(
        `${versionBase(providerId, versionId)}/probes`,
      );
      return data.probes ?? [];
    },
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 30 * 1000,
  });
}

// ---- Validation runs ----

export function useProviderValidationRuns(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderValidationRun[]>({
    queryKey: keys.runs(providerId, versionId),
    queryFn: async () => {
      const data = await apiClient<ProviderValidationRunsResponse>(
        `${versionBase(providerId, versionId)}/validation-runs`,
      );
      return data.runs ?? [];
    },
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 15 * 1000,
  });
}

/** A single run including its immutable probe results. */
export function useProviderValidationRun(
  providerId: string,
  versionId: string,
  runId: string | null,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderValidationRun>({
    queryKey: keys.run(providerId, versionId, runId ?? ""),
    queryFn: () =>
      apiClient<ProviderValidationRun>(
        `${versionBase(providerId, versionId)}/validation-runs/${runId}`,
      ),
    enabled:
      !!token &&
      !!providerId &&
      !!versionId &&
      !!runId &&
      (options?.enabled ?? true),
    staleTime: 10 * 1000,
  });
}

export function useStartValidationRun(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: StartValidationRunRequest) =>
      apiClient<ProviderValidationRun>(
        `${versionBase(providerId, versionId)}/validate`,
        { method: "POST", body: JSON.stringify(data) },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.runs(providerId, versionId) });
    },
  });
}

// ---- Lifecycle / publication ----

export function useProviderPublicationEvents(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderPublicationEvent[]>({
    queryKey: keys.publicationEvents(providerId, versionId),
    queryFn: async () => {
      const data = await apiClient<ProviderPublicationEventsResponse>(
        `${versionBase(providerId, versionId)}/publication-events`,
      );
      return data.events ?? [];
    },
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 30 * 1000,
  });
}

/** Invalidate every surface that depends on a version's lifecycle state. */
function invalidateLifecycle(
  queryClient: ReturnType<typeof useQueryClient>,
  providerId: string,
  versionId: string,
) {
  queryClient.invalidateQueries({
    queryKey: keys.publicationEvents(providerId, versionId),
  });
  queryClient.invalidateQueries({ queryKey: keys.version(providerId, versionId) });
  // The version list (and read-only derivation) depends on publication_state.
  queryClient.invalidateQueries({
    queryKey: ["platform", "providers", providerId, "versions"],
  });
  queryClient.invalidateQueries({
    queryKey: ["platform", "providers", providerId],
  });
}

export function usePublishProviderVersion(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: PublishRequest) =>
      apiClient<ProviderVersion>(`${versionBase(providerId, versionId)}/publish`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => invalidateLifecycle(queryClient, providerId, versionId),
  });
}

export function useEmergencyDisableProviderVersion(
  providerId: string,
  versionId: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: EmergencyDisableRequest) =>
      apiClient<ProviderVersion>(`${versionBase(providerId, versionId)}/disable`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => invalidateLifecycle(queryClient, providerId, versionId),
  });
}

export function useEnableTenantForVersion(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: EnableTenantRequest) =>
      apiClient<ProviderTenantEnable>(
        `${versionBase(providerId, versionId)}/enable-tenant`,
        { method: "POST", body: JSON.stringify(data) },
      ),
    onSuccess: () => invalidateLifecycle(queryClient, providerId, versionId),
  });
}
