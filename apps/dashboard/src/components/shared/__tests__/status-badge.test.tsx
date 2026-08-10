import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "@/components/shared/status-badge";
import {
  ORDER_STATUSES,
  SHIPMENT_STATUSES,
  RETURN_STATUSES,
  DROPSHIP_STATUSES,
  LOYALTY_STATUSES,
  STOCKTAKE_STATUSES,
  RECURRING_ORDER_STATUSES,
  WAREHOUSE_DOCUMENT_STATUSES,
  PICK_PACK_STATUSES,
  REPRICING_RULE_STATUSES,
} from "@/lib/constants";

describe("StatusBadge", () => {
  it("renders the label text for a known status", () => {
    render(<StatusBadge status="new" statusMap={ORDER_STATUSES} />);
    expect(screen.getByText("new")).toBeInTheDocument();
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
    expect(screen.getByText("in_transit")).toBeInTheDocument();
  });

  it("renders correct label for return statuses", () => {
    render(<StatusBadge status="approved" statusMap={RETURN_STATUSES} />);
    expect(screen.getByText("approved")).toBeInTheDocument();
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

  it("renders semantic warning tone without a legacy status map", () => {
    render(<StatusBadge status="warning" label="Needs review" tone="warning" />);
    const badge = screen.getByText("Needs review");
    expect(badge).toHaveClass("bg-amber-50");
    expect(badge).toHaveClass("text-amber-800");
  });

  it("keeps legacy statusMap classes when tone is not provided", () => {
    const { container } = render(<StatusBadge status="new" statusMap={ORDER_STATUSES} />);
    const badge = container.querySelector("span");
    expect(badge).toHaveClass("bg-blue-100");
    expect(badge).toHaveClass("text-blue-800");
  });
});

// Families the dashboard pages delegate to StatusBadge instead of a local colour map
// (openoms-dev-7sl). Every status of every family must render as a coloured badge.
describe("StatusBadge status families", () => {
  const families: Array<[string, string, Record<string, { label: string; color: string }>]> = [
    ["dropship", "dropship", DROPSHIP_STATUSES],
    ["loyalty", "loyalty", LOYALTY_STATUSES],
    ["stocktake", "stocktake", STOCKTAKE_STATUSES],
    ["recurringOrder", "recurringOrder", RECURRING_ORDER_STATUSES],
    ["warehouseDocument", "warehouseDocument", WAREHOUSE_DOCUMENT_STATUSES],
    ["pickPack", "pickPack", PICK_PACK_STATUSES],
    ["repricingRule", "repricingRule", REPRICING_RULE_STATUSES],
  ];

  it.each(families)("renders every %s status as a coloured badge", (_name, prefix, family) => {
    for (const [status, { color }] of Object.entries(family)) {
      const { container, unmount } = render(
        <StatusBadge status={status} statusMap={family} translationPrefix={prefix} />,
      );
      const badge = container.querySelector("span");
      expect(badge).not.toBeNull();
      for (const cls of color.split(" ")) {
        expect(badge).toHaveClass(cls);
      }
      unmount();
    }
  });
});
