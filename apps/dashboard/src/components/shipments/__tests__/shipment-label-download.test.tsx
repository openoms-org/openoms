import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ShipmentLabelDownload } from "@/components/shipments/shipment-label-download";

const downloadShipmentLabel = vi.fn();

vi.mock("@/hooks/use-shipments", () => ({
  downloadShipmentLabel: (...args: unknown[]) => downloadShipmentLabel(...args),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

const SHIPMENT_ID = "a8326d95-1111-2222-3333-444444444444";
const RAW_LABEL_URL =
  "https://api.openoms.org/uploads/tenant-1/a8326d95-label.pdf";

describe("ShipmentLabelDownload", () => {
  beforeEach(() => {
    downloadShipmentLabel.mockReset();
    downloadShipmentLabel.mockResolvedValue(undefined);
  });

  it("is a button, not a bare href to the API uploads URL", () => {
    render(
      <ShipmentLabelDownload shipmentId={SHIPMENT_ID}>
        Pobierz etykietę
      </ShipmentLabelDownload>
    );

    const control = screen.getByRole("button", { name: "Pobierz etykietę" });
    expect(control).not.toHaveAttribute("href");
    expect(control.closest("a")).toBeNull();
    expect(document.querySelector(`a[href="${RAW_LABEL_URL}"]`)).toBeNull();
    expect(
      document.querySelector('a[href*="api.openoms.org/uploads"]')
    ).toBeNull();
  });

  it("downloads the label through the authenticated API client on click", async () => {
    const user = userEvent.setup();
    render(
      <ShipmentLabelDownload shipmentId={SHIPMENT_ID}>
        Pobierz etykietę
      </ShipmentLabelDownload>
    );

    await user.click(screen.getByRole("button", { name: "Pobierz etykietę" }));

    expect(downloadShipmentLabel).toHaveBeenCalledTimes(1);
    expect(downloadShipmentLabel).toHaveBeenCalledWith(SHIPMENT_ID);
  });
});
