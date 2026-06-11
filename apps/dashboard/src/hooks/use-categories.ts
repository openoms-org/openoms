import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
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

const categoryHooks = createCrudHooks<
  ProductCategory,
  CreateCategoryRequest,
  UpdateCategoryRequest
>({
  resourceKey: "categories",
  basePath: "/v1/categories",
});

export const useCreateCategory = categoryHooks.useCreate;
export const useUpdateCategory = categoryHooks.useUpdate;
export const useDeleteCategory = categoryHooks.useDelete;
