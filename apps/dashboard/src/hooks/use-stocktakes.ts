import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Stocktake,
  StocktakeItem,
  ListResponse,
  StocktakeListParams,
  StocktakeItemListParams,
  CreateStocktakeRequest,
  UpdateStocktakeItemRequest,
} from "@/types/api";

const stocktakeHooks = createCrudHooks<
  Stocktake,
  CreateStocktakeRequest,
  Partial<Stocktake>,
  StocktakeListParams
>({
  resourceKey: "stocktakes",
  basePath: "/v1/stocktakes",
});

export const useStocktakes = stocktakeHooks.useList;
export const useStocktake = stocktakeHooks.useGet;
export const useCreateStocktake = stocktakeHooks.useCreate;
export const useDeleteStocktake = stocktakeHooks.useDelete;

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
