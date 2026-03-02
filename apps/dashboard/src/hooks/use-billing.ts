import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { SubscriptionStatus } from "@/types/api";

export function useSubscription() {
  return useQuery<SubscriptionStatus>({
    queryKey: ["billing", "subscription"],
    queryFn: () => apiClient<SubscriptionStatus>("/v1/billing/subscription"),
    staleTime: 5 * 60 * 1000,
  });
}
