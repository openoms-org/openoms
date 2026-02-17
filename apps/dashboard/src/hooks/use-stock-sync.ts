import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  StockSyncChannel,
  StockSyncEvent,
  StockSyncDashboard,
  StockAllocation,
  ListResponse,
  StockSyncChannelListParams,
  StockSyncEventListParams,
  CreateStockSyncChannelRequest,
  UpdateStockSyncChannelRequest,
} from "@/types/api";

// --- Channels ---

export function useStockSyncChannels(params: StockSyncChannelListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.enabled !== undefined) query.set("enabled", String(params.enabled));
  if (params.channel_type) query.set("channel_type", params.channel_type);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["stock-sync-channels", params],
    queryFn: () =>
      apiClient<ListResponse<StockSyncChannel>>(
        `/v1/stock-sync/channels${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useStockSyncChannel(id: string) {
  return useQuery({
    queryKey: ["stock-sync-channels", id],
    queryFn: () =>
      apiClient<StockSyncChannel>(`/v1/stock-sync/channels/${id}`),
    enabled: !!id,
  });
}

export function useCreateStockSyncChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateStockSyncChannelRequest) =>
      apiClient<StockSyncChannel>("/v1/stock-sync/channels", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stock-sync-channels"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-dashboard"] });
    },
  });
}

export function useUpdateStockSyncChannel(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateStockSyncChannelRequest) =>
      apiClient<void>(`/v1/stock-sync/channels/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stock-sync-channels"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-channels", id] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-dashboard"] });
    },
  });
}

export function useDeleteStockSyncChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/stock-sync/channels/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stock-sync-channels"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-dashboard"] });
    },
  });
}

// --- Push & Reconcile ---

export function usePushAllStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ channels_synced: number; message: string }>(
        "/v1/stock-sync/push",
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stock-sync-channels"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-events"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-dashboard"] });
    },
  });
}

export function usePushProductStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (productId: string) =>
      apiClient<{ message: string }>(`/v1/stock-sync/push/${productId}`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stock-sync-events"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-dashboard"] });
    },
  });
}

export function useReconcileProductStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (productId: string) =>
      apiClient<StockSyncEvent>(
        `/v1/stock-sync/reconcile/${productId}`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stock-sync-events"] });
      queryClient.invalidateQueries({ queryKey: ["stock-sync-dashboard"] });
    },
  });
}

// --- Events ---

export function useStockSyncEvents(params: StockSyncEventListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.product_id) query.set("product_id", params.product_id);
  if (params.trigger_type) query.set("trigger_type", params.trigger_type);

  const qs = query.toString();

  return useQuery({
    queryKey: ["stock-sync-events", params],
    queryFn: () =>
      apiClient<ListResponse<StockSyncEvent>>(
        `/v1/stock-sync/events${qs ? `?${qs}` : ""}`
      ),
  });
}

// --- Dashboard ---

export function useStockSyncDashboard() {
  return useQuery({
    queryKey: ["stock-sync-dashboard"],
    queryFn: () =>
      apiClient<StockSyncDashboard>("/v1/stock-sync/dashboard"),
    refetchInterval: 30000, // auto-refresh every 30s
  });
}

// --- Allocations ---

export function useStockAllocations(productId: string) {
  return useQuery({
    queryKey: ["stock-sync-allocations", productId],
    queryFn: () =>
      apiClient<StockAllocation[]>(
        `/v1/stock-sync/allocations/${productId}`
      ),
    enabled: !!productId,
  });
}
