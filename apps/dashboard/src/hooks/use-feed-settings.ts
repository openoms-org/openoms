"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { ProductFeedConfig } from "@/types/api";

export function useFeedSettings() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["settings", "feeds"],
    queryFn: () => apiClient<ProductFeedConfig>("/v1/settings/feeds"),
    enabled: isAuthenticated,
  });
}

export function useUpdateFeedSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ProductFeedConfig) =>
      apiClient<ProductFeedConfig>("/v1/settings/feeds", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "feeds"] });
    },
  });
}

export function useRegenerateFeedToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (feedType: "ceneo" | "google") =>
      apiClient<ProductFeedConfig>("/v1/settings/feeds/regenerate-token", {
        method: "POST",
        body: JSON.stringify({ feed_type: feedType }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "feeds"] });
    },
  });
}
