import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  ExchangeRate,
  ExchangeRateListParams,
  CreateExchangeRateRequest,
  UpdateExchangeRateRequest,
  ConvertAmountRequest,
  ConvertAmountResponse,
  FetchNBPResponse,
} from "@/types/api";

const exchangeRateHooks = createCrudHooks<
  ExchangeRate,
  CreateExchangeRateRequest,
  UpdateExchangeRateRequest,
  ExchangeRateListParams
>({
  resourceKey: "exchange-rates",
  basePath: "/v1/exchange-rates",
});

export const useExchangeRates = exchangeRateHooks.useList;
export const useExchangeRate = exchangeRateHooks.useGet;
export const useCreateExchangeRate = exchangeRateHooks.useCreate;
export const useUpdateExchangeRate = exchangeRateHooks.useUpdate;
export const useDeleteExchangeRate = exchangeRateHooks.useDelete;

export function useFetchNBPRates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<FetchNBPResponse>("/v1/exchange-rates/fetch", {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exchange-rates"] });
    },
  });
}

export function useConvertAmount() {
  return useMutation({
    mutationFn: (data: ConvertAmountRequest) =>
      apiClient<ConvertAmountResponse>("/v1/exchange-rates/convert", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}
