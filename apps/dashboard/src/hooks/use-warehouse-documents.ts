import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  WarehouseDocument,
  WarehouseDocumentListParams,
  CreateWarehouseDocumentRequest,
  UpdateWarehouseDocumentRequest,
} from "@/types/api";

const warehouseDocumentHooks = createCrudHooks<
  WarehouseDocument,
  CreateWarehouseDocumentRequest,
  UpdateWarehouseDocumentRequest,
  WarehouseDocumentListParams
>({
  resourceKey: "warehouse-documents",
  basePath: "/v1/warehouse-documents",
});

export const useWarehouseDocuments = warehouseDocumentHooks.useList;
export const useWarehouseDocument = warehouseDocumentHooks.useGet;
export const useCreateWarehouseDocument = warehouseDocumentHooks.useCreate;
export const useUpdateWarehouseDocument = warehouseDocumentHooks.useUpdate;
export const useDeleteWarehouseDocument = warehouseDocumentHooks.useDelete;

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
