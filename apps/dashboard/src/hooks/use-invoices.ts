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

/**
 * Lists every invoice linked to a given order (GET /v1/orders/{id}/invoices).
 * Used to gate manual invoice issuance: the order already has an active
 * (non-cancelled, non-error) invoice ⇒ do not offer "issue invoice" again,
 * mirroring the auto-create dedup in invoice_service.HandleOrderStatusChange.
 */
export function useOrderInvoices(
  orderId: string,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: ["invoices", "by-order", orderId],
    queryFn: () => apiClient<Invoice[]>(`/v1/orders/${orderId}/invoices`),
    enabled: !!orderId && (options.enabled ?? true),
  });
}

/** An invoice blocks re-issuance unless it has been cancelled or errored. */
export function isActiveInvoice(invoice: Invoice): boolean {
  return invoice.status !== "cancelled" && invoice.status !== "error";
}

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
