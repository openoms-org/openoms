import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  Return,
  ListResponse,
  ReturnListParams,
  CreateReturnRequest,
  UpdateReturnRequest,
  ReturnStatusRequest,
} from "@/types/api";

export function useReturns(params: ReturnListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit) query.set("limit", String(params.limit));
  if (params.offset) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.order_id) query.set("order_id", params.order_id);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["returns", params],
    queryFn: () =>
      apiClient<ListResponse<Return>>(`/v1/returns${qs ? `?${qs}` : ""}`),
  });
}

export function useReturn(id: string) {
  return useQuery({
    queryKey: ["returns", id],
    queryFn: () => apiClient<Return>(`/v1/returns/${id}`),
    enabled: !!id,
  });
}

export function useCreateReturn() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateReturnRequest) =>
      apiClient<Return>("/v1/returns", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["returns"] });
    },
  });
}

export function useUpdateReturn(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateReturnRequest) =>
      apiClient<Return>(`/v1/returns/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["returns"] });
    },
  });
}

export function useTransitionReturnStatus(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: ReturnStatusRequest) =>
      apiClient<Return>(`/v1/returns/${id}/status`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["returns"] });
      queryClient.invalidateQueries({ queryKey: ["returns", id] });
    },
  });
}

export function useDeleteReturn() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/returns/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["returns"] });
    },
  });
}
