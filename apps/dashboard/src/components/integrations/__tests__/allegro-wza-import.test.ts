import { describe, expect, it } from "vitest";
import { ApiClientError } from "@/lib/api-client";
import { wzaImportDialogError, wzaLabelDownloadTarget } from "../allegro-wza-import";

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

describe("wzaImportDialogError", () => {
  it("surfaces a 409 empty checkout as a dialog error, not a silent success", () => {
    expect(
      wzaImportDialogError({
        error: new ApiClientError(
          409,
          "wysyłam z Allegro has no existing shipment for this checkout"
        ),
      })
    ).toEqual({ kind: "empty" });
  });

  it("surfaces a 200 with no imported rows as a dialog error", () => {
    expect(wzaImportDialogError({ shipments: [] })).toEqual({ kind: "empty" });
  });

  it("keeps other request failures visible in the dialog", () => {
    expect(
      wzaImportDialogError({
        error: new ApiClientError(500, "Failed to store imported shipment"),
      })
    ).toEqual({
      kind: "request",
      message: "Failed to store imported shipment",
    });
  });

  it("keeps the exact store/audit inet cause on the dialog", () => {
    const message =
      'Failed to store imported shipment: audit log: ERROR: invalid input syntax for type inet: "allegro-wza-import" (SQLSTATE 22P02)';
    expect(
      wzaImportDialogError({
        error: new ApiClientError(500, message),
      })
    ).toEqual({ kind: "request", message });
  });

  it("is silent only when a shipment was actually imported", () => {
    expect(
      wzaImportDialogError({
        shipments: [
          {
            id: "oms-1",
            provider: "allegro",
            label_ready: false,
            created: true,
            waybill: "605500867604760112200733",
          },
        ],
      })
    ).toBeNull();
  });
});
