"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

// --- Account Types ---

export interface AllegroUser {
  id: string;
  login: string;
  email: string;
  features: string[];
}

export interface AllegroSellerQuality {
  recommendPercentage: string;
  recommendCount: number;
}

export interface AllegroAccountInfo {
  user: AllegroUser;
  quality: AllegroSellerQuality;
  sandbox?: boolean;
}

// --- Billing Types ---

export interface AllegroBillingEntry {
  id: string;
  type: { id: string; name: string; group: string };
  amount: { amount: string; currency: string };
  occurredAt: string;
}

export interface AllegroBillingList {
  billingEntries: AllegroBillingEntry[];
  count: number;
}

// --- Return Policy Types ---

export interface AllegroReturnPolicy {
  id: string;
  name: string;
  availability?: { range: string; restrictionCause?: string };
  withdrawalPeriod?: string;
  returnCost?: { coveredBy: string };
  options?: AllegroReturnOptions;
  address?: {
    name: string;
    street: string;
    city: string;
    postCode: string;
    countryCode: string;
  };
  description?: string;
  contact?: {
    phoneNumber?: string;
    email?: string;
  };
}

export interface AllegroReturnPolicyList {
  returnPolicies: AllegroReturnPolicy[];
}

export interface AllegroReturnOptions {
  cashOnDeliveryNotAllowed: boolean;
  freeAccessoriesReturnRequired: boolean;
  refundLoweredByReceivedDiscount: boolean;
  businessReturnAllowed: boolean;
  collectBySellerOnly: boolean;
}

export interface AllegroCreateReturnPolicyRequest {
  name: string;
  availability?: { range: string; restrictionCause?: string };
  withdrawalPeriod?: string; // ISO 8601 e.g. "P14D"
  returnCost?: { coveredBy: string };
  options?: AllegroReturnOptions;
  address?: {
    name: string;
    street: string;
    city: string;
    postCode: string;
    countryCode: string;
  };
  description?: string;
  contact?: {
    phoneNumber?: string;
    email?: string;
  };
}

// --- Implied Warranty Types ---

export interface AllegroImpliedWarranty {
  id: string;
  name: string;
  individual?: { period: string; type: string };
  corporate?: { period: string; type: string };
}

export interface AllegroWarrantyList {
  impliedWarranties: AllegroImpliedWarranty[];
}

export interface AllegroCreateWarrantyRequest {
  name: string;
  individual?: { period: string; type: string };
  corporate?: { period: string; type: string };
  address?: {
    name: string;
    street: string;
    city: string;
    postCode: string;
    countryCode: string;
  };
}

// --- Size Table Types ---

export interface AllegroSizeTable {
  id: string;
  name: string;
  type: string;
  headers: { name: string }[];
  values: string[][];
}

export interface AllegroSizeTableList {
  sizeTables: AllegroSizeTable[];
}

export interface AllegroCreateSizeTableRequest {
  name: string;
  type: string;
  headers: { name: string }[];
  values: string[][];
}

// --- Promotion Types ---

