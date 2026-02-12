import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import type {
  Shipment,
  ListResponse,
  ShipmentListParams,
  CreateShipmentRequest,
  UpdateShipmentRequest,
  StatusTransitionRequest,
  GenerateLabelRequest,
  BatchLabelsRequest,
} from "@/types/api";

export function useShipments(params: ShipmentListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.provider) query.set("provider", params.provider);
  if (params.order_id) query.set("order_id", params.order_id);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["shipments", params],
    queryFn: () =>
      apiClient<ListResponse<Shipment>>(`/v1/shipments${qs ? `?${qs}` : ""}`),
  });
}

export function useShipment(id: string) {
  return useQuery({
    queryKey: ["shipments", id],
    queryFn: () => apiClient<Shipment>(`/v1/shipments/${id}`),
    enabled: !!id,
  });
}

export function useCreateShipment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateShipmentRequest) =>
      apiClient<Shipment>("/v1/shipments", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["shipments"] });
    },
  });
}

export function useUpdateShipment(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateShipmentRequest) =>
      apiClient<Shipment>(`/v1/shipments/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["shipments"] });
    },
  });
}

export function useDeleteShipment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/shipments/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["shipments"] });
    },
  });
}

export function useTransitionShipmentStatus(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: StatusTransitionRequest) =>
      apiClient<Shipment>(`/v1/shipments/${id}/status`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["shipments"] });
    },
  });
}

export function useGenerateLabel(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: GenerateLabelRequest) =>
      apiClient<Shipment>(`/v1/shipments/${id}/label`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["shipments"] });
    },
  });
}

export function useBatchLabels() {
  return useMutation({
    mutationFn: async (data: BatchLabelsRequest) => {
      const res = await apiFetch("/v1/shipments/batch-labels", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "labels.zip";
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    },
  });
}
