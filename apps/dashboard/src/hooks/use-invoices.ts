import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Invoice,
  InvoiceListParams,
  CreateInvoiceRequest,
  InvoicingSettings,
} from "@/types/api";

const invoiceHooks = createCrudHooks<
  Invoice,
  CreateInvoiceRequest,
  Partial<Invoice>,
  InvoiceListParams
>({
  resourceKey: "invoices",
  basePath: "/v1/invoices",
});

export const useInvoices = invoiceHooks.useList;
export const useInvoice = invoiceHooks.useGet;
export const useCreateInvoice = invoiceHooks.useCreate;
// NOTE: no useUpdateInvoice — there is no PATCH/PUT /v1/invoices/{id} route.
// Polish invoices are legally immutable; corrections are separate documents.

export function useCancelInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/invoices/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
    },
  });
}

export function useInvoicingSettings() {
  return useQuery({
    queryKey: ["settings", "invoicing"],
    queryFn: () => apiClient<InvoicingSettings>("/v1/settings/invoicing"),
  });
}

export function useUpdateInvoicingSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: InvoicingSettings) =>
      apiClient<InvoicingSettings>("/v1/settings/invoicing", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "invoicing"] });
      queryClient.invalidateQueries({ queryKey: ["settings", "accounting"] });
    },
  });
}

// === Accounting Settings (wFirma, inFakt, Fakturownia connection management) ===

export interface AccountingConfig {
  provider: string;
  credentials: Record<string, string>;
  connected: boolean;
}

export interface AccountingTestResult {
  success: boolean;
  message: string;
}

export function useAccountingSettings() {
  return useQuery({
    queryKey: ["settings", "accounting"],
    queryFn: () => apiClient<AccountingConfig>("/v1/settings/accounting"),
  });
}

export function useUpdateAccountingSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { provider: string; credentials: Record<string, string> }) =>
      apiClient<{ message: string; provider: string }>("/v1/settings/accounting", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "accounting"] });
      queryClient.invalidateQueries({ queryKey: ["settings", "invoicing"] });
    },
  });
}

export function useTestAccountingConnection() {
  return useMutation({
    mutationFn: () =>
      apiClient<AccountingTestResult>("/v1/settings/accounting/test", {
        method: "POST",
      }),
  });
}