export interface AllegroPromotion {
  id: string;
  name: string;
  benefits: {
    specification?: {
      type: string;
      value?: { amount: string; currency: string };
    };
  }[];
  criteria?: {
    type: string;
    offers?: { id: string; quantity?: number }[];
    value?: { amount: string; currency: string };
  }[];
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface AllegroPromotionList {
  promotions: AllegroPromotion[];
  count: number;
}

export interface AllegroCreatePromotionRequest {
  name: string;
  benefits: {
    specification?: {
      type: string;
      value?: { amount: string; currency: string };
    };
  }[];
  criteria?: {
    type: string;
    offers?: { id: string; quantity?: number }[];
    value?: { amount: string; currency: string };
  }[];
}

export interface AllegroPromoBadge {
  id: string;
  name: string;
  description?: string;
  price: { amount: string; currency: string };
}

export interface AllegroPromoBadgeList {
  packages: AllegroPromoBadge[];
}

// --- Delivery Settings Types ---

export interface AllegroDeliverySettings {
  freeDelivery?: {
    amount: { amount: string; currency: string };
    threshold: { amount: string; currency: string };
  };
  joinPolicy?: { strategy: string };
  customCost?: { allowed: boolean };
  abroadDelivery?: { enabled: boolean };
}

export interface AllegroShippingRateSet {
  id: string;
  name: string;
  rates: AllegroShippingRateEntry[];
}

export interface AllegroShippingRateEntry {
  deliveryMethod: { id: string };
  maxQuantityPerPackage?: number;
  firstItemRate: { amount: string; currency: string };
  nextItemRate: { amount: string; currency: string };
  shippingTime?: { from: string; to: string };
}

export interface AllegroShippingRateList {
  shippingRates: AllegroShippingRateSet[];
}

export interface AllegroCreateShippingRateRequest {
  name: string;
  rates: AllegroShippingRateEntry[];
}

export interface AutoGenerateShippingRateRequest {
  weight_kg: number;
  width_cm: number;
  height_cm: number;
  length_cm: number;
  name?: string;
}

export interface AllegroDeliveryMethodItem {
  id: string;
  name: string;
  paymentPolicy: string;
  shippingRatesConstraints?: {
    maxQuantityPerPackage?: { value: number };
    allowedForFreeShipping: boolean;
  };
}

export interface AllegroDeliveryMethodList {
  deliveryMethods: AllegroDeliveryMethodItem[];
}

// --- Account Hooks ---

export function useAllegroAccount(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ["allegro", "account"],
    queryFn: () =>
      apiClient<AllegroAccountInfo>("/v1/integrations/allegro/account"),
    enabled: options?.enabled ?? true,
  });
}

export function useAllegroBilling(params?: {
  limit?: number;
  offset?: number;
  type_group?: string;
}) {
  const searchParams = new URLSearchParams();
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  if (params?.type_group) searchParams.set("type_group", params.type_group);
  const qs = searchParams.toString();
  return useQuery({
    queryKey: ["allegro", "billing", params],
    queryFn: () =>
      apiClient<AllegroBillingList>(
        `/v1/integrations/allegro/billing${qs ? `?${qs}` : ""}`
      ),
  });
}

// --- Return Policy Hooks ---

export function useAllegroReturnPolicies() {
  return useQuery({
    queryKey: ["allegro", "return-policies"],
    queryFn: () =>
      apiClient<AllegroReturnPolicyList>(
        "/v1/integrations/allegro/return-policies"
      ),
    staleTime: 1000 * 60 * 30, // 30min
  });
}

export function useCreateAllegroReturnPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroCreateReturnPolicyRequest) =>
      apiClient<AllegroReturnPolicy>(
        "/v1/integrations/allegro/return-policies",
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "return-policies"],
      });
    },
  });
}

export function useUpdateAllegroReturnPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      policyId,
      data,
    }: {
      policyId: string;
      data: AllegroCreateReturnPolicyRequest;
    }) =>
      apiClient<AllegroReturnPolicy>(
        `/v1/integrations/allegro/return-policies/${policyId}`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "return-policies"],
      });
    },
  });
}

// --- Warranty Hooks ---

export function useAllegroWarranties() {
  return useQuery({
    queryKey: ["allegro", "warranties"],
    queryFn: () =>
      apiClient<AllegroWarrantyList>(
        "/v1/integrations/allegro/warranties"
      ),
    staleTime: 1000 * 60 * 30, // 30min
  });
}

export function useCreateAllegroWarranty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroCreateWarrantyRequest) =>
      apiClient<AllegroImpliedWarranty>(
        "/v1/integrations/allegro/warranties",
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "warranties"],
      });
    },
  });
}

export function useUpdateAllegroWarranty() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      warrantyId,
      data,
    }: {
      warrantyId: string;
      data: AllegroCreateWarrantyRequest;
    }) =>
      apiClient<AllegroImpliedWarranty>(
        `/v1/integrations/allegro/warranties/${warrantyId}`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "warranties"],
      });
    },
  });
}

// --- Size Table Hooks ---

export function useAllegroSizeTables() {
  return useQuery({
    queryKey: ["allegro", "size-tables"],
    queryFn: () =>
      apiClient<AllegroSizeTableList>(
        "/v1/integrations/allegro/size-tables"
      ),
  });
}

