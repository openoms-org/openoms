"use client";

import { useQuery } from "@tanstack/react-query";
import { API_URL } from "@/lib/api-client";

interface PublicConfigData {
  registration_mode: "open" | "invite" | "closed" | "disabled";
  license_enabled: boolean;
  billing_enabled: boolean;
  stripe_public_key?: string;
}

interface PublicConfig extends PublicConfigData {
  isLoading: boolean;
}

// Fail-closed fallback used while the public config is loading or when it cannot
// be fetched (retry: false). It must NEVER default to "open" registration — if
// the backend is unreachable we assume the most restrictive mode so an outage
// cannot expose open self-registration. Server-side enforcement is authoritative.
const defaultConfig: PublicConfigData = {
  registration_mode: "closed",
  license_enabled: false,
  billing_enabled: false,
};

export const publicConfigQueryKey = ["public-config"] as const;

async function fetchPublicConfig(): Promise<PublicConfigData> {
  const response = await fetch(`${API_URL}/v1/config/public`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load public config");
  }

  return response.json() as Promise<PublicConfigData>;
}

export function usePublicConfig() {
  const query = useQuery({
    queryKey: publicConfigQueryKey,
    queryFn: fetchPublicConfig,
    retry: false,
    staleTime: 60_000,
  });

  return {
    ...(query.data ?? defaultConfig),
    isLoading: query.isLoading,
  } satisfies PublicConfig;
}
