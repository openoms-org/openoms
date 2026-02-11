import { useQuery, useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  MarketingSyncResponse,
  MarketingStatusResponse,
  CreateCampaignRequest,
  CreateCampaignResponse,
} from "@/types/api";

export function useMarketingStatus() {
  return useQuery({
    queryKey: ["marketing", "status"],
    queryFn: () => apiClient<MarketingStatusResponse>("/v1/marketing/status"),
  });
}

export function useSyncCustomers() {
  return useMutation({
    mutationFn: () =>
      apiClient<MarketingSyncResponse>("/v1/marketing/sync", {
        method: "POST",
      }),
  });
}

export function useCreateCampaign() {
  return useMutation({
    mutationFn: (data: CreateCampaignRequest) =>
      apiClient<CreateCampaignResponse>("/v1/marketing/campaigns", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}
