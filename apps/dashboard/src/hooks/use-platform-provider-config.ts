import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type {
  ProviderCapabilitiesResponse,
  ProviderCapabilitiesUpdateRequest,
  ProviderCapability,
  ProviderCreateGapRequest,
  ProviderFieldSchema,
  ProviderGapsResponse,
  ProviderIntegrationGap,
  ProviderSchemaUpdateRequest,
  ProviderStatusMapping,
  ProviderStatusMappingsResponse,
  ProviderStatusMappingsUpdateRequest,
  ProviderUpdateGapRequest,
} from "@/types/platform";

const BASE = "/v1/platform/providers";

function versionBase(providerId: string, versionId: string): string {
  return `${BASE}/${providerId}/versions/${versionId}`;
}

// Query keys are scoped by provider + version so a config edit only invalidates
// the affected surface, not the whole studio.
const keys = {
  schema: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "schema"] as const,
  capabilities: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "capabilities"] as const,
  statusMappings: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "status-mappings"] as const,
  gaps: (p: string, v: string) =>
    ["platform", "providers", p, "versions", v, "gaps"] as const,
};

// ---- Field schema ----

export function useProviderSchema(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderFieldSchema>({
    queryKey: keys.schema(providerId, versionId),
    queryFn: () =>
      apiClient<ProviderFieldSchema>(`${versionBase(providerId, versionId)}/schema`),
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 30 * 1000,
  });
}

export function useUpdateProviderSchema(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ProviderSchemaUpdateRequest) =>
      apiClient<ProviderFieldSchema>(`${versionBase(providerId, versionId)}/schema`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.schema(providerId, versionId) });
    },
  });
}

// ---- Capabilities ----

export function useProviderCapabilities(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderCapability[]>({
    queryKey: keys.capabilities(providerId, versionId),
    queryFn: async () => {
      const data = await apiClient<ProviderCapabilitiesResponse>(
        `${versionBase(providerId, versionId)}/capabilities`,
      );
      return data.capabilities ?? [];
    },
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 30 * 1000,
  });
}

export function useUpdateProviderCapabilities(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ProviderCapabilitiesUpdateRequest) => {
      const res = await apiClient<ProviderCapabilitiesResponse>(
        `${versionBase(providerId, versionId)}/capabilities`,
        { method: "PATCH", body: JSON.stringify(data) },
      );
      return res.capabilities ?? [];
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: keys.capabilities(providerId, versionId),
      });
      // Capability edits can open/close gaps server-side.
      queryClient.invalidateQueries({ queryKey: keys.gaps(providerId, versionId) });
    },
  });
}

// ---- Status mappings ----

export function useProviderStatusMappings(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderStatusMapping[]>({
    queryKey: keys.statusMappings(providerId, versionId),
    queryFn: async () => {
      const data = await apiClient<ProviderStatusMappingsResponse>(
        `${versionBase(providerId, versionId)}/status-mappings`,
      );
      return data.status_mappings ?? [];
    },
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 30 * 1000,
  });
}

export function useUpdateProviderStatusMappings(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ProviderStatusMappingsUpdateRequest) => {
      const res = await apiClient<ProviderStatusMappingsResponse>(
        `${versionBase(providerId, versionId)}/status-mappings`,
        { method: "PATCH", body: JSON.stringify(data) },
      );
      return res.status_mappings ?? [];
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: keys.statusMappings(providerId, versionId),
      });
      queryClient.invalidateQueries({ queryKey: keys.gaps(providerId, versionId) });
    },
  });
}

// ---- Integration gaps ----

export function useProviderGaps(
  providerId: string,
  versionId: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);
  return useQuery<ProviderIntegrationGap[]>({
    queryKey: keys.gaps(providerId, versionId),
    queryFn: async () => {
      const data = await apiClient<ProviderGapsResponse>(
        `${versionBase(providerId, versionId)}/gaps`,
      );
      return data.gaps ?? [];
    },
    enabled: !!token && !!providerId && !!versionId && (options?.enabled ?? true),
    staleTime: 30 * 1000,
  });
}

export function useCreateProviderGap(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ProviderCreateGapRequest) =>
      apiClient<ProviderIntegrationGap>(`${versionBase(providerId, versionId)}/gaps`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.gaps(providerId, versionId) });
    },
  });
}

export function useUpdateProviderGap(providerId: string, versionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ gapId, ...data }: ProviderUpdateGapRequest & { gapId: string }) =>
      apiClient<ProviderIntegrationGap>(
        `${versionBase(providerId, versionId)}/gaps/${gapId}`,
        { method: "PATCH", body: JSON.stringify(data) },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.gaps(providerId, versionId) });
    },
  });
}
