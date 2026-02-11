"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  BundleComponent,
  CreateBundleComponentRequest,
  UpdateBundleComponentRequest,
  BundleStockResponse,
} from "@/types/api";

export function useBundleComponents(productId: string) {
  return useQuery({
    queryKey: ["bundles", productId],
    queryFn: () =>
      apiClient<BundleComponent[]>(`/v1/products/${productId}/bundle`),
    enabled: !!productId,
  });
}

export function useBundleStock(productId: string) {
  return useQuery({
    queryKey: ["bundles", productId, "stock"],
    queryFn: () =>
      apiClient<BundleStockResponse>(`/v1/products/${productId}/bundle/stock`),
    enabled: !!productId,
  });
}

export function useAddBundleComponent(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateBundleComponentRequest) =>
      apiClient<BundleComponent>(`/v1/products/${productId}/bundle`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bundles", productId] });
    },
  });
}

export function useUpdateBundleComponent(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      componentId,
      data,
    }: {
      componentId: string;
      data: UpdateBundleComponentRequest;
    }) =>
      apiClient<BundleComponent>(
        `/v1/products/${productId}/bundle/${componentId}`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bundles", productId] });
    },
  });
}

export function useRemoveBundleComponent(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (componentId: string) =>
      apiClient<void>(`/v1/products/${productId}/bundle/${componentId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bundles", productId] });
    },
  });
}
