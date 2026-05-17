"use client";

import { useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { PROVIDER_CATEGORIES } from "@/lib/constants";
import { integrationQueryKeys } from "./integration-query-keys";
import type {
  Integration,
  CreateIntegrationRequest,
  UpdateIntegrationRequest,
} from "@/types/api";

export function useIntegrations() {
  return useQuery({
    queryKey: integrationQueryKeys.all,
    queryFn: () => apiClient<Integration[]>("/v1/integrations"),
  });
}

export function useIntegration(id: string) {
  return useQuery({
    queryKey: integrationQueryKeys.detail(id),
    queryFn: () => apiClient<Integration>(`/v1/integrations/${id}`),
    enabled: !!id,
  });
}

export function useCreateIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateIntegrationRequest) =>
      apiClient<Integration>("/v1/integrations", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: integrationQueryKeys.all });
    },
  });
}

export function useUpdateIntegration(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateIntegrationRequest) =>
      apiClient<Integration>(`/v1/integrations/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: integrationQueryKeys.all });
      queryClient.invalidateQueries({ queryKey: integrationQueryKeys.detail(id) });
    },
  });
}

export function useDeleteIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/integrations/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: integrationQueryKeys.all });
    },
  });
}

export function useIntegrationsByCategory() {
  const query = useIntegrations();
  const categorized = useMemo(() => {
    const all = query.data ?? [];
    return {
      carriers: all.filter((i) =>
        PROVIDER_CATEGORIES.carrier.providers.includes(i.provider)
      ),
      marketplaces: all.filter((i) =>
        PROVIDER_CATEGORIES.marketplace.providers.includes(i.provider)
      ),
      invoicing: all.filter((i) =>
        PROVIDER_CATEGORIES.invoicing.providers.includes(i.provider)
      ),
      suppliers: all.filter((i) =>
        PROVIDER_CATEGORIES.supplier.providers.includes(i.provider)
      ),
    };
  }, [query.data]);
  return { ...query, ...categorized };
}
