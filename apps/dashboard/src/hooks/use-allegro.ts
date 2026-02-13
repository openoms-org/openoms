"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";

// --- Types ---

export interface AllegroDeliveryService {
  id: string;
  name: string;
  carrierId: string;
}

export interface AllegroCreateShipmentCommand {
  commandId: string;
  input: {
    deliveryMethodId: string;
    credentialsId?: string;
    sender: AllegroShipmentAddress;
    receiver: AllegroShipmentAddress;
    packages: AllegroShipmentPackage[];
    labelFormat?: string;
  };
}

export interface AllegroShipmentAddress {
  name?: string;
  company?: string;
  street: string;
  city: string;
  zipCode: string;
  countryCode: string;
  phone?: string;
  email?: string;
}

export interface AllegroShipmentPackage {
  type?: string;
  length?: { value: number; unit: string };
  width?: { value: number; unit: string };
  height?: { value: number; unit: string };
  weight?: { value: number; unit: string };
}

export interface AllegroCreateShipmentResponse {
  commandId: string;
  shipmentId: string;
  status: string;
}

// --- Carrier Types ---

export interface AllegroCarrier {
  id: string;
  name: string;
}

// --- Hooks ---

// Fulfillment + tracking (Batch 1)

export function useAllegroCarriers() {
  return useQuery({
    queryKey: ["allegro", "carriers"],
    queryFn: () =>
      apiClient<{ carriers: AllegroCarrier[] }>(
        "/v1/integrations/allegro/carriers"
      ),
  });
}

export function useAllegroFulfillment(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { status: string }) =>
      apiClient<{ status: string }>(
        `/v1/integrations/allegro/orders/${orderId}/fulfillment`,
        { method: "POST", body: JSON.stringify(data) }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders", orderId] });
    },
  });
}

export function useAllegroTracking(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { carrier_id: string; waybill: string }) =>
      apiClient<{ status: string }>(
        `/v1/integrations/allegro/orders/${orderId}/tracking`,
        { method: "POST", body: JSON.stringify(data) }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders", orderId] });
    },
  });
}

export function useAllegroSync() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ synced_count: number; cursor: string }>(
        "/v1/integrations/allegro/sync",
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}

// Shipment management ("Wysyłam z Allegro")

export function useAllegroDeliveryServices() {
  return useQuery({
    queryKey: ["allegro", "delivery-services"],
    queryFn: () =>
      apiClient<{ delivery_services: AllegroDeliveryService[] }>(
        "/v1/integrations/allegro/delivery-services"
      ),
  });
}

export function useCreateAllegroShipment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (cmd: AllegroCreateShipmentCommand) =>
      apiClient<AllegroCreateShipmentResponse>(
        "/v1/integrations/allegro/shipments",
        {
          method: "POST",
          body: JSON.stringify(cmd),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "shipments"] });
    },
  });
}

export function useAllegroLabel(shipmentId: string | null) {
  return useQuery({
    queryKey: ["allegro", "label", shipmentId],
    queryFn: async () => {
      if (!shipmentId) return null;
      const res = await apiFetch(
        `/v1/integrations/allegro/shipments/${shipmentId}/label`
      );
      return res.blob();
    },
    enabled: !!shipmentId,
  });
}

export async function downloadAllegroLabel(shipmentId: string) {
  const res = await apiFetch(
    `/v1/integrations/allegro/shipments/${shipmentId}/label`
  );
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `etykieta-${shipmentId}.pdf`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 60000);
}

export async function downloadAllegroProtocol(shipmentIds: string[]) {
  const res = await apiFetch("/v1/integrations/allegro/protocol", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ shipment_ids: shipmentIds }),
  });
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "protokol-allegro.pdf";
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 60000);
}

// --- Messaging Types ---

export interface AllegroThread {
  id: string;
  subject: string;
  interlocutor: { id: string; login: string };
  lastMessageDateTime: string;
  read: boolean;
  offer?: { id: string; name: string };
}

export interface AllegroThreadList {
  threads: AllegroThread[];
  count: number;
}

export interface AllegroMessage {
  id: string;
  text: string;
  author: { login: string; isInterlocutor: boolean };
  createdAt: string;
  type: string;
  hasAdditionalAttachments: boolean;
}

export interface AllegroMessageList {
  messages: AllegroMessage[];
  count: number;
}

// --- Return Types ---

export interface AllegroCustomerReturn {
  id: string;
  createdAt: string;
  referenceNumber: string;
  buyer: { login: string; email: string };
  items: { offerId: string; name: string; quantity: number }[];
  refund?: { amount: string; currency: string };
  status: string;
  parcelSentByBuyer: boolean;
}

export interface AllegroCustomerReturnList {
  customerReturns: AllegroCustomerReturn[];
  count: number;
}

// --- Refund Types ---

