"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  OrderGroup,
  MergeOrdersRequest,
  SplitOrderRequest,
} from "@/types/api";

export function useOrderGroups(orderId: string) {
  return useQuery({
    queryKey: ["order-groups", orderId],
    queryFn: () =>
      apiClient<OrderGroup[]>(`/v1/orders/${orderId}/groups`),
    enabled: !!orderId,
  });
}

export function useMergeOrders() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: MergeOrdersRequest) =>
      apiClient<OrderGroup>("/v1/orders/merge", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["order-groups"] });
    },
  });
}

export function useSplitOrder(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: SplitOrderRequest) =>
      apiClient<OrderGroup>(`/v1/orders/${orderId}/split`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["orders", orderId] });
      queryClient.invalidateQueries({ queryKey: ["order-groups"] });
    },
  });
}
