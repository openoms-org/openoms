import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "@/components/shared/status-badge";
import { ORDER_STATUSES, SHIPMENT_STATUSES, RETURN_STATUSES } from "@/lib/constants";

describe("StatusBadge", () => {
  it("renders the label text for a known status", () => {
    render(<StatusBadge status="new" statusMap={ORDER_STATUSES} />);
    expect(screen.getByText("Nowe")).toBeInTheDocument();
  });

  it("renders correct color classes for 'new' order status", () => {
    const { container } = render(<StatusBadge status="new" statusMap={ORDER_STATUSES} />);
    const badge = container.querySelector("span");
    expect(badge).toHaveClass("bg-blue-100");
    expect(badge).toHaveClass("text-blue-800");
  });

  it("renders correct color classes for 'delivered' order status", () => {
    const { container } = render(<StatusBadge status="delivered" statusMap={ORDER_STATUSES} />);
    const badge = container.querySelector("span");
    expect(badge).toHaveClass("bg-green-100");
    expect(badge).toHaveClass("text-green-800");
  });

  it("renders correct color classes for 'cancelled' order status", () => {
    const { container } = render(<StatusBadge status="cancelled" statusMap={ORDER_STATUSES} />);
    const badge = container.querySelector("span");
    expect(badge).toHaveClass("bg-red-100");
    expect(badge).toHaveClass("text-red-800");
  });

  it("renders correct label for shipment statuses", () => {
    render(<StatusBadge status="in_transit" statusMap={SHIPMENT_STATUSES} />);
    expect(screen.getByText("W transporcie")).toBeInTheDocument();
  });

  it("renders correct label for return statuses", () => {
    render(<StatusBadge status="approved" statusMap={RETURN_STATUSES} />);
    expect(screen.getByText("Zatwierdzone")).toBeInTheDocument();
  });

  it("falls back to outline Badge with raw status for unknown status", () => {
    render(<StatusBadge status="unknown_status" statusMap={ORDER_STATUSES} />);
    expect(screen.getByText("unknown_status")).toBeInTheDocument();
  });

  it("renders all order statuses correctly", () => {
    const statuses = Object.keys(ORDER_STATUSES);
    for (const status of statuses) {
      const { unmount } = render(<StatusBadge status={status} statusMap={ORDER_STATUSES} />);
      expect(screen.getByText(ORDER_STATUSES[status].label)).toBeInTheDocument();
      unmount();
    }
  });
});
