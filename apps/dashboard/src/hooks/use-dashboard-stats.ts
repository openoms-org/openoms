"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { DashboardStats } from "@/types/api";

export function useDashboardStats() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["dashboard-stats"],
    queryFn: () => apiClient<DashboardStats>("/v1/stats/dashboard"),
    enabled: isAuthenticated,
  });
}
