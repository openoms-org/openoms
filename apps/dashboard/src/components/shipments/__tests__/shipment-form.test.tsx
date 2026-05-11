import { describe, expect, it, vi } from "vitest";
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
});
