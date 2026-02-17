"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  ProductForecast,
  ReorderRecommendation,
  SeasonalityData,
  ProductVelocity,
  ForecastConfig,
} from "@/types/api";

export function useProductForecasts(days?: number) {
  const params = days ? `?days=${days}` : "";
  return useQuery({
    queryKey: ["forecast", "products", days],
    queryFn: () =>
      apiClient<ProductForecast[]>(`/v1/forecast/products${params}`),
  });
}

export function useProductForecast(productId: string, days?: number) {
  const params = days ? `?days=${days}` : "";
  return useQuery({
    queryKey: ["forecast", "product", productId, days],
    queryFn: () =>
      apiClient<ProductForecast>(`/v1/forecast/products/${productId}${params}`),
    enabled: !!productId,
  });
}

export function useReorderRecommendations() {
  return useQuery({
    queryKey: ["forecast", "reorder"],
    queryFn: () =>
      apiClient<ReorderRecommendation[]>("/v1/forecast/reorder"),
  });
}

export function useSeasonality(productId: string) {
  return useQuery({
    queryKey: ["forecast", "seasonality", productId],
    queryFn: () =>
      apiClient<SeasonalityData>(`/v1/forecast/seasonality/${productId}`),
    enabled: !!productId,
  });
}

export function useProductVelocity() {
  return useQuery({
    queryKey: ["forecast", "velocity"],
    queryFn: () => apiClient<ProductVelocity[]>("/v1/forecast/velocity"),
  });
}

export function useForecastConfig() {
  return useQuery({
    queryKey: ["forecast", "config"],
    queryFn: () => apiClient<ForecastConfig>("/v1/forecast/config"),
  });
}

export function useUpdateForecastConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: ForecastConfig) =>
      apiClient<ForecastConfig>("/v1/forecast/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(config),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["forecast"] });
    },
  });
}
