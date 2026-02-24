"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

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

// --- Rating Management Types ---

export interface AllegroRatingAnswer {
  id?: string;
  text: string;
  createdAt?: string;
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