export interface AllegroRefund {
  id: string;
  payment: { id: string };
  reason: string;
  status: string;
  createdAt: string;
  totalValue: { amount: string; currency: string };
  lineItems: { offerId: string; quantity: number; amount: { amount: string; currency: string } }[];
}

export interface AllegroRefundList {
  refunds: AllegroRefund[];
  count: number;
}

export interface AllegroCreateRefundRequest {
  payment: { id: string };
  reason: string;
  lineItems: { offerId: string; quantity: number; amount: { amount: string; currency: string } }[];
}

// --- Messaging Hooks ---

export function useAllegroThreads(params?: { limit?: number; offset?: number }) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));
  const query = searchParams.toString();

  return useQuery({
    queryKey: ["allegro", "threads", params],
    queryFn: () =>
      apiClient<AllegroThreadList>(
        `/v1/integrations/allegro/messages${query ? `?${query}` : ""}`
      ),
  });
}

export function useAllegroMessages(threadId: string | null, params?: { limit?: number; offset?: number }) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));
  const query = searchParams.toString();

  return useQuery({
    queryKey: ["allegro", "messages", threadId, params],
    queryFn: () =>
      apiClient<AllegroMessageList>(
        `/v1/integrations/allegro/messages/${threadId}${query ? `?${query}` : ""}`
      ),
    enabled: !!threadId,
  });
}

export function useSendAllegroMessage(threadId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (text: string) =>
      apiClient<AllegroMessage>(
        `/v1/integrations/allegro/messages/${threadId}`,
        {
          method: "POST",
          body: JSON.stringify({ text }),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "messages", threadId] });
      queryClient.invalidateQueries({ queryKey: ["allegro", "threads"] });
    },
  });
}

// --- Returns Hooks ---

export function useAllegroReturns(params?: { limit?: number; offset?: number; status?: string }) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));
  if (params?.status) searchParams.set("status", params.status);
  const query = searchParams.toString();

  return useQuery({
    queryKey: ["allegro", "returns", params],
    queryFn: () =>
      apiClient<AllegroCustomerReturnList>(
        `/v1/integrations/allegro/returns${query ? `?${query}` : ""}`
      ),
  });
}

export function useAllegroReturn(returnId: string | null) {
  return useQuery({
    queryKey: ["allegro", "returns", returnId],
    queryFn: () =>
      apiClient<AllegroCustomerReturn>(
        `/v1/integrations/allegro/returns/${returnId}`
      ),
    enabled: !!returnId,
  });
}

export function useRejectAllegroReturn(returnId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reason: string) =>
      apiClient<{ status: string }>(
        `/v1/integrations/allegro/returns/${returnId}/reject`,
        {
          method: "POST",
          body: JSON.stringify({ reason }),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "returns"] });
      queryClient.invalidateQueries({ queryKey: ["allegro", "returns", returnId] });
    },
  });
}

// --- Refund Hooks ---

export function useCreateAllegroRefund() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AllegroCreateRefundRequest) =>
      apiClient<AllegroRefund>("/v1/integrations/allegro/refunds", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "refunds"] });
    },
  });
}

export function useAllegroRefunds(params?: { limit?: number; offset?: number }) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null) searchParams.set("offset", String(params.offset));
  const query = searchParams.toString();

  return useQuery({
    queryKey: ["allegro", "refunds", params],
    queryFn: () =>
      apiClient<AllegroRefundList>(
        `/v1/integrations/allegro/refunds${query ? `?${query}` : ""}`
      ),
  });
}

// --- Account & Offers Types (Batch 4) ---

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
}

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

export interface AllegroOfferList {
  offers: AllegroOffer[];
  count: number;
  totalCount: number;
}

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

// --- Account & Offers Hooks (Batch 4) ---

