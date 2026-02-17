import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import type {
  PaymentSettlement,
  PaymentSettlementWithTransactions,
  PaymentTransaction,
  ReconciliationSummary,
  AutoMatchResponse,
  ManualMatchRequest,
  CreateSettlementRequest,
  ListResponse,
  SettlementListParams,
  TransactionListParams,
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
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.provider) query.set("provider", params.provider);
  if (params.status) query.set("status", params.status);
  if (params.date_from) query.set("date_from", params.date_from);
  if (params.date_to) query.set("date_to", params.date_to);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);
  const qs = query.toString();

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

export function useTransactions(params: TransactionListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.settlement_id) query.set("settlement_id", params.settlement_id);
  if (params.match_status) query.set("match_status", params.match_status);
  if (params.provider) query.set("provider", params.provider);
  if (params.transaction_type)
    query.set("transaction_type", params.transaction_type);
  if (params.date_from) query.set("date_from", params.date_from);
  if (params.date_to) query.set("date_to", params.date_to);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);
  const qs = query.toString();

  return useQuery({
    queryKey: ["reconciliation", "transactions", params],
    queryFn: () =>
      apiClient<ListResponse<PaymentTransaction>>(
        `/v1/reconciliation/transactions${qs ? `?${qs}` : ""}`
      ),
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
