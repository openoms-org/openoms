"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api-client";
import type { CustomerImportPreview, CustomerImportResult } from "@/types/api";

export function useCustomerImportPreview() {
  return useMutation({
    mutationFn: async (file: File): Promise<CustomerImportPreview> => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await apiFetch("/v1/customers/import/preview", {
        method: "POST",
        body: fd,
      });
      return res.json();
    },
  });
}

export function useCustomerImport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file: File): Promise<CustomerImportResult> => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await apiFetch("/v1/customers/import", {
        method: "POST",
        body: fd,
      });
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}
