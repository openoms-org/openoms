import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type {
  PurchaseOrder,
  ListResponse,
  PurchaseOrderListParams,
  CreatePurchaseOrderRequest,
  UpdatePurchaseOrderRequest,
  ReceiveItemsRequest,
} from "@/types/api";

export function usePurchaseOrders(params: PurchaseOrderListParams = {}) {
  const qs = buildSearchParams(params).toString();

  return useQuery({
    queryKey: ["purchase-orders", params],
    queryFn: () =>
      apiClient<ListResponse<PurchaseOrder>>(
        `/v1/purchase-orders${qs ? `?${qs}` : ""}`
      ),
  });
}

export function usePurchaseOrder(id: string) {
  return useQuery({
    queryKey: ["purchase-orders", id],
    queryFn: () => apiClient<PurchaseOrder>(`/v1/purchase-orders/${id}`),
    enabled: !!id,
  });
}

export function useCreatePurchaseOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePurchaseOrderRequest) =>
      apiClient<PurchaseOrder>("/v1/purchase-orders", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["purchase-orders"] });
    },
  });
}

export function useUpdatePurchaseOrder(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdatePurchaseOrderRequest) =>
      apiClient<PurchaseOrder>(`/v1/purchase-orders/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["purchase-orders"] });
      queryClient.invalidateQueries({ queryKey: ["purchase-orders", id] });
    },
  });
}

export function useDeletePurchaseOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/purchase-orders/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["purchase-orders"] });
    },
  });
}

export function useSendPurchaseOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<PurchaseOrder>(`/v1/purchase-orders/${id}/send`, {
        method: "POST",
      }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["purchase-orders"] });
      queryClient.invalidateQueries({ queryKey: ["purchase-orders", id] });
    },
  });
}

export function useReceivePurchaseOrderItems(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ReceiveItemsRequest) =>
      apiClient<PurchaseOrder>(`/v1/purchase-orders/${id}/receive`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["purchase-orders"] });
      queryClient.invalidateQueries({ queryKey: ["purchase-orders", id] });
    },
  });
}

export function useCancelPurchaseOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<PurchaseOrder>(`/v1/purchase-orders/${id}/cancel`, {
        method: "POST",
      }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["purchase-orders"] });
      queryClient.invalidateQueries({ queryKey: ["purchase-orders", id] });
    },
  });
}
