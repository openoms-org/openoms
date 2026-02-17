import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  Invoice,
  ListResponse,
  InvoiceListParams,
  CreateInvoiceRequest,
  InvoicingSettings,
} from "@/types/api";

export function useInvoices(params: InvoiceListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.provider) query.set("provider", params.provider);
  if (params.order_id) query.set("order_id", params.order_id);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["invoices", params],
    queryFn: () =>
      apiClient<ListResponse<Invoice>>(`/v1/invoices${qs ? `?${qs}` : ""}`),
  });
}

export function useInvoice(id: string) {
  return useQuery({
    queryKey: ["invoices", id],
    queryFn: () => apiClient<Invoice>(`/v1/invoices/${id}`),
    enabled: !!id,
  });
}

export function useOrderInvoices(orderId: string) {
  return useQuery({
    queryKey: ["invoices", "order", orderId],
    queryFn: () => apiClient<Invoice[]>(`/v1/orders/${orderId}/invoices`),
    enabled: !!orderId,
  });
}

export function useCreateInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateInvoiceRequest) =>
      apiClient<Invoice>("/v1/invoices", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
    },
  });
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
