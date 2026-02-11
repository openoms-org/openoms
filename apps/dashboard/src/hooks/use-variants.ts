import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  ProductVariant,
  ListResponse,
  VariantListParams,
  CreateVariantRequest,
  UpdateVariantRequest,
} from "@/types/api";

export function useVariants(productId: string, params: VariantListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.active != null) query.set("active", String(params.active));
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["variants", productId, params],
    queryFn: () =>
      apiClient<ListResponse<ProductVariant>>(
        `/v1/products/${productId}/variants${qs ? `?${qs}` : ""}`
      ),
    enabled: !!productId,
  });
}

export function useVariant(productId: string, id: string) {
  return useQuery({
    queryKey: ["variants", productId, id],
    queryFn: () =>
      apiClient<ProductVariant>(
        `/v1/products/${productId}/variants/${id}`
      ),
    enabled: !!productId && !!id,
  });
}

export function useCreateVariant(productId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateVariantRequest) =>
      apiClient<ProductVariant>(
        `/v1/products/${productId}/variants`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variants", productId] });
      queryClient.invalidateQueries({ queryKey: ["products", productId] });
    },
  });
}

export function useUpdateVariant(productId: string, id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateVariantRequest) =>
      apiClient<ProductVariant>(
        `/v1/products/${productId}/variants/${id}`,
        {
          method: "PATCH",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variants", productId] });
    },
  });
}

export function useDeleteVariant(productId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/products/${productId}/variants/${id}`,
        {
          method: "DELETE",
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variants", productId] });
      queryClient.invalidateQueries({ queryKey: ["products", productId] });
    },
  });
}
