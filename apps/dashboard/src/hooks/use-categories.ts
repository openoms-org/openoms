import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  ProductCategory,
  CreateCategoryRequest,
  UpdateCategoryRequest,
} from "@/types/api";

function useCategories(params?: { tree?: boolean; parent_id?: string }) {
  const query = new URLSearchParams();
  if (params?.tree) query.set("tree", "true");
  if (params?.parent_id) query.set("parent_id", params.parent_id);

  const qs = query.toString();

  return useQuery({
    queryKey: ["categories", params],
    queryFn: () =>
      apiClient<ProductCategory[]>(
        `/v1/categories${qs ? `?${qs}` : ""}`
      ),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCategoryTree() {
  return useCategories({ tree: true });
}

export function useCreateCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateCategoryRequest) =>
      apiClient<ProductCategory>("/v1/categories", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
  });
}

export function useUpdateCategory(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateCategoryRequest) =>
      apiClient<ProductCategory>(`/v1/categories/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
  });
}

export function useDeleteCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/categories/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
  });
}
