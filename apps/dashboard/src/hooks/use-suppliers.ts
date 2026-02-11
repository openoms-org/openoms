import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  Supplier,
  ListResponse,
  SupplierListParams,
  CreateSupplierRequest,
  UpdateSupplierRequest,
  SupplierProduct,
  SupplierProductListParams,
} from "@/types/api";

export function useSuppliers(params: SupplierListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit) query.set("limit", String(params.limit));
  if (params.offset) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["suppliers", params],
    queryFn: () =>
      apiClient<ListResponse<Supplier>>(`/v1/suppliers${qs ? `?${qs}` : ""}`),
  });
}

export function useSupplier(id: string) {
  return useQuery({
    queryKey: ["suppliers", id],
    queryFn: () => apiClient<Supplier>(`/v1/suppliers/${id}`),
    enabled: !!id,
  });
}

export function useCreateSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSupplierRequest) =>
      apiClient<Supplier>("/v1/suppliers", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
    },
  });
}

export function useUpdateSupplier(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateSupplierRequest) =>
      apiClient<Supplier>(`/v1/suppliers/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
      queryClient.invalidateQueries({ queryKey: ["suppliers", id] });
    },
  });
}

export function useDeleteSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/suppliers/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
    },
  });
}

export function useSyncSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<{ message: string }>(`/v1/suppliers/${id}/sync`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
    },
  });
}

export function useSupplierProducts(
  supplierId: string,
  params: SupplierProductListParams = {}
) {
  const query = new URLSearchParams();
  if (params.limit) query.set("limit", String(params.limit));
  if (params.offset) query.set("offset", String(params.offset));
  if (params.ean) query.set("ean", params.ean);
  if (params.linked !== undefined) query.set("linked", String(params.linked));

  const qs = query.toString();

  return useQuery({
    queryKey: ["supplier-products", supplierId, params],
    queryFn: () =>
      apiClient<ListResponse<SupplierProduct>>(
        `/v1/suppliers/${supplierId}/products${qs ? `?${qs}` : ""}`
      ),
    enabled: !!supplierId,
  });
}
