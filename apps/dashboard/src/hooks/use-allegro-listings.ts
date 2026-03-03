"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { ProductListing } from "@/types/api";

// --- Offer Types ---

export interface AllegroOffer {
  id: string;
  name: string;
  sellingMode?: {
    price: { amount: string; currency: string };
    format: string;
  };
  stock?: { available: number; unit: string };
  publication?: { status: string };
  primaryImage?: { url: string };
}

interface AllegroOfferList {
  offers: AllegroOffer[];
  count: number;
  totalCount: number;
}

// --- Listing Search Types ---

interface AllegroListingSearchItem {
  id: string;
  name: string;
  category?: { id: string };
  parameters?: {
    id: string;
    name: string;
    values?: string[];
    valuesIds?: string[];
  }[];
}

interface AllegroListingSearchResult {
  items: {
    promoted: AllegroListingSearchItem[];
    regular: AllegroListingSearchItem[];
  };
}

// --- Product Listing Types ---

export interface CreateProductListingRequest {
  integration_id: string;
  category_id: string;
  parameters: { id: string; valuesIds?: string[]; values?: string[] }[];
  description_html?: string;
  shipping_rate_id: string;
  return_policy_id: string;
  warranty_id: string;
  handling_time: string;
  price_override?: number;
  stock_override?: number;
  location?: {
    city: string;
    post_code: string;
    province: string;
    country_code: string;
  };
}

interface CreateWooCommerceListingRequest {
  integration_id: string;
  price_override?: number;
  stock_override?: number;
  categories?: string[];
  description?: string;
}

// --- Offer Hooks ---

export function useAllegroOffers(params?: {
  limit?: number;
  offset?: number;
  name?: string;
  publication_status?: string;
}) {
  const searchParams = new URLSearchParams();
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  if (params?.name) searchParams.set("name", params.name);
  if (params?.publication_status)
    searchParams.set("publication_status", params.publication_status);
  const qs = searchParams.toString();
  return useQuery({
    queryKey: ["allegro", "offers", params],
    queryFn: () =>
      apiClient<AllegroOfferList>(
        `/v1/integrations/allegro/offers${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useDeactivateAllegroOffer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (offerId: string) =>
      apiClient(`/v1/integrations/allegro/offers/${offerId}/deactivate`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "offers"] });
    },
  });
}

export function useActivateAllegroOffer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (offerId: string) =>
      apiClient(`/v1/integrations/allegro/offers/${offerId}/activate`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "offers"] });
    },
  });
}

export function useUpdateAllegroOfferStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      offerId,
      quantity,
    }: {
      offerId: string;
      quantity: number;
    }) =>
      apiClient(`/v1/integrations/allegro/offers/${offerId}/stock`, {
        method: "PATCH",
        body: JSON.stringify({ quantity }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "offers"] });
    },
  });
}

export function useUpdateAllegroOfferPrice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      offerId,
      amount,
      currency,
    }: {
      offerId: string;
      amount: number;
      currency?: string;
    }) =>
      apiClient(`/v1/integrations/allegro/offers/${offerId}/price`, {
        method: "PATCH",
        body: JSON.stringify({ amount, currency: currency ?? "PLN" }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "offers"] });
    },
  });
}

// --- Listing Search Hook ---

export function useAllegroListingSearch(phrase: string | undefined) {
  return useQuery({
    queryKey: ["allegro", "listing-search", phrase],
    queryFn: () =>
      apiClient<AllegroListingSearchResult>(
        `/v1/integrations/allegro/offers/listing?phrase=${encodeURIComponent(phrase!)}&limit=1`
      ),
    enabled: !!phrase,
    staleTime: 1000 * 60 * 30, // 30min cache
  });
}

// --- Product Listing Hooks ---

export function useProductListings(productId: string) {
  return useQuery({
    queryKey: ["products", productId, "listings"],
    queryFn: () =>
      apiClient<ProductListing[]>(
        `/v1/products/${productId}/listings`
      ),
    enabled: !!productId,
  });
}

export function useCreateProductListing(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateProductListingRequest) =>
      apiClient<ProductListing>(
        `/v1/products/${productId}/listings/allegro`,
        { method: "POST", body: JSON.stringify(data) }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products", productId, "listings"] });
    },
  });
}

export function useCreateWooCommerceListing(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateWooCommerceListingRequest) =>
      apiClient<ProductListing>(
        `/v1/products/${productId}/listings/woocommerce`,
        { method: "POST", body: JSON.stringify(data) }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products", productId, "listings"] });
    },
  });
}

export function useDeleteProductListing(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient(`/v1/products/${productId}/listings/${listingId}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products", productId, "listings"] });
    },
  });
}

export function useSyncProductListing(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient<ProductListing>(
        `/v1/products/${productId}/listings/${listingId}/sync`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products", productId, "listings"] });
    },
  });
}

export function useUpdateListingSyncMode(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ listingId, mode }: { listingId: string; mode: 'auto' | 'manual' }) =>
      apiClient(
        `/v1/products/${productId}/listings/${listingId}`,
        { method: "PUT", body: JSON.stringify({ stock_sync_mode: mode }) }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products", productId, "listings"] });
    },
  });
}

export function useForcePushListing() {
  return useMutation({
    mutationFn: (listingId: string) =>
      apiClient(`/v1/stock-sync/push/listing/${listingId}`, { method: "POST" }),
  });
}
