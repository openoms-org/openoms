import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type { ProductCategoriesConfig } from "@/types/api";

export function useProductCategories() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["product-categories"],
    queryFn: () => apiClient<ProductCategoriesConfig>("/v1/product-categories"),
    staleTime: 5 * 60 * 1000,
    enabled: isAuthenticated,
  });
}

export function useUpdateProductCategories() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ProductCategoriesConfig) =>
      apiClient<ProductCategoriesConfig>("/v1/settings/product-categories", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["product-categories"] });
    },
  });
}
