import { describe, expect, it } from "vitest";
import {
  computeMappingWarnings,
  emptyMapping,
  isTerminalCanonical,
  isUnknownMapping,
  warningsByRow,
} from "@/components/platform/providers/status-mapping/status-mapping-helpers";
import type {
  ProviderIntegrationGap,
  ProviderStatusMapping,
  StatusDomain,
} from "@/types/platform";

function mapping(
  domain: StatusDomain,
  overrides: Partial<ProviderStatusMapping>,
): ProviderStatusMapping {
  return { ...emptyMapping(domain), ...overrides };
}

function gap(overrides: Partial<ProviderIntegrationGap>): ProviderIntegrationGap {
  return {
    id: "g1",
    provider_version_id: "v1",
    gap_type: "missing_status_mapping",
    severity: "action_required",
    status: "open",
    description: "",
    created_at: "",
    updated_at: "",
    ...overrides,
  };
}

describe("isUnknownMapping", () => {
  it("treats empty and 'unknown' canonical targets as unknown", () => {
    expect(isUnknownMapping(mapping("order", { canonical_status: "" }))).toBe(true);
    expect(
      isUnknownMapping(mapping("order", { canonical_status: "Unknown" })),
    ).toBe(true);
    expect(
      isUnknownMapping(mapping("order", { canonical_status: "shipped" })),
    ).toBe(false);
  });
});

describe("isTerminalCanonical", () => {
  it("detects the terminal flag or a terminal canonical status", () => {
    expect(
      isTerminalCanonical(mapping("order", { is_terminal: true })),
    ).toBe(true);
    expect(
      isTerminalCanonical(mapping("order", { canonical_status: "delivered" })),
    ).toBe(true);
    expect(
      isTerminalCanonical(mapping("order", { canonical_status: "processing" })),
    ).toBe(false);
  });
});

describe("computeMappingWarnings", () => {
  it("warns on a terminal status with low confidence", () => {
    const warnings = computeMappingWarnings([
      mapping("order", {
        raw_status: "done",
        canonical_status: "completed",
        is_terminal: true,
        confidence: "low",
      }),
    ]);
    expect(warnings.map((w) => w.code)).toContain("terminalLowConfidence");
    expect(warnings[0].blocking).toBe(true);
  });

  it("does NOT warn on a terminal status with high confidence", () => {
    const warnings = computeMappingWarnings([
      mapping("order", {
        raw_status: "done",
        canonical_status: "completed",
        is_terminal: true,
        confidence: "high",
      }),
    ]);
    expect(warnings.map((w) => w.code)).not.toContain("terminalLowConfidence");
  });

  it("warns when a shipment status would overwrite a commercial order status", () => {
    const warnings = computeMappingWarnings([
      mapping("shipment", {
        raw_status: "delivered",
        canonical_status: "completed",
        confidence: "high",
      }),
    ]);
    expect(warnings.map((w) => w.code)).toContain("shipmentOverwritesOrder");
  });

  it("does NOT warn when a shipment status maps to a shipment-level status", () => {
    const warnings = computeMappingWarnings([
      mapping("shipment", {
        raw_status: "sent",
        canonical_status: "shipped",
        confidence: "high",
      }),
    ]);
    expect(warnings.map((w) => w.code)).not.toContain("shipmentOverwritesOrder");
  });

  it("flags an unknown mapping as needing a gap and never as success", () => {
    const warnings = computeMappingWarnings([
      mapping("order", { raw_status: "weird_value", canonical_status: "" }),
    ]);
    expect(warnings.map((w) => w.code)).toContain("unknownNeedsGap");
    expect(warnings[0].blocking).toBe(true);
  });

  it("suppresses the unknown warning once an open missing_status_mapping gap exists", () => {
    const warnings = computeMappingWarnings(
      [mapping("order", { raw_status: "weird_value", canonical_status: "" })],
      [gap({ status: "open" })],
    );
    expect(warnings.map((w) => w.code)).not.toContain("unknownNeedsGap");
  });

  it("does not evaluate terminal/shipment rules for an unknown mapping", () => {
    // An unknown (empty canonical) row must not also be treated as a usable
    // mapping — only the unknownNeedsGap warning applies.
    const warnings = computeMappingWarnings([
      mapping("shipment", {
        raw_status: "weird",
        canonical_status: "",
        is_terminal: true,
        confidence: "low",
      }),
    ]);
    expect(warnings.map((w) => w.code)).toEqual(["unknownNeedsGap"]);
  });
});

describe("warningsByRow", () => {
  it("indexes warnings by domain + raw status", () => {
    const warnings = computeMappingWarnings([
      mapping("shipment", {
        raw_status: "delivered",
        canonical_status: "completed",
        is_terminal: true,
        confidence: "low",
      }),
    ]);
    const index = warningsByRow(warnings);
    const row = index.get("shipment::delivered");
    expect(row?.map((w) => w.code).sort()).toEqual([
      "shipmentOverwritesOrder",
      "terminalLowConfidence",
    ]);
  });
});
