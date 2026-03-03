"use client";

import { useMutation } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api-client";

interface ImageDownloadResult {
  total: number;
  downloaded: number;
  failed: number;
  skipped: number;
}

export function useRedownloadImages() {
  return useMutation({
    mutationFn: async (): Promise<ImageDownloadResult> => {
      const res = await apiFetch("/v1/products/redownload-images", {
        method: "POST",
      });
      return res.json();
    },
  });
}
