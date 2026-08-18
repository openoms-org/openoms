import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OrderStatusActions } from "@/components/orders/order-status-actions";

vi.mock("@/hooks/use-order-statuses", () => ({
  useOrderStatuses: () => ({ data: undefined }),
  statusesToMap: () => ({}),
}));

describe("OrderStatusActions", () => {
  it("spins the clicked transition while the group is pending", async () => {
    const onTransition = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(
      <OrderStatusActions currentStatus="new" onTransition={onTransition} />
    );

    await user.click(screen.getByRole("button", { name: "confirmed" }));
    expect(onTransition).toHaveBeenCalledWith("confirmed", false);

    rerender(
      <OrderStatusActions
        currentStatus="new"
        onTransition={onTransition}
        isPending
      />
    );

    const confirmed = screen.getByRole("button", { name: "confirmed" });
    expect(confirmed).toBeDisabled();
    expect(confirmed.querySelector(".animate-spin")).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "on_hold" }).querySelector(".animate-spin")
    ).toBeNull();
  });
});