export function useAllegroAccount() {
  return useQuery({
    queryKey: ["allegro", "account"],
    queryFn: () =>
      apiClient<AllegroAccountInfo>("/v1/integrations/allegro/account"),
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

// --- Category Types (Batch 5 — Catalog) ---

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

// --- Category Hooks (Batch 5) ---

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
  category_id?: string;
  limit?: number;
  offset?: number;
}) {
  const searchParams = new URLSearchParams();
  if (params?.phrase) searchParams.set("phrase", params.phrase);
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

export function useAllegroPromotion(promotionId: string | null) {
  return useQuery({
    queryKey: ["allegro", "promotions", promotionId],
    queryFn: () =>
      apiClient<AllegroPromotion>(
        `/v1/integrations/allegro/promotions/${promotionId}`
      ),
    enabled: !!promotionId,
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

export function useAllegroShippingRate(rateId: string | null) {
  return useQuery({
    queryKey: ["allegro", "shipping-rates", rateId],
    queryFn: () =>
      apiClient<AllegroShippingRateSet>(
        `/v1/integrations/allegro/shipping-rates/${rateId}`
      ),
    enabled: !!rateId,
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

export interface AutoGenerateShippingRateRequest {
  weight_kg: number;
  width_cm: number;
  height_cm: number;
  length_cm: number;
  name?: string;
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

// --- Dispute Types ---

export interface AllegroDispute {
  id: string;
  subject: string;
  status: string;
  buyer: { login: string };
  checkoutForm: { id: string };
  messages?: AllegroDisputeMessage[];
  createdAt: string;
  updatedAt: string;
}

export interface AllegroDisputeList {
  disputes: AllegroDispute[];
  count: number;
}

export interface AllegroDisputeMessage {
  id: string;
  text: string;
  author: string; // BUYER or SELLER
  createdAt: string;
  type: string;
}

export interface AllegroDisputeMessageList {
  messages: AllegroDisputeMessage[];
  count: number;
}

// --- Dispute Hooks ---

export function useAllegroDisputes(params?: {
  limit?: number;
  offset?: number;
  status?: string;
}) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null)
    searchParams.set("offset", String(params.offset));
  if (params?.status) searchParams.set("status", params.status);
  const query = searchParams.toString();

  return useQuery({
    queryKey: ["allegro", "disputes", params],
    queryFn: () =>
      apiClient<AllegroDisputeList>(
        `/v1/integrations/allegro/disputes${query ? `?${query}` : ""}`
      ),
  });
}

export function useAllegroDisputeMessages(disputeId: string | null) {
  return useQuery({
    queryKey: ["allegro", "dispute-messages", disputeId],
    queryFn: () =>
      apiClient<AllegroDisputeMessageList>(
        `/v1/integrations/allegro/disputes/${disputeId}/messages`
      ),
    enabled: !!disputeId,
  });
}

export function useSendAllegroDisputeMessage(disputeId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { text: string; type?: string }) =>
      apiClient<AllegroDisputeMessage>(
        `/v1/integrations/allegro/disputes/${disputeId}/messages`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro", "dispute-messages", disputeId],
      });
      queryClient.invalidateQueries({ queryKey: ["allegro", "disputes"] });
    },
  });
}

// --- Rating Management Types ---

export interface AllegroRatingAnswer {
  id?: string;
  text: string;
  createdAt?: string;
}

// --- Rating Management Hooks ---

export function useAllegroRatings(params?: {
  limit?: number;
  offset?: number;
}) {
  const searchParams = new URLSearchParams();
  if (params?.limit != null) searchParams.set("limit", String(params.limit));
  if (params?.offset != null)
    searchParams.set("offset", String(params.offset));
  const query = searchParams.toString();

  return useQuery({
    queryKey: ["allegro", "ratings", params],
    queryFn: () =>
      apiClient<{ ratings: AllegroUserRating[]; count: number }>(
        `/v1/integrations/allegro/ratings${query ? `?${query}` : ""}`
      ),
  });
}

export function useAllegroRatingAnswer(ratingId: string | null) {
  return useQuery({
    queryKey: ["allegro", "rating-answer", ratingId],
    queryFn: () =>
      apiClient<AllegroRatingAnswer>(
        `/v1/integrations/allegro/ratings/${ratingId}/answer`
      ),
    enabled: !!ratingId,
  });
}

export function useCreateAllegroRatingAnswer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      ratingId,
      text,
    }: {
      ratingId: string;
      text: string;
    }) =>
      apiClient<AllegroRatingAnswer>(
        `/v1/integrations/allegro/ratings/${ratingId}/answer`,
        {
          method: "PUT",
          body: JSON.stringify({ text }),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "ratings"] });
      queryClient.invalidateQueries({
        queryKey: ["allegro", "rating-answer"],
      });
    },
  });
}

export function useDeleteAllegroRatingAnswer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ratingId: string) =>
      apiClient(`/v1/integrations/allegro/ratings/${ratingId}/answer`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "ratings"] });
      queryClient.invalidateQueries({
        queryKey: ["allegro", "rating-answer"],
      });
    },
  });
}

export function useRequestAllegroRatingRemoval() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      ratingId,
      reason,
    }: {
      ratingId: string;
      reason: string;
    }) =>
      apiClient(`/v1/integrations/allegro/ratings/${ratingId}/removal`, {
        method: "POST",
        body: JSON.stringify({ reason }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "ratings"] });
    },
  });
}

// AllegroUserRating represents a single rating in the rating management context.
export interface AllegroUserRating {
  id: string;
  rate: string; // POSITIVE, NEGATIVE, NEUTRAL
  comment: string;
  createdAt: string;
  buyer: { login: string };
  order: { id: string };
}

// ==================== Product Listings ====================

export interface ProductListing {
  id: string;
  tenant_id: string;
  product_id: string;
  integration_id: string;
  external_id?: string;
  status: string;
  url?: string;
  price_override?: number;
  stock_override?: number;
  sync_status: string;
  last_synced_at?: string;
  error_message?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateProductListingRequest {
  integration_id: string;
  category_id: string;
  parameters: { id: string; valuesIds?: string[]; values?: string[] }[];
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
