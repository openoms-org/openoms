"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

// --- Category Types ---

export interface AllegroCategory {
  id: string;
  name: string;
  parent?: { id: string };
  leaf: boolean;
  options?: {
    advertisement: boolean;
    advertisementPriceOptional: boolean;
    variantsByColorPatternAllowed: boolean;
    offersWithProductPublicationEnabled: boolean;
    productCreationEnabled: boolean;
  };
}

export interface AllegroCategoryList {
  categories: AllegroCategory[];
}

export interface AllegroMatchingCategory {
  id: string;
  name: string;
  parent?: AllegroMatchingCategory | null;
}

export interface AllegroMatchingCategoriesResponse {
  matchingCategories: AllegroMatchingCategory[];
}

export interface AllegroCategoryParameter {
  id: string;
  name: string;
  type: string;
  required: boolean;
  unit?: string;
  options?: {
    variantsAllowed: boolean;
    ambiguousValueId?: string;
    dependsOnParameterId?: string;
  };
  restrictions?: {
    min?: number;
    max?: number;
    range: boolean;
    precision: number;
    minLength?: number;
    maxLength?: number;
  };
  dictionary?: { id: string; value: string }[];
}

export interface AllegroCategoryParameterList {
  parameters: AllegroCategoryParameter[];
}

// --- Product Catalog Types ---

export interface AllegroCatalogProduct {
  id: string;
  name: string;
  category?: { id: string };
  images?: { url: string }[];
  parameters?: {
    id: string;
    name: string;
    values?: string[];
    valuesIds?: string[];
    unit?: string;
  }[];
  description?: {
    sections: { items: { type: string; content: string }[] }[];
  };
}

export interface AllegroCatalogProductList {
  products: AllegroCatalogProduct[];
  count: number;
}

// --- Pricing/Fee Types ---

export interface AllegroFeePreview {
  commissions: { type: string; rate: { amount: string; currency: string } }[];
  quotes: { type: string; fee: { amount: string; currency: string }; name: string }[];
}

export interface AllegroCommissionList {
  commissions: {
    categoryId: string;
    rates: { type: string; value: number; percent: number }[];
  }[];
}

// --- Category Hooks ---

export function useAllegroCategories(parentId?: string | null) {
  const searchParams = new URLSearchParams();
  if (parentId) searchParams.set("parent_id", parentId);
  const qs = searchParams.toString();
  return useQuery({
    queryKey: ["allegro", "categories", parentId],
    queryFn: () =>
      apiClient<AllegroCategoryList>(
        `/v1/integrations/allegro/categories${qs ? `?${qs}` : ""}`
      ),
    staleTime: 1000 * 60 * 60, // 1h - categories rarely change
  });
}

export function useAllegroCategorySearch(query: string) {
  return useQuery({
    queryKey: ["allegro", "categories", "search", query],
    queryFn: () =>
      apiClient<AllegroMatchingCategoriesResponse>(
        `/v1/integrations/allegro/categories/search?name=${encodeURIComponent(query)}`
      ),
    enabled: query.length >= 2,
    staleTime: 1000 * 60 * 30, // 30min
  });
}

export function useAllegroCategoryParams(categoryId: string | null) {
  return useQuery({
    queryKey: ["allegro", "category-params", categoryId],
    queryFn: () =>
      apiClient<AllegroCategoryParameterList>(
        `/v1/integrations/allegro/categories/${categoryId}/parameters`
      ),
    enabled: !!categoryId,
    staleTime: 1000 * 60 * 60, // 1h - params rarely change
  });
}

// --- Product Catalog Hooks ---

export function useAllegroProductSearch(params?: {
  phrase?: string;
  mode?: string; // "GTIN" for EAN search, "MPN" for manufacturer part number
  category_id?: string;
  limit?: number;
  offset?: number;
}) {
  const searchParams = new URLSearchParams();
  if (params?.phrase) searchParams.set("phrase", params.phrase);
  if (params?.mode) searchParams.set("mode", params.mode);
  if (params?.category_id) searchParams.set("category_id", params.category_id);
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  return useQuery({
    queryKey: ["allegro", "product-catalog", params],
    queryFn: () =>
      apiClient<AllegroCatalogProductList>(
        `/v1/integrations/allegro/products/catalog${qs ? `?${qs}` : ""}`
      ),
    enabled: !!params?.phrase,
  });
}

// --- Pricing Hooks ---

export function useAllegroFees(offerId: string | null) {
  return useQuery({
    queryKey: ["allegro", "fees", offerId],
    queryFn: () =>
      apiClient<AllegroFeePreview>(
        `/v1/integrations/allegro/pricing/fees?offer_id=${offerId}`
      ),
    enabled: !!offerId,
  });
}

export function useAllegroCommissions(categoryId: string | null) {
  return useQuery({
    queryKey: ["allegro", "commissions", categoryId],
    queryFn: () =>
      apiClient<AllegroCommissionList>(
        `/v1/integrations/allegro/pricing/commissions?category_id=${categoryId}`
      ),
    enabled: !!categoryId,
  });
}
