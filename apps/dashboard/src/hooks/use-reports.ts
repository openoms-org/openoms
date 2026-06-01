"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { TopProduct, SourceRevenue, DailyOrderTrend } from "@/types/api";

export function useTopProducts(limit = 10) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["stats", "top-products", limit],
    queryFn: () =>
      apiClient<TopProduct[]>(`/v1/stats/products/top?limit=${limit}`),
    enabled: isAuthenticated,
  });
}

export function useRevenueBySource(days = 30) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["stats", "revenue-by-source", days],
    queryFn: () =>
      apiClient<SourceRevenue[]>(`/v1/stats/revenue/by-source?days=${days}`),
    enabled: isAuthenticated,
  });
}

export function useOrderTrends(days = 30) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["stats", "order-trends", days],
    queryFn: () =>
      apiClient<DailyOrderTrend[]>(`/v1/stats/trends?days=${days}`),
    enabled: isAuthenticated,
  });
}

export function usePaymentMethodStats() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["stats", "payment-methods"],
    queryFn: () =>
      apiClient<Record<string, number>>(`/v1/stats/payment-methods`),
    enabled: isAuthenticated,
  });
}
