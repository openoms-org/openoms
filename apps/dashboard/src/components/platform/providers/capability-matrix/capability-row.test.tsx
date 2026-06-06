import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TooltipProvider } from "@/components/ui/tooltip";
import { CapabilityRow } from "@/components/platform/providers/capability-matrix/capability-row";
import { emptyCapability } from "@/components/platform/providers/capability-matrix/capability-helpers";
import type { ProviderCapability } from "@/types/platform";

function cap(overrides: Partial<ProviderCapability>): ProviderCapability {
  return { ...emptyCapability(), ...overrides };
}

function renderRow(capability: ProviderCapability, expanded = false) {
  return render(
    <TooltipProvider>
      <CapabilityRow
        capability={capability}
        expanded={expanded}
        readOnly={false}
        onToggle={vi.fn()}
        onChange={vi.fn()}
        onRemove={vi.fn()}
      />
    </TooltipProvider>,
  );
}

describe("CapabilityRow", () => {
  it("renders the support-state label for the capability", () => {
    renderRow(cap({ capability_key: "supplier.catalog.read", support_status: "supported" }));
    expect(
      screen.getByText("capabilityMatrix.supportStatus.supported"),
    ).toBeInTheDocument();
  });

  it("shows the evidence-required badge for an unknown capability with no evidence", () => {
    renderRow(
      cap({
        capability_key: "supplier.order.create",
        support_status: "unknown",
        notes: "",
      }),
    );
    expect(
      screen.getByText("capabilityMatrix.evidenceRequired"),
    ).toBeInTheDocument();
  });

  it("does not show evidence-required when a supported capability has no notes", () => {
    renderRow(
      cap({
        capability_key: "supplier.catalog.read",
        support_status: "supported",
        notes: "",
      }),
    );
    expect(
      screen.queryByText("capabilityMatrix.evidenceRequired"),
    ).not.toBeInTheDocument();
  });

  it("shows the customer-hidden indicator for an unknown capability when expanded", () => {
    renderRow(
      cap({ capability_key: "supplier.x.y", support_status: "unknown" }),
      true,
    );
    expect(
      screen.getByText("capabilityMatrix.visibilityHidden"),
    ).toBeInTheDocument();
  });

  it("shows the customer-shown indicator for a supported capability when expanded", () => {
    renderRow(
      cap({ capability_key: "supplier.x.y", support_status: "supported" }),
      true,
    );
    expect(
      screen.getByText("capabilityMatrix.visibilityShown"),
    ).toBeInTheDocument();
  });
});
