"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import { downloadBlob } from "@/lib/download";
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
  // apiClient<Blob> is wrong: apiClient always res.json()s the body, so it can
  // never yield a Blob. Use apiFetch (raw Response) + downloadBlob instead,
  // mirroring useDownloadOSSReportCSV in use-vat-oss.ts.
  return useMutation({
    mutationFn: async () => {
      const res = await apiFetch("/v1/settings/export");
      const blob = await res.blob();
      const stamp = new Date().toISOString().slice(0, 10);
      downloadBlob(blob, `openoms-settings-${stamp}.json`);
    },
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
