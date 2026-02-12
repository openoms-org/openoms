import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  KSeFSettings,
  KSeFTestResult,
  KSeFBulkSendResult,
} from "@/types/api";

export function useKSeFSettings() {
  return useQuery({
    queryKey: ["settings", "ksef"],
    queryFn: () => apiClient<KSeFSettings>("/v1/settings/ksef"),
  });
}

export function useUpdateKSeFSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: KSeFSettings) =>
      apiClient<KSeFSettings>("/v1/settings/ksef", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "ksef"] });
    },
  });
}

export function useTestKSeFConnection() {
  return useMutation({
    mutationFn: () =>
      apiClient<KSeFTestResult>("/v1/settings/ksef/test", {
        method: "POST",
      }),
  });
}

export function useSendToKSeF() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (invoiceId: string) =>
      apiClient<{ message: string }>(`/v1/invoices/${invoiceId}/ksef/send`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
    },
  });
}

export function useCheckKSeFStatus() {
  return useMutation({
    mutationFn: (invoiceId: string) =>
      apiClient<{
        ksef_status: string;
        ksef_number?: string;
        ksef_sent_at?: string;
        ksef_response?: Record<string, unknown>;
      }>(`/v1/invoices/${invoiceId}/ksef/status`),
  });
}

export function useBulkSendToKSeF() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (invoiceIds: string[]) =>
      apiClient<KSeFBulkSendResult>("/v1/invoices/ksef/bulk-send", {
        method: "POST",
        body: JSON.stringify({ invoice_ids: invoiceIds }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
    },
  });
}
