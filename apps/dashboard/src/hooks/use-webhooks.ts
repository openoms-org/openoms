"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { WebhookConfig, WebhookDelivery, WebhookDeliveryParams, ListResponse } from "@/types/api";

export function useWebhookConfig() {
  return useQuery({
    queryKey: ["webhook-config"],
    queryFn: () => apiClient<WebhookConfig>("/v1/settings/webhooks"),
    staleTime: 5 * 60 * 1000,
  });
}

export function useUpdateWebhookConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: WebhookConfig) =>
      apiClient<WebhookConfig>("/v1/settings/webhooks", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhook-config"] });
    },
  });
}

export function useWebhookDeliveries(params: WebhookDeliveryParams) {
  return useQuery({
    queryKey: ["webhook-deliveries", params],
    queryFn: () => {
      const query = new URLSearchParams();
      if (params.limit != null) query.set("limit", params.limit.toString());
      if (params.offset != null) query.set("offset", params.offset.toString());
      if (params.event_type) query.set("event_type", params.event_type);
      if (params.status) query.set("status", params.status);
      return apiClient<ListResponse<WebhookDelivery>>(`/v1/webhook-deliveries?${query.toString()}`);
    },
  });
}
