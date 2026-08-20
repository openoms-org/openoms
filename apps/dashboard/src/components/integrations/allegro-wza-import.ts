import { ApiClientError } from "@/lib/api-client";

export type ImportedWzAShipment = {
  id: string;
  waybill?: string;
  allegro_shipment_id?: string;
  provider: string;
  label_ready: boolean;
  created: boolean;
};

export type WzAImportDialogError =
  | { kind: "empty" }
  | { kind: "request"; message: string };

// Keep import failures on the dialog. A 3s toast behind the overlay looked
// like a successful no-op on the live locker order.
export function wzaImportDialogError(input: {
  error?: unknown;
  shipments?: ImportedWzAShipment[];
}): WzAImportDialogError | null {
  if (input.error instanceof ApiClientError && (input.error.status === 409 || input.error.status === 404)) {
    return { kind: "empty" };
  }
  if (input.error instanceof Error && input.error.message) {
    return { kind: "request", message: input.error.message };
  }
  if (input.error) {
    return { kind: "empty" };
  }
  if (input.shipments && input.shipments.length === 0) {
    return { kind: "empty" };
  }
  return null;
}

export type WzALabelDownloadTarget =
  | { kind: "oms"; shipmentId: string }
  | { kind: "allegro"; shipmentId: string }
  | { kind: "none" };

// Prefer the logged-in OMS label download. Fall back to the authenticated
// Allegro label route when the PDF was not stored locally. Never return a
// raw upload URL.
export function wzaLabelDownloadTarget(shipment: {
  id?: string;
  allegro_shipment_id?: string;
  label_ready?: boolean;
}): WzALabelDownloadTarget {
  if (shipment.label_ready && shipment.id) {
    return { kind: "oms", shipmentId: shipment.id };
  }
  if (shipment.allegro_shipment_id) {
    return { kind: "allegro", shipmentId: shipment.allegro_shipment_id };
  }
  return { kind: "none" };
}
