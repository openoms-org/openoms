"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type {
  ListingSyncConfig,
  ListingSyncLog,
  CreateListingSyncConfigRequest,
  UpdateListingSyncConfigRequest,
  SyncResult,
  ListResponse,
  ListingSyncConfigListParams,
  ListingSyncLogListParams,
} from "@/types/api";

function buildParams(params: Record<string, unknown>): string {
  const search = buildSearchParams(params);
  const str = search.toString();
  return str ? `?${str}` : "";
}

export function useListingSyncConfigs(params?: ListingSyncConfigListParams) {
  return useQuery({
    queryKey: ["listing-sync-configs", params],
    queryFn: () =>
      apiClient<ListResponse<ListingSyncConfig>>(
        `/v1/listing-sync/configs${buildParams({
          limit: params?.limit ?? 20,
          offset: params?.offset ?? 0,
          integration_id: params?.integration_id,
          status: params?.status,
        })}`
      ),
  });
}

export function useListingSyncConfig(id: string) {
  return useQuery({
    queryKey: ["listing-sync-configs", id],
    queryFn: () =>
      apiClient<ListingSyncConfig>(`/v1/listing-sync/configs/${id}`),
    enabled: !!id,
  });
}

export function useCreateListingSyncConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateListingSyncConfigRequest) =>
      apiClient<ListingSyncConfig>("/v1/listing-sync/configs", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listing-sync-configs"] });
    },
  });
}

export function useUpdateListingSyncConfig(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateListingSyncConfigRequest) =>
      apiClient<ListingSyncConfig>(`/v1/listing-sync/configs/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listing-sync-configs"] });
      queryClient.invalidateQueries({
        queryKey: ["listing-sync-configs", id],
      });
    },
  });
}

export function useDeleteListingSyncConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/listing-sync/configs/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listing-sync-configs"] });
    },
  });
}

export function useTriggerSync(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<SyncResult>(`/v1/listing-sync/configs/${id}/sync`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listing-sync-configs"] });
      queryClient.invalidateQueries({
        queryKey: ["listing-sync-configs", id],
      });
      queryClient.invalidateQueries({
        queryKey: ["listing-sync-logs", id],
      });
    },
  });
}

export function useTriggerSyncPrices(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<SyncResult>(`/v1/listing-sync/configs/${id}/sync-prices`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listing-sync-configs"] });
      queryClient.invalidateQueries({
        queryKey: ["listing-sync-logs", id],
      });
    },
  });
}

export function useTriggerSyncStock(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<SyncResult>(`/v1/listing-sync/configs/${id}/sync-stock`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listing-sync-configs"] });
      queryClient.invalidateQueries({
        queryKey: ["listing-sync-logs", id],
      });
    },
  });
}

export function useListingSyncLogs(
  configId: string,
  params?: ListingSyncLogListParams
) {
  return useQuery({
    queryKey: ["listing-sync-logs", configId, params],
    queryFn: () =>
      apiClient<ListResponse<ListingSyncLog>>(
        `/v1/listing-sync/configs/${configId}/logs${buildParams({
          limit: params?.limit ?? 20,
          offset: params?.offset ?? 0,
          direction: params?.direction,
          entity_type: params?.entity_type,
          status: params?.status,
        })}`
      ),
    enabled: !!configId,
  });
}
