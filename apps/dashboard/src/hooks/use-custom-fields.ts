"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { CustomFieldsConfig } from "@/types/api";

export function useCustomFields() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["custom-fields"],
    queryFn: () => apiClient<CustomFieldsConfig>("/v1/custom-fields"),
    staleTime: 5 * 60 * 1000,
    enabled: isAuthenticated,
  });
}
