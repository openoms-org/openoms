import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  Stocktake,
  StocktakeItem,
  ListResponse,
  StocktakeListParams,
  StocktakeItemListParams,
  CreateStocktakeRequest,
  UpdateStocktakeItemRequest,
} from "@/types/api";

export function useStocktakes(params: StocktakeListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.warehouse_id) query.set("warehouse_id", params.warehouse_id);
  if (params.status) query.set("status", params.status);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["stocktakes", params],
    queryFn: () =>
      apiClient<ListResponse<Stocktake>>(
        `/v1/stocktakes${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useStocktake(id: string) {
  return useQuery({
    queryKey: ["stocktakes", id],
    queryFn: () => apiClient<Stocktake>(`/v1/stocktakes/${id}`),
    enabled: !!id,
  });
}

export function useCreateStocktake() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateStocktakeRequest) =>
      apiClient<Stocktake>("/v1/stocktakes", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stocktakes"] });
    },
  });
}

export function useDeleteStocktake() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/stocktakes/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stocktakes"] });
    },
  });
}

export function useStartStocktake() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<Stocktake>(`/v1/stocktakes/${id}/start`, { method: "POST" }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["stocktakes"] });
      queryClient.invalidateQueries({ queryKey: ["stocktakes", id] });
    },
  });
}

export function useCompleteStocktake() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<Stocktake>(`/v1/stocktakes/${id}/complete`, {
        method: "POST",
      }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["stocktakes"] });
      queryClient.invalidateQueries({ queryKey: ["stocktakes", id] });
      queryClient.invalidateQueries({ queryKey: ["stocktake-items", id] });
    },
  });
}

export function useCancelStocktake() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<Stocktake>(`/v1/stocktakes/${id}/cancel`, { method: "POST" }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["stocktakes"] });
      queryClient.invalidateQueries({ queryKey: ["stocktakes", id] });
    },
  });
}

export function useStocktakeItems(
  stocktakeId: string,
  params: StocktakeItemListParams = {}
) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.filter) query.set("filter", params.filter);

  const qs = query.toString();

  return useQuery({
    queryKey: ["stocktake-items", stocktakeId, params],
    queryFn: () =>
      apiClient<ListResponse<StocktakeItem>>(
        `/v1/stocktakes/${stocktakeId}/items${qs ? `?${qs}` : ""}`
      ),
    enabled: !!stocktakeId,
  });
}

export function useRecordCount(stocktakeId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      itemId,
      data,
    }: {
      itemId: string;
      data: UpdateStocktakeItemRequest;
    }) =>
      apiClient<StocktakeItem>(
        `/v1/stocktakes/${stocktakeId}/items/${itemId}/count`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["stocktake-items", stocktakeId],
      });
      queryClient.invalidateQueries({
        queryKey: ["stocktakes", stocktakeId],
      });
    },
  });
}
