import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { BGRemovalResult, BGRemovalStatus } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/** Check if background removal is configured on the server. */
export function useBGRemovalStatus() {
  return useQuery({
    queryKey: ["bg-removal-status"],
    queryFn: () =>
      apiClient<BGRemovalStatus>("/v1/images/remove-background/status"),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/** Upload an image file and remove its background (standalone). */
export function useRemoveBackground() {
  return useMutation({
    mutationFn: async (file: File): Promise<BGRemovalResult> => {
      const token = useAuthStore.getState().token;
      const formData = new FormData();
      formData.append("file", file);

      const headers: Record<string, string> = {};
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }

      const res = await fetch(`${API_URL}/v1/images/remove-background`, {
        method: "POST",
        headers,
        body: formData,
        credentials: "include",
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: "Request failed" }));
        throw new Error(body.error || "Usuwanie tła nie powiodło się");
      }

      return res.json();
    },
  });
}

/** Remove background from an existing product image. index=-1 for main image_url, 0+ for images[]. */
export function useRemoveProductImageBackground(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (index: number) =>
      apiClient<BGRemovalResult>(
        `/v1/products/${productId}/images/${index}/remove-background`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["product", productId] });
    },
  });
}
