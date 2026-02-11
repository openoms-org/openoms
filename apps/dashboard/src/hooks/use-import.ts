"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/auth";
import { ApiClientError } from "@/lib/api-client";
import type {
  ImportPreviewResponse,
  ImportResult,
  ImportColumnMapping,
} from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchWithAuth(
  url: string,
  body: FormData
): Promise<Response> {
  const token = useAuthStore.getState().token;
  const headers: Record<string, string> = {};
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(url, {
    method: "POST",
    headers,
    body,
    credentials: "include",
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Request failed" }));
    throw new ApiClientError(res.status, errorBody.error);
  }

  return res;
}

export function useImportPreview() {
  return useMutation({
    mutationFn: async (file: File): Promise<ImportPreviewResponse> => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await fetchWithAuth(
        `${API_URL}/v1/orders/import/preview`,
        fd
      );
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
      const res = await fetchWithAuth(`${API_URL}/v1/orders/import`, fd);
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}
