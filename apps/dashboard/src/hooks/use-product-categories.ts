import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { ProductCategoriesConfig } from "@/types/api";

export function useProductCategories() {
  return useQuery({
    queryKey: ["product-categories"],
    queryFn: () => apiClient<ProductCategoriesConfig>("/v1/product-categories"),
    staleTime: 5 * 60 * 1000,
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
