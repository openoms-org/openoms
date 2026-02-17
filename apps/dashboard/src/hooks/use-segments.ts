import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  CustomerSegment,
  SegmentMember,
  CustomerRFM,
  ListResponse,
  SegmentListParams,
  CreateSegmentRequest,
  UpdateSegmentRequest,
  AddSegmentMemberRequest,
} from "@/types/api";

export function useSegments(params: SegmentListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["segments", params],
    queryFn: () =>
      apiClient<ListResponse<CustomerSegment>>(
        `/v1/segments${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useSegment(id: string) {
  return useQuery({
    queryKey: ["segments", id],
    queryFn: () => apiClient<CustomerSegment>(`/v1/segments/${id}`),
    enabled: !!id,
  });
}

export function useCreateSegment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSegmentRequest) =>
      apiClient<CustomerSegment>("/v1/segments", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["segments"] });
    },
  });
}

export function useUpdateSegment(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateSegmentRequest) =>
      apiClient<CustomerSegment>(`/v1/segments/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["segments"] });
      queryClient.invalidateQueries({ queryKey: ["segments", id] });
    },
  });
}

export function useDeleteSegment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/segments/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["segments"] });
    },
  });
}

export function useSegmentMembers(
  segmentId: string,
  params: { limit?: number; offset?: number } = {}
) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));

  const qs = query.toString();

  return useQuery({
    queryKey: ["segments", segmentId, "members", params],
    queryFn: () =>
      apiClient<ListResponse<SegmentMember>>(
        `/v1/segments/${segmentId}/members${qs ? `?${qs}` : ""}`
      ),
    enabled: !!segmentId,
  });
}

export function useAddSegmentMember(segmentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AddSegmentMemberRequest) =>
      apiClient<void>(`/v1/segments/${segmentId}/members`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["segments", segmentId, "members"],
      });
      queryClient.invalidateQueries({ queryKey: ["segments"] });
    },
  });
}

export function useRemoveSegmentMember(segmentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (customerId: string) =>
      apiClient<void>(`/v1/segments/${segmentId}/members/${customerId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["segments", segmentId, "members"],
      });
      queryClient.invalidateQueries({ queryKey: ["segments"] });
    },
  });
}

export function useRunRFMAnalysis() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<CustomerRFM[]>("/v1/segments/rfm-analysis", {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["segments"] });
    },
  });
}

export function useCustomerSegments(customerId: string) {
  return useQuery({
    queryKey: ["segments", "customer", customerId],
    queryFn: () =>
      apiClient<CustomerSegment[]>(`/v1/segments/customer/${customerId}`),
    enabled: !!customerId,
  });
}
