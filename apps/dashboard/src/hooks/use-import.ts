"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api-client";
import type {
  ImportPreviewResponse,
  ImportResult,
  ImportColumnMapping,
} from "@/types/api";

export function useImportPreview() {
  return useMutation({
    mutationFn: async (file: File): Promise<ImportPreviewResponse> => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await apiFetch("/v1/orders/import/preview", {
        method: "POST",
        body: fd,
      });
      return res.json();
    },
  });
}

export function useImportOrders() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      file,
      mappings,
    }: {
      file: File;
      mappings: ImportColumnMapping[];
    }): Promise<ImportResult> => {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("mappings", JSON.stringify(mappings));
      const res = await apiFetch("/v1/orders/import", {
        method: "POST",
        body: fd,
      });
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}
