import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  ExchangeRate,
  ListResponse,
  ExchangeRateListParams,
  CreateExchangeRateRequest,
  UpdateExchangeRateRequest,
  ConvertAmountRequest,
  ConvertAmountResponse,
  FetchNBPResponse,
} from "@/types/api";

export function useExchangeRates(params: ExchangeRateListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.base_currency) query.set("base_currency", params.base_currency);
  if (params.target_currency)
    query.set("target_currency", params.target_currency);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["exchange-rates", params],
    queryFn: () =>
      apiClient<ListResponse<ExchangeRate>>(
        `/v1/exchange-rates${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useExchangeRate(id: string) {
  return useQuery({
    queryKey: ["exchange-rates", id],
    queryFn: () => apiClient<ExchangeRate>(`/v1/exchange-rates/${id}`),
    enabled: !!id,
  });
}

export function useCreateExchangeRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateExchangeRateRequest) =>
      apiClient<ExchangeRate>("/v1/exchange-rates", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exchange-rates"] });
    },
  });
}

export function useUpdateExchangeRate(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateExchangeRateRequest) =>
      apiClient<ExchangeRate>(`/v1/exchange-rates/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exchange-rates"] });
      queryClient.invalidateQueries({ queryKey: ["exchange-rates", id] });
    },
  });
}

export function useDeleteExchangeRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/exchange-rates/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exchange-rates"] });
    },
  });
}

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
