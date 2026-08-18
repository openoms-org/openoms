import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ShipmentStatusActions } from "@/components/shipments/shipment-status-actions";

describe("ShipmentStatusActions", () => {
  it("spins the clicked transition while the group is pending", async () => {
    const onTransition = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(
      <ShipmentStatusActions currentStatus="created" onTransition={onTransition} />
    );

    await user.click(screen.getByRole("button", { name: "label_ready" }));
    expect(onTransition).toHaveBeenCalledWith("label_ready");

    rerender(
      <ShipmentStatusActions
        currentStatus="created"
        onTransition={onTransition}
        isPending
      />
    );

    const labelReady = screen.getByRole("button", { name: "label_ready" });
    expect(labelReady).toBeDisabled();
    expect(labelReady.querySelector(".animate-spin")).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "failed" }).querySelector(".animate-spin")
    ).toBeNull();
  });
});
