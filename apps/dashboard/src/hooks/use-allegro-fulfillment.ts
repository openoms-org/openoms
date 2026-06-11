"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import { downloadBlob } from "@/lib/download";

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
  // Optional: OpenOMS order UUID or Allegro external order ID to link the shipment
  order_id?: string;
}

interface AllegroShipmentAddress {
  name?: string;
  company?: string;
  street: string;
  city: string;
  zipCode: string;
  countryCode: string;
  phone?: string;
  email?: string;
}

interface AllegroShipmentPackage {
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

// --- Pickup Types ---

export interface AllegroPickupTimeWindow {
  from: string;
  to: string;
}

export interface AllegroPickupProposal {
  date: string;
  timeWindows: AllegroPickupTimeWindow[];
}

// --- Hooks ---

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

// Shipment management ("Wysylam z Allegro")

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

export async function downloadAllegroLabel(shipmentId: string) {
  const res = await apiFetch(
    `/v1/integrations/allegro/shipments/${shipmentId}/label`
  );
  const blob = await res.blob();
  downloadBlob(blob, `etykieta-${shipmentId}.pdf`);
}

// Cancel a managed shipment
export function useCancelAllegroShipment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (shipmentId: string) =>
      apiClient<void>(
        `/v1/integrations/allegro/shipments/${shipmentId}`,
        { method: "DELETE" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "shipments"] });
    },
  });
}

// Get pickup proposals
export function useAllegroPickupProposals() {
  return useMutation({
    mutationFn: (req: { deliveryMethodId: string; shipmentIds: string[] }) =>
      apiClient<{ proposals: AllegroPickupProposal[] }>(
        "/v1/integrations/allegro/pickup-proposals",
        {
          method: "POST",
          body: JSON.stringify(req),
        }
      ),
  });
}

// Schedule a courier pickup
export function useScheduleAllegroPickup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      commandId: string;
      pickupDate: string;
      timeWindow: AllegroPickupTimeWindow;
      shipmentIds: string[];
    }) =>
      apiClient<void>("/v1/integrations/allegro/pickups", {
        method: "POST",
        body: JSON.stringify(cmd),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["allegro", "shipments"] });
    },
  });
}

export async function downloadAllegroProtocol(shipmentIds: string[]) {
  const res = await apiFetch("/v1/integrations/allegro/protocol", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ shipment_ids: shipmentIds }),
  });
  const blob = await res.blob();
  downloadBlob(blob, "protokol-allegro.pdf");
}
