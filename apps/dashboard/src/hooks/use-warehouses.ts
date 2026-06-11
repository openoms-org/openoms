import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import { createCrudHooks } from "./create-crud-hooks";
import { useAllListItems } from "./use-all-list-items";
import type {
  Warehouse,
  WarehouseStock,
  ListResponse,
  WarehouseListParams,
  WarehouseStockListParams,
  CreateWarehouseRequest,
  UpdateWarehouseRequest,
  UpsertWarehouseStockRequest,
} from "@/types/api";

const warehouseHooks = createCrudHooks<
  Warehouse,
  CreateWarehouseRequest,
  UpdateWarehouseRequest,
  WarehouseListParams
>({
  resourceKey: "warehouses",
  basePath: "/v1/warehouses",
});

export const useWarehouses = warehouseHooks.useList;
export const useWarehouse = warehouseHooks.useGet;
export const useCreateWarehouse = warehouseHooks.useCreate;
export const useUpdateWarehouse = warehouseHooks.useUpdate;
export const useDeleteWarehouse = warehouseHooks.useDelete;

export function useAllWarehouses(params: WarehouseListParams = {}) {
  return useAllListItems<Warehouse, WarehouseListParams>(
    ["warehouses", "all", params],
    "/v1/warehouses",
    params,
  );
}

export function useWarehouseStock(
  warehouseId: string,
  params: WarehouseStockListParams = {}
) {
  const sp = buildSearchParams(params as Record<string, string | number | boolean | null | undefined>);
  const qs = sp.toString();

  return useQuery({
    queryKey: ["warehouse-stock", warehouseId, params],
    queryFn: () =>
      apiClient<ListResponse<WarehouseStock>>(
        `/v1/warehouses/${warehouseId}/stock${qs ? `?${qs}` : ""}`
      ),
    enabled: !!warehouseId,
  });
}

export function useUpsertWarehouseStock(warehouseId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpsertWarehouseStockRequest) =>
      apiClient<WarehouseStock>(`/v1/warehouses/${warehouseId}/stock`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["warehouse-stock", warehouseId],
      });
      queryClient.invalidateQueries({ queryKey: ["product-stock"] });
    },
  });
}
