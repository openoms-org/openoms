import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type {
  PaymentSettlement,
  PaymentSettlementWithTransactions,
  ReconciliationSummary,
  AutoMatchResponse,
  ManualMatchRequest,
  CreateSettlementRequest,
  ListResponse,
  SettlementListParams,
} from "@/types/api";

export function useReconciliationSummary(dateFrom?: string, dateTo?: string) {
  const query = new URLSearchParams();
  if (dateFrom) query.set("date_from", dateFrom);
  if (dateTo) query.set("date_to", dateTo);
  const qs = query.toString();

  return useQuery({
    queryKey: ["reconciliation", "summary", dateFrom, dateTo],
    queryFn: () =>
      apiClient<ReconciliationSummary>(
        `/v1/reconciliation/summary${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useSettlements(params: SettlementListParams = {}) {
  const qs = buildSearchParams(params).toString();

  return useQuery({
    queryKey: ["reconciliation", "settlements", params],
    queryFn: () =>
      apiClient<ListResponse<PaymentSettlement>>(
        `/v1/reconciliation/settlements${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useSettlement(id: string) {
  return useQuery({
    queryKey: ["reconciliation", "settlements", id],
    queryFn: () =>
      apiClient<PaymentSettlementWithTransactions>(
        `/v1/reconciliation/settlements/${id}`
      ),
    enabled: !!id,
  });
}

export function useCreateSettlement() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateSettlementRequest) =>
      apiClient<PaymentSettlement>("/v1/reconciliation/settlements", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reconciliation"] });
    },
  });
}

export function useImportCSV() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      file,
      provider,
    }: {
      file: File;
      provider: string;
    }) => {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("provider", provider);

      const res = await apiFetch("/v1/reconciliation/import-csv", {
        method: "POST",
        body: formData,
      });
      return res.json() as Promise<PaymentSettlement>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reconciliation"] });
    },
  });
}

export function useAutoMatch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settlementId: string) =>
      apiClient<AutoMatchResponse>(
        `/v1/reconciliation/settlements/${settlementId}/auto-match`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reconciliation"] });
    },
  });
}

export function useManualMatch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      transactionId,
      data,
    }: {
      transactionId: string;
      data: ManualMatchRequest;
    }) =>
      apiClient<void>(
        `/v1/reconciliation/transactions/${transactionId}/match`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reconciliation"] });
    },
  });
}

export function useUnmatchTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (transactionId: string) =>
      apiClient<void>(
        `/v1/reconciliation/transactions/${transactionId}/unmatch`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reconciliation"] });
    },
  });
}
