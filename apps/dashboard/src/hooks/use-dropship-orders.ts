import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type {
  DropshipOrder,
  ListResponse,
  DropshipOrderListParams,
  CreateDropshipOrderRequest,
  UpdateDropshipStatusRequest,
} from "@/types/api";

export function useDropshipOrders(params: DropshipOrderListParams = {}) {
  const qs = buildSearchParams(params).toString();

  return useQuery({
    queryKey: ["dropship-orders", params],
    queryFn: () =>
      apiClient<ListResponse<DropshipOrder>>(
        `/v1/dropship-orders${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useDropshipOrder(id: string) {
  return useQuery({
    queryKey: ["dropship-orders", id],
    queryFn: () => apiClient<DropshipOrder>(`/v1/dropship-orders/${id}`),
    enabled: !!id,
  });
}

export function useOrderDropshipOrders(
  orderId: string,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: ["dropship-orders", "by-order", orderId],
    queryFn: () =>
      apiClient<DropshipOrder[]>(`/v1/orders/${orderId}/dropship-orders`),
    enabled: !!orderId && (options.enabled ?? true),
  });
}

export function useCreateDropshipOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateDropshipOrderRequest) =>
      apiClient<DropshipOrder>("/v1/dropship-orders", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dropship-orders"] });
    },
  });
}

export function useAutoRouteDropship() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (orderId: string) =>
      apiClient<DropshipOrder[]>(`/v1/orders/${orderId}/dropship`, {
        method: "POST",
      }),
    onSuccess: (_data, orderId) => {
      queryClient.invalidateQueries({ queryKey: ["dropship-orders"] });
      queryClient.invalidateQueries({
        queryKey: ["dropship-orders", "by-order", orderId],
      });
    },
  });
}

export function useUpdateDropshipStatus(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateDropshipStatusRequest) =>
      apiClient<DropshipOrder>(`/v1/dropship-orders/${id}/status`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dropship-orders"] });
      queryClient.invalidateQueries({ queryKey: ["dropship-orders", id] });
    },
  });
}

export function useCancelDropshipOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<DropshipOrder>(`/v1/dropship-orders/${id}/cancel`, {
        method: "POST",
      }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["dropship-orders"] });
      queryClient.invalidateQueries({ queryKey: ["dropship-orders", id] });
    },
  });
}
