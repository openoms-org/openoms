import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import { createCrudHooks } from "./create-crud-hooks";
import { useAllListItems } from "./use-all-list-items";
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

const priceListHooks = createCrudHooks<
  PriceList,
  CreatePriceListRequest,
  UpdatePriceListRequest,
  PriceListListParams
>({
  resourceKey: "price-lists",
  basePath: "/v1/price-lists",
});

export const usePriceLists = priceListHooks.useList;
export const usePriceList = priceListHooks.useGet;
export const useCreatePriceList = priceListHooks.useCreate;
export const useUpdatePriceList = priceListHooks.useUpdate;
export const useDeletePriceList = priceListHooks.useDelete;

export function useAllPriceLists(params: PriceListListParams = {}) {
  return useAllListItems<PriceList, PriceListListParams>(
    ["price-lists", "all", params],
    "/v1/price-lists",
    params,
  );
}

export function usePriceListItems(priceListId: string, params: PaginationParams = {}) {
  const sp = buildSearchParams(params as Record<string, string | number | boolean | null | undefined>);
  const qs = sp.toString();

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
