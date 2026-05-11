import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ShipmentForm } from "@/components/shipments/shipment-form";

vi.mock("next-intl", () => ({
  useTranslations: vi.fn(() => (key: string) => key),
}));

vi.mock("@/components/shared/order-search-combobox", () => ({
  OrderSearchCombobox: ({
    value,
    onValueChange,
    disabled,
  }: {
    value: string;
    onValueChange: (value: string) => void;
    disabled?: boolean;
  }) => (
    <input
      aria-label="order"
      disabled={disabled}
      onChange={(event) => onValueChange(event.target.value)}
      value={value}
    />
  ),
}));

vi.mock("@/components/shared/paczkomat-selector", () => ({
  PaczkomatSelector: () => <button type="button">Paczkomat</button>,
}));

describe("ShipmentForm", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("only exposes client-ready shipment providers by default", async () => {
    const user = userEvent.setup();

    render(<ShipmentForm onSubmit={vi.fn()} />);

    await user.click(screen.getByRole("combobox"));

    expect(screen.getByRole("option", { name: "INPOST" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "MANUAL" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "DHL" })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "DPD" })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "GLS" })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "UPS" })).not.toBeInTheDocument();
  });

  it("exposes all selectable shipment providers in full dashboard mode", async () => {
    const user = userEvent.setup();
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE", "full");

    render(<ShipmentForm onSubmit={vi.fn()} />);

    await user.click(screen.getByRole("combobox"));

    for (const provider of [
      "INPOST",
      "DHL",
      "DPD",
      "GLS",
      "UPS",
      "POCZTA_POLSKA",
      "ORLEN_PACZKA",
      "FEDEX",
      "MANUAL",
    ]) {
      expect(screen.getByRole("option", { name: provider })).toBeInTheDocument();
    }
  });

  it("keeps an existing non-client-ready provider visible while editing", async () => {
    const user = userEvent.setup();

    render(
      <ShipmentForm
        onSubmit={vi.fn()}
        shipment={{
          id: "shipment-1",
          order_id: "order-1",
          tenant_id: "tenant-1",
          provider: "dhl",
          tracking_number: "",
          label_url: "",
          status: "created",
          carrier_data: {},
          created_at: "2026-05-11T00:00:00Z",
          updated_at: "2026-05-11T00:00:00Z",
        }}
      />,
    );

    await user.click(screen.getByRole("combobox"));

    expect(screen.getByRole("option", { name: "DHL" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "INPOST" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "DPD" })).not.toBeInTheDocument();
  });
});
