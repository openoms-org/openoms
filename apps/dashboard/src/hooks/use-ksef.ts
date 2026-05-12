import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { KSeFBulkSendResult } from "@/types/api";

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
