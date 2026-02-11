import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  AISuggestion,
  AIBulkCategorizeResponse,
} from "@/types/api";

export function useSuggestCategories() {
  return useMutation({
    mutationFn: (productId: string) =>
      apiClient<AISuggestion>("/v1/ai/categorize", {
        method: "POST",
        body: JSON.stringify({ product_id: productId }),
      }),
  });
}

export function useGenerateDescription() {
  return useMutation({
    mutationFn: (productId: string) =>
      apiClient<AISuggestion>("/v1/ai/describe", {
        method: "POST",
        body: JSON.stringify({ product_id: productId }),
      }),
  });
}

export function useBulkCategorize() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (productIds: string[]) =>
      apiClient<AIBulkCategorizeResponse>("/v1/ai/bulk-categorize", {
        method: "POST",
        body: JSON.stringify({ product_ids: productIds }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
    },
  });
}
