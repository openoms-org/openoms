import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
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

export function useWarehouses(params: WarehouseListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.active !== undefined) query.set("active", String(params.active));
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["warehouses", params],
    queryFn: () =>
      apiClient<ListResponse<Warehouse>>(
        `/v1/warehouses${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useWarehouse(id: string) {
  return useQuery({
    queryKey: ["warehouses", id],
    queryFn: () => apiClient<Warehouse>(`/v1/warehouses/${id}`),
    enabled: !!id,
  });
}

export function useCreateWarehouse() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateWarehouseRequest) =>
      apiClient<Warehouse>("/v1/warehouses", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouses"] });
    },
  });
}

export function useUpdateWarehouse(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateWarehouseRequest) =>
      apiClient<Warehouse>(`/v1/warehouses/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouses"] });
      queryClient.invalidateQueries({ queryKey: ["warehouses", id] });
    },
  });
}

export function useDeleteWarehouse() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/warehouses/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouses"] });
    },
  });
}

export function useWarehouseStock(
  warehouseId: string,
  params: WarehouseStockListParams = {}
) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));

  const qs = query.toString();

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

export function useProductStock(productId: string) {
  return useQuery({
    queryKey: ["product-stock", productId],
    queryFn: () =>
      apiClient<WarehouseStock[]>(`/v1/products/${productId}/stock`),
    enabled: !!productId,
  });
}
