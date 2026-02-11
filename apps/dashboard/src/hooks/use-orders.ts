"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
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

export function useOrders(params: OrderListParams = {}) {
  const searchParams = new URLSearchParams();
  if (params.limit != null) searchParams.set("limit", String(params.limit));
  if (params.offset != null) searchParams.set("offset", String(params.offset));
  if (params.status) searchParams.set("status", params.status);
  if (params.source) searchParams.set("source", params.source);
  if (params.search) searchParams.set("search", params.search);
  if (params.payment_status) searchParams.set("payment_status", params.payment_status);
  if (params.tag) searchParams.set("tag", params.tag);
  if (params.sort_by) searchParams.set("sort_by", params.sort_by);
  if (params.sort_order) searchParams.set("sort_order", params.sort_order);

  const query = searchParams.toString();

  return useQuery({
    queryKey: ["orders", params],
    queryFn: () => apiClient<ListResponse<Order>>(`/v1/orders${query ? `?${query}` : ""}`),
  });
}

export function useOrder(id: string) {
  return useQuery({
    queryKey: ["orders", id],
    queryFn: () => apiClient<Order>(`/v1/orders/${id}`),
    enabled: !!id,
  });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateOrderRequest) =>
      apiClient<Order>("/v1/orders", { method: "POST", body: JSON.stringify(data) }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}

export function useUpdateOrder(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateOrderRequest) =>
      apiClient<Order>(`/v1/orders/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["orders", id] });
    },
  });
}

export function useDeleteOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/orders/${id}`, { method: "DELETE" }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["orders", id] });
    },
  });
}

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

export async function exportOrdersCSV(params: OrderListParams) {
  const searchParams = new URLSearchParams();
  if (params.status) searchParams.set("status", params.status);
  if (params.source) searchParams.set("source", params.source);
  if (params.search) searchParams.set("search", params.search);
  if (params.payment_status) searchParams.set("payment_status", params.payment_status);
  if (params.tag) searchParams.set("tag", params.tag);

  const response = await apiFetch(`/v1/orders/export?${searchParams}`);

  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `zamówienia-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
