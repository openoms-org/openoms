import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  AISuggestion,
  AIBulkCategorizeResponse,
  AIDescribeRequest,
  AITextResult,
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
    mutationFn: (req: AIDescribeRequest) =>
      apiClient<AISuggestion>("/v1/ai/describe", {
        method: "POST",
        body: JSON.stringify(req),
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

export function useImproveDescription() {
  return useMutation({
    mutationFn: (data: { description: string; style?: string; language?: string }) =>
      apiClient<AITextResult>("/v1/ai/improve", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}

export function useTranslateDescription() {
  return useMutation({
    mutationFn: (data: { description: string; target_language: string }) =>
      apiClient<AITextResult>("/v1/ai/translate", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}
