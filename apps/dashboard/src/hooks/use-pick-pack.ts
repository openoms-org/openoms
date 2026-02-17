import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  PickPackSession,
  PickPackItem,
  ListResponse,
  PickPackSessionListParams,
  CreatePickPackSessionRequest,
  ScanItemRequest,
  ScanItemResponse,
  MarkPackedRequest,
} from "@/types/api";

export function usePickPackSessions(params: PickPackSessionListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.assigned_to) query.set("assigned_to", params.assigned_to);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["pick-pack-sessions", params],
    queryFn: () =>
      apiClient<ListResponse<PickPackSession>>(
        `/v1/pick-pack/sessions${qs ? `?${qs}` : ""}`
      ),
  });
}

export function usePickPackSession(id: string) {
  return useQuery({
    queryKey: ["pick-pack-sessions", id],
    queryFn: () => apiClient<PickPackSession>(`/v1/pick-pack/sessions/${id}`),
    enabled: !!id,
    refetchInterval: 5000, // poll for updates during active sessions
  });
}

export function useCreatePickPackSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePickPackSessionRequest) =>
      apiClient<PickPackSession>("/v1/pick-pack/sessions", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["pick-pack-sessions"] });
    },
  });
}

export function useScanItem(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ScanItemRequest) =>
      apiClient<ScanItemResponse>(
        `/v1/pick-pack/sessions/${sessionId}/scan`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["pick-pack-sessions", sessionId],
      });
    },
  });
}

export function useMoveToPacking(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<PickPackSession>(
        `/v1/pick-pack/sessions/${sessionId}/move-to-packing`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["pick-pack-sessions", sessionId],
      });
      queryClient.invalidateQueries({ queryKey: ["pick-pack-sessions"] });
    },
  });
}

export function useMarkItemPacked(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      itemId,
      data,
    }: {
      itemId: string;
      data: MarkPackedRequest;
    }) =>
      apiClient<PickPackItem>(
        `/v1/pick-pack/sessions/${sessionId}/items/${itemId}/pack`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["pick-pack-sessions", sessionId],
      });
    },
  });
}

export function useCompletePickPackSession(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<PickPackSession>(
        `/v1/pick-pack/sessions/${sessionId}/complete`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["pick-pack-sessions", sessionId],
      });
      queryClient.invalidateQueries({ queryKey: ["pick-pack-sessions"] });
    },
  });
}

export function useCancelPickPackSession(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<PickPackSession>(
        `/v1/pick-pack/sessions/${sessionId}/cancel`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["pick-pack-sessions", sessionId],
      });
      queryClient.invalidateQueries({ queryKey: ["pick-pack-sessions"] });
    },
  });
}
