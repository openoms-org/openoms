import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  PriceList,
  PriceListItem,
  ListResponse,
  PriceListListParams,
  CreatePriceListRequest,
  UpdatePriceListRequest,
  CreatePriceListItemRequest,
  PaginationParams,
} from "@/types/api";

export function usePriceLists(params: PriceListListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.active !== undefined) query.set("active", String(params.active));
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["price-lists", params],
    queryFn: () =>
      apiClient<ListResponse<PriceList>>(
        `/v1/price-lists${qs ? `?${qs}` : ""}`
      ),
  });
}

export function usePriceList(id: string) {
  return useQuery({
    queryKey: ["price-lists", id],
    queryFn: () => apiClient<PriceList>(`/v1/price-lists/${id}`),
    enabled: !!id,
  });
}

export function useCreatePriceList() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePriceListRequest) =>
      apiClient<PriceList>("/v1/price-lists", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["price-lists"] });
    },
  });
}

export function useUpdatePriceList(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdatePriceListRequest) =>
      apiClient<PriceList>(`/v1/price-lists/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["price-lists"] });
      queryClient.invalidateQueries({ queryKey: ["price-lists", id] });
    },
  });
}

export function useDeletePriceList() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/price-lists/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["price-lists"] });
    },
  });
}

export function usePriceListItems(priceListId: string, params: PaginationParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));

  const qs = query.toString();

  return useQuery({
    queryKey: ["price-list-items", priceListId, params],
    queryFn: () =>
      apiClient<ListResponse<PriceListItem>>(
        `/v1/price-lists/${priceListId}/items${qs ? `?${qs}` : ""}`
      ),
    enabled: !!priceListId,
  });
}

export function useCreatePriceListItem(priceListId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePriceListItemRequest) =>
      apiClient<PriceListItem>(`/v1/price-lists/${priceListId}/items`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["price-list-items", priceListId],
      });
    },
  });
}

export function useDeletePriceListItem(priceListId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (itemId: string) =>
      apiClient<void>(`/v1/price-lists/${priceListId}/items/${itemId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["price-list-items", priceListId],
      });
    },
  });
}
