"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { CompanySettings, OrderStatusConfig, CustomFieldsConfig, InventorySettings } from "@/types/api";

export function useCompanySettings() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["settings", "company"],
    queryFn: () => apiClient<CompanySettings>("/v1/settings/company"),
    enabled: isAuthenticated,
  });
}

export function useUpdateCompanySettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CompanySettings) =>
      apiClient<CompanySettings>("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "company"] });
    },
  });
}

export function useUpdateOrderStatuses() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: OrderStatusConfig) =>
      apiClient<OrderStatusConfig>("/v1/settings/order-statuses", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["order-statuses"] });
    },
  });
}

export function useUpdateCustomFields() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CustomFieldsConfig) =>
      apiClient<CustomFieldsConfig>("/v1/settings/custom-fields", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom-fields"] });
    },
  });
}

export function useInventorySettings() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["settings", "inventory"],
    queryFn: () => apiClient<InventorySettings>("/v1/settings/inventory"),
    enabled: isAuthenticated,
  });
}

export function useUpdateInventorySettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: InventorySettings) =>
      apiClient<InventorySettings>("/v1/settings/inventory", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "inventory"] });
    },
  });
}

export function useExportSettings() {
  return useMutation({
    mutationFn: () => apiClient<Blob>("/v1/settings/export"),
  });
}

export function useImportSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      apiClient("/v1/settings/import", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
    },
  });
}
