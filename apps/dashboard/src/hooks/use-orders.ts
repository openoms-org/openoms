"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import { downloadBlob } from "@/lib/download";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Order,
  ListResponse,
  OrderListParams,
  CreateOrderRequest,
  UpdateOrderRequest,
  StatusTransitionRequest,
  AuditLogEntry,
  BulkStatusTransitionResponse,
} from "@/types/api";

const orderHooks = createCrudHooks<
  Order,
  CreateOrderRequest,
  UpdateOrderRequest,
  OrderListParams
>({
  resourceKey: "orders",
  basePath: "/v1/orders",
});

export const useOrders = orderHooks.useList;
export const useOrder = orderHooks.useGet;
export const useCreateOrder = orderHooks.useCreate;
export const useUpdateOrder = orderHooks.useUpdate;
export const useDeleteOrder = orderHooks.useDelete;

export function useTransitionOrderStatus(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: StatusTransitionRequest) =>
      apiClient<Order>(`/v1/orders/${id}/status`, { method: "POST", body: JSON.stringify(data) }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["orders", id] });
    },
  });
}

export function useBulkTransitionStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { order_ids: string[]; status: string; force?: boolean }) =>
      apiClient<BulkStatusTransitionResponse>("/v1/orders/bulk-status", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}

export function useOrderAudit(id: string) {
  return useQuery({
    queryKey: ["orders", id, "audit"],
    queryFn: () => apiClient<AuditLogEntry[]>(`/v1/orders/${id}/audit`),
    enabled: !!id,
  });
}

export function useDuplicateOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<Order>(`/v1/orders/${id}/duplicate`, { method: "POST" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}

export async function exportOrdersCSV(params: OrderListParams) {
  const searchParams = new URLSearchParams();
  if (params.status) searchParams.set("status", params.status);
  if (params.source) searchParams.set("source", params.source);
  if (params.search) searchParams.set("search", params.search);
  if (params.payment_status) searchParams.set("payment_status", params.payment_status);
  if (params.tag) searchParams.set("tag", params.tag);

  const response = await apiFetch(`/v1/orders/export?${searchParams}`);

  const blob = await response.blob();
  downloadBlob(blob, `orders-${new Date().toISOString().slice(0, 10)}.csv`);
}
