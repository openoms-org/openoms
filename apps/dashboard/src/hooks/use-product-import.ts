"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api-client";
import type { ProductImportPreview, ProductImportResult } from "@/types/api";

export function useProductImportPreview() {
  return useMutation({
    mutationFn: async (file: File): Promise<ProductImportPreview> => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await apiFetch("/v1/products/import/preview", {
        method: "POST",
        body: fd,
      });
      return res.json();
    },
  });
}

export function useProductImport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file: File): Promise<ProductImportResult> => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await apiFetch("/v1/products/import", {
        method: "POST",
        body: fd,
      });
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
    },
  });
}
