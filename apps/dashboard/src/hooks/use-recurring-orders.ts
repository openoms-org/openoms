import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  RecurringOrder,
  ListResponse,
  RecurringOrderListParams,
  CreateRecurringOrderRequest,
  UpdateRecurringOrderRequest,
} from "@/types/api";

export function useRecurringOrders(params: RecurringOrderListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.customer_id) query.set("customer_id", params.customer_id);
  if (params.next_date_before) query.set("next_date_before", params.next_date_before);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["recurring-orders", params],
    queryFn: () =>
      apiClient<ListResponse<RecurringOrder>>(
        `/v1/recurring-orders${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useRecurringOrder(id: string) {
  return useQuery({
    queryKey: ["recurring-orders", id],
    queryFn: () => apiClient<RecurringOrder>(`/v1/recurring-orders/${id}`),
    enabled: !!id,
  });
}

export function useCreateRecurringOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateRecurringOrderRequest) =>
      apiClient<RecurringOrder>("/v1/recurring-orders", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recurring-orders"] });
    },
  });
}

export function useUpdateRecurringOrder(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateRecurringOrderRequest) =>
      apiClient<RecurringOrder>(`/v1/recurring-orders/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recurring-orders"] });
      queryClient.invalidateQueries({ queryKey: ["recurring-orders", id] });
    },
  });
}

export function useDeleteRecurringOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/recurring-orders/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recurring-orders"] });
    },
  });
}

export function usePauseRecurringOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<{ status: string }>(`/v1/recurring-orders/${id}/pause`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recurring-orders"] });
    },
  });
}

export function useResumeRecurringOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<{ status: string }>(`/v1/recurring-orders/${id}/resume`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recurring-orders"] });
    },
  });
}

export function useCancelRecurringOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<{ status: string }>(`/v1/recurring-orders/${id}/cancel`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recurring-orders"] });
    },
  });
}
