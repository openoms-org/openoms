"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { ProductListing } from "@/types/api";

export function useProductListings(productId: string) {
  return useQuery({
    queryKey: ["product-listings", productId],
    queryFn: () =>
      apiClient<ProductListing[]>(`/v1/products/${productId}/listings`),
    enabled: !!productId,
  });
}
