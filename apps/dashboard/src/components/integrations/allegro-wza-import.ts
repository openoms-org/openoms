export type ImportedWzAShipment = {
  id: string;
  waybill?: string;
  allegro_shipment_id?: string;
  provider: string;
  label_ready: boolean;
  created: boolean;
};

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
