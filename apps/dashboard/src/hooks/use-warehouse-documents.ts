import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  WarehouseDocument,
  ListResponse,
  WarehouseDocumentListParams,
  CreateWarehouseDocumentRequest,
  UpdateWarehouseDocumentRequest,
} from "@/types/api";

export function useWarehouseDocuments(params: WarehouseDocumentListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.document_type) query.set("document_type", params.document_type);
  if (params.status) query.set("status", params.status);
  if (params.warehouse_id) query.set("warehouse_id", params.warehouse_id);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["warehouse-documents", params],
    queryFn: () =>
      apiClient<ListResponse<WarehouseDocument>>(
        `/v1/warehouse-documents${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useWarehouseDocument(id: string) {
  return useQuery({
    queryKey: ["warehouse-documents", id],
    queryFn: () =>
      apiClient<WarehouseDocument>(`/v1/warehouse-documents/${id}`),
    enabled: !!id,
  });
}

export function useCreateWarehouseDocument() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateWarehouseDocumentRequest) =>
      apiClient<WarehouseDocument>("/v1/warehouse-documents", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouse-documents"] });
    },
  });
}

export function useUpdateWarehouseDocument(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateWarehouseDocumentRequest) =>
      apiClient<WarehouseDocument>(`/v1/warehouse-documents/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouse-documents"] });
      queryClient.invalidateQueries({ queryKey: ["warehouse-documents", id] });
    },
  });
}

export function useDeleteWarehouseDocument() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/warehouse-documents/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouse-documents"] });
    },
  });
}

export function useConfirmWarehouseDocument() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<WarehouseDocument>(`/v1/warehouse-documents/${id}/confirm`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouse-documents"] });
      queryClient.invalidateQueries({ queryKey: ["warehouse-stock"] });
      queryClient.invalidateQueries({ queryKey: ["product-stock"] });
    },
  });
}

export function useCancelWarehouseDocument() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<WarehouseDocument>(`/v1/warehouse-documents/${id}/cancel`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["warehouse-documents"] });
    },
  });
}
