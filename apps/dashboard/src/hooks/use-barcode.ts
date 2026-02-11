import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  BarcodeLookupResponse,
  PackOrderRequest,
  PackOrderResponse,
} from "@/types/api";

export function useBarcodeLookup(code: string) {
  return useQuery({
    queryKey: ["barcode", code],
    queryFn: () =>
      apiClient<BarcodeLookupResponse>(`/v1/barcode/${encodeURIComponent(code)}`),
    enabled: !!code && code.length > 0,
    retry: false,
    staleTime: 30_000,
  });
}

export function usePackOrder(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: PackOrderRequest) =>
      apiClient<PackOrderResponse>(`/v1/orders/${orderId}/pack`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["orders", orderId] });
    },
  });
}
