import { describe, expect, it } from "vitest";
import { wzaLabelDownloadTarget } from "../allegro-wza-import";

describe("wzaLabelDownloadTarget", () => {
  it("uses the logged-in OMS shipment label route when a PDF was stored", () => {
    expect(
      wzaLabelDownloadTarget({
        id: "oms-shipment-1",
        allegro_shipment_id: "cb92efe4-1b2f-4cac-9e35-da69b0000001",
        label_ready: true,
      })
    ).toEqual({ kind: "oms", shipmentId: "oms-shipment-1" });
  });

  it("falls back to the authenticated Allegro label route when only the WzA id is known", () => {
    expect(
      wzaLabelDownloadTarget({
        id: "oms-shipment-1",
        allegro_shipment_id: "cb92efe4-1b2f-4cac-9e35-da69b0000001",
        label_ready: false,
      })
    ).toEqual({
      kind: "allegro",
      shipmentId: "cb92efe4-1b2f-4cac-9e35-da69b0000001",
    });
  });

  it("does not invent a download target when Allegro has no WzA id", () => {
    expect(
      wzaLabelDownloadTarget({
        id: "oms-shipment-1",
        label_ready: false,
      })
    ).toEqual({ kind: "none" });
  });
});
