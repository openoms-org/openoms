import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MappingRow } from "@/components/platform/providers/status-mapping/mapping-row";
import {
  computeMappingWarnings,
  emptyMapping,
} from "@/components/platform/providers/status-mapping/status-mapping-helpers";
import type { ProviderStatusMapping, StatusDomain } from "@/types/platform";

function mapping(
  domain: StatusDomain,
  overrides: Partial<ProviderStatusMapping>,
): ProviderStatusMapping {
  return { ...emptyMapping(domain), ...overrides };
}

function renderRow(m: ProviderStatusMapping) {
  const warnings = computeMappingWarnings([m]);
  return render(
    <MappingRow
      mapping={m}
      warnings={warnings}
      readOnly={false}
      onChange={vi.fn()}
      onRemove={vi.fn()}
    />,
  );
}

describe("MappingRow warnings", () => {
  it("renders the terminal low-confidence warning", () => {
    renderRow(
      mapping("order", {
        raw_status: "done",
        canonical_status: "completed",
        is_terminal: true,
        confidence: "low",
      }),
    );
    expect(
      screen.getByText("statusMapping.warning.terminalLowConfidence"),
    ).toBeInTheDocument();
    expect(screen.getByText("statusMapping.blocksAutomation")).toBeInTheDocument();
  });

  it("renders the shipment-overwrites-order warning", () => {
    renderRow(
      mapping("shipment", {
        raw_status: "delivered",
        canonical_status: "completed",
        confidence: "high",
      }),
    );
    expect(
      screen.getByText("statusMapping.warning.shipmentOverwritesOrder"),
    ).toBeInTheDocument();
  });

  it("renders the unknown-needs-gap warning for an empty canonical status", () => {
    renderRow(mapping("order", { raw_status: "weird", canonical_status: "" }));
    expect(
      screen.getByText("statusMapping.warning.unknownNeedsGap"),
    ).toBeInTheDocument();
  });

  it("renders no warnings for a clean high-confidence mapping", () => {
    renderRow(
      mapping("shipment", {
        raw_status: "sent",
        canonical_status: "shipped",
        confidence: "high",
      }),
    );
    expect(
      screen.queryByText("statusMapping.warning.shipmentOverwritesOrder"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("statusMapping.blocksAutomation"),
    ).not.toBeInTheDocument();
  });
});
