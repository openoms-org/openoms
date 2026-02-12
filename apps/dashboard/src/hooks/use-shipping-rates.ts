import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { GetRatesRequest, GetRatesResponse } from "@/types/api";

export function useShippingRates() {
  return useMutation({
    mutationFn: (data: GetRatesRequest) =>
      apiClient<GetRatesResponse>("/v1/shipping/rates", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}
