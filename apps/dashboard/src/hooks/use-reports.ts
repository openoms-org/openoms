"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { TopProduct, SourceRevenue, DailyOrderTrend } from "@/types/api";

export function useTopProducts(limit = 10) {
  return useQuery({
    queryKey: ["stats", "top-products", limit],
    queryFn: () =>
      apiClient<TopProduct[]>(`/v1/stats/products/top?limit=${limit}`),
  });
}

export function useRevenueBySource(days = 30) {
  return useQuery({
    queryKey: ["stats", "revenue-by-source", days],
    queryFn: () =>
      apiClient<SourceRevenue[]>(`/v1/stats/revenue/by-source?days=${days}`),
  });
}

export function useOrderTrends(days = 30) {
  return useQuery({
    queryKey: ["stats", "order-trends", days],
    queryFn: () =>
      apiClient<DailyOrderTrend[]>(`/v1/stats/trends?days=${days}`),
  });
}

export function usePaymentMethodStats() {
  return useQuery({
    queryKey: ["stats", "payment-methods"],
    queryFn: () =>
      apiClient<Record<string, number>>(`/v1/stats/payment-methods`),
  });
}
