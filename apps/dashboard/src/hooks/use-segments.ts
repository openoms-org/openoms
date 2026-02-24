import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  CustomerSegment,
  SegmentMember,
  CustomerRFM,
  ListResponse,
  SegmentListParams,
  CreateSegmentRequest,
  UpdateSegmentRequest,
  AddSegmentMemberRequest,
  PaginationParams,
} from "@/types/api";

const segmentHooks = createCrudHooks<
  CustomerSegment,
  CreateSegmentRequest,
  UpdateSegmentRequest,
  PaginationParams
>({
  resourceKey: "segments",
  basePath: "/v1/segments",
  updateMethod: "PUT",
});

export const useSegments = segmentHooks.useList;
export const useSegment = segmentHooks.useGet;
export const useCreateSegment = segmentHooks.useCreate;
export const useUpdateSegment = segmentHooks.useUpdate;
export const useDeleteSegment = segmentHooks.useDelete;

export function useSegmentMembers(
  segmentId: string,
  params: { limit?: number; offset?: number } = {}
) {
  const sp = buildSearchParams(params);
  const qs = sp.toString();

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