export function useCreateAllegroSizeTable() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroCreateSizeTableRequest) =>
      apiClient<AllegroSizeTable>(
        "/v1/integrations/allegro/size-tables",
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "size-tables"],
      });
    },
  });
}

export function useUpdateAllegroSizeTable() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      tableId,
      data,
    }: {
      tableId: string;
      data: AllegroCreateSizeTableRequest;
    }) =>
      apiClient<AllegroSizeTable>(
        `/v1/integrations/allegro/size-tables/${tableId}`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "size-tables"],
      });
    },
  });
}

export function useDeleteAllegroSizeTable() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tableId: string) =>
      apiClient(`/v1/integrations/allegro/size-tables/${tableId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "size-tables"],
      });
    },
  });
}

// --- Promotion Hooks ---

export function useAllegroPromotions(params?: {
  limit?: number;
  offset?: number;
}) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null)
    searchParams.set("offset", String(params.offset));
  const qs = searchParams.toString();
  return useQuery({
    queryKey: ["allegro", "promotions", params],
    queryFn: () =>
      apiClient<AllegroPromotionList>(
        `/v1/integrations/allegro/promotions${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useCreateAllegroPromotion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroCreatePromotionRequest) =>
      apiClient<AllegroPromotion>(
        "/v1/integrations/allegro/promotions",
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "promotions"],
      });
    },
  });
}

export function useUpdateAllegroPromotion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      promotionId,
      data,
    }: {
      promotionId: string;
      data: AllegroCreatePromotionRequest;
    }) =>
      apiClient<AllegroPromotion>(
        `/v1/integrations/allegro/promotions/${promotionId}`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "promotions"],
      });
    },
  });
}

export function useDeleteAllegroPromotion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (promotionId: string) =>
      apiClient(`/v1/integrations/allegro/promotions/${promotionId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "promotions"],
      });
    },
  });
}

export function useAllegroPromoBadges() {
  return useQuery({
    queryKey: ["allegro", "promotion-badges"],
    queryFn: () =>
      apiClient<AllegroPromoBadgeList>(
        "/v1/integrations/allegro/promotion-badges"
      ),
  });
}

// --- Delivery Settings Hooks ---

export function useAllegroDeliverySettings() {
  return useQuery({
    queryKey: ["allegro", "delivery-settings"],
    queryFn: () =>
      apiClient<AllegroDeliverySettings>(
        "/v1/integrations/allegro/delivery-settings"
      ),
  });
}

export function useUpdateAllegroDeliverySettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroDeliverySettings) =>
      apiClient<{ status: string }>(
        "/v1/integrations/allegro/delivery-settings",
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "delivery-settings"],
      });
    },
  });
}

export function useAllegroShippingRates() {
  return useQuery({
    queryKey: ["allegro", "shipping-rates"],
    queryFn: () =>
      apiClient<AllegroShippingRateList>(
        "/v1/integrations/allegro/shipping-rates"
      ),
    staleTime: 1000 * 60 * 30, // 30min
  });
}

export function useCreateAllegroShippingRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroCreateShippingRateRequest) =>
      apiClient<AllegroShippingRateSet>(
        "/v1/integrations/allegro/shipping-rates",
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "shipping-rates"],
      });
    },
  });
}

export function useUpdateAllegroShippingRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      rateId,
      data,
    }: {
      rateId: string;
      data: AllegroCreateShippingRateRequest;
    }) =>
      apiClient<AllegroShippingRateSet>(
        `/v1/integrations/allegro/shipping-rates/${rateId}`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "shipping-rates"],
      });
    },
  });
}

export function useAutoGenerateShippingRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AutoGenerateShippingRateRequest) =>
      apiClient<AllegroShippingRateSet>(
        "/v1/integrations/allegro/shipping-rates/auto-generate",
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "shipping-rates"],
      });
    },
  });
}

export function useAllegroDeliveryMethods() {
  return useQuery({
    queryKey: ["allegro", "delivery-methods"],
    queryFn: () =>
      apiClient<AllegroDeliveryMethodList>(
        "/v1/integrations/allegro/delivery-methods"
      ),
  });
}
