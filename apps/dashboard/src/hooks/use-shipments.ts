import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import { downloadBlob } from "@/lib/download";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Shipment,
  ShipmentListParams,
  CreateShipmentRequest,
  UpdateShipmentRequest,
  StatusTransitionRequest,
  GenerateLabelRequest,
  BatchLabelsRequest,
  TrackingEvent,
  CreateDispatchOrderRequest,
  DispatchOrderResponse,
} from "@/types/api";

const shipmentHooks = createCrudHooks<
  Shipment,
  CreateShipmentRequest,
  UpdateShipmentRequest,
  ShipmentListParams
>({
  resourceKey: "shipments",
  basePath: "/v1/shipments",
});

export const useShipments = shipmentHooks.useList;
export const useShipment = shipmentHooks.useGet;
export const useCreateShipment = shipmentHooks.useCreate;
export const useUpdateShipment = shipmentHooks.useUpdate;
export const useDeleteShipment = shipmentHooks.useDelete;

// Returns all shipments for a specific order, sorted by package_number.
export function useOrderShipments(orderId: string) {
  return useQuery({
    queryKey: ["orders", orderId, "shipments"],
    queryFn: () =>
      apiClient<Shipment[]>(`/v1/orders/${orderId}/shipments`),
    enabled: !!orderId,
  });
}

export function useCreateOrderShipment(orderId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: Omit<CreateShipmentRequest, "order_id">) =>
      apiClient<Shipment>(`/v1/orders/${orderId}/shipments`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders", orderId, "shipments"] });
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

export function useShipmentTracking(id: string, enabled = true) {
  return useQuery({
    queryKey: ["shipments", id, "tracking"],
    queryFn: () =>
      apiClient<TrackingEvent[]>(`/v1/shipments/${id}/tracking`),
    enabled: !!id && enabled,
    refetchInterval: 60_000,
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
      downloadBlob(blob, "labels.zip");
    },
  });
}

export function useCreateDispatchOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateDispatchOrderRequest) =>
      apiClient<DispatchOrderResponse>("/v1/shipments/dispatch-order", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["shipments"] });
    },
  });
}
