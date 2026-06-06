import { useQuery } from "@tanstack/react-query";
import { apiClient, ApiClientError } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type {
  PlatformMe,
  ProviderDefinition,
  ProviderDefinitionsResponse,
  ProviderVersion,
  ProviderVersionsResponse,
} from "@/types/platform";

const PLATFORM_PROVIDERS_BASE = "/v1/platform/providers";

/**
 * Identifies the current user as a platform administrator.
 * A non-platform-admin gets a 403, which we do NOT retry — callers use the
 * error/empty-permissions state to gate the rest of the studio UI.
 */
export function usePlatformMe() {
  const token = useAuthStore((s) => s.token);

  return useQuery<PlatformMe, ApiClientError>({
    queryKey: ["platform", "me"],
    queryFn: () => apiClient<PlatformMe>("/v1/platform/me"),
    enabled: !!token,
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}

/**
 * True when the platform "me" response grants any platform permission.
 * Frontend gating only — the backend remains authoritative.
 */
export function hasPlatformAccess(me: PlatformMe | undefined): boolean {
  return !!me && me.permissions.length > 0;
}

export function useProviderDefinitions(options?: { enabled?: boolean }) {
  const token = useAuthStore((s) => s.token);

  return useQuery<ProviderDefinition[]>({
    queryKey: ["platform", "providers"],
    queryFn: async () => {
      const data = await apiClient<ProviderDefinitionsResponse>(
        PLATFORM_PROVIDERS_BASE,
      );
      return data.definitions ?? [];
    },
    enabled: !!token && (options?.enabled ?? true),
    staleTime: 60 * 1000,
  });
}

export function useProviderDefinition(
  id: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);

  return useQuery<ProviderDefinition>({
    queryKey: ["platform", "providers", id],
    queryFn: () =>
      apiClient<ProviderDefinition>(`${PLATFORM_PROVIDERS_BASE}/${id}`),
    enabled: !!token && !!id && (options?.enabled ?? true),
    staleTime: 60 * 1000,
  });
}

export function useProviderVersions(
  id: string,
  options?: { enabled?: boolean },
) {
  const token = useAuthStore((s) => s.token);

  return useQuery<ProviderVersion[]>({
    queryKey: ["platform", "providers", id, "versions"],
    queryFn: async () => {
      const data = await apiClient<ProviderVersionsResponse>(
        `${PLATFORM_PROVIDERS_BASE}/${id}/versions`,
      );
      return data.versions ?? [];
    },
    enabled: !!token && !!id && (options?.enabled ?? true),
    staleTime: 60 * 1000,
  });
}
