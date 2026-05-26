import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import { usePublicConfig } from "@/hooks/use-public-config";
import type { SubscriptionStatus } from "@/types/api";

export function useSubscription() {
  const token = useAuthStore((s) => s.token);
  const config = usePublicConfig();

  return useQuery<SubscriptionStatus>({
    queryKey: ["billing", "subscription"],
    queryFn: () => apiClient<SubscriptionStatus>("/v1/billing/subscription"),
    staleTime: 5 * 60 * 1000,
    enabled: !!token && !config.isLoading && config.billing_enabled,
  });
}
