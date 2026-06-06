import { describe, expect, it } from "vitest";
import {
  capabilityDomain,
  emptyCapability,
  groupByDomain,
  hasProbeReference,
  isCustomerVisible,
  isEvidenceMissing,
  isEvidenceRequired,
  supportStatusTone,
} from "@/components/platform/providers/capability-matrix/capability-helpers";
import type { ProviderCapability, SupportStatus } from "@/types/platform";

function cap(overrides: Partial<ProviderCapability>): ProviderCapability {
  return { ...emptyCapability(), ...overrides };
}

describe("supportStatusTone", () => {
  it("maps supported to success and unknown to destructive", () => {
    expect(supportStatusTone("supported")).toBe("success");
    expect(supportStatusTone("unknown")).toBe("destructive");
    expect(supportStatusTone("degraded")).toBe("warning");
  });
});

describe("isEvidenceRequired / isEvidenceMissing", () => {
  it("requires evidence for unknown and degraded", () => {
    expect(isEvidenceRequired(cap({ support_status: "unknown" }))).toBe(true);
    expect(isEvidenceRequired(cap({ support_status: "degraded" }))).toBe(true);
    expect(isEvidenceRequired(cap({ support_status: "supported" }))).toBe(false);
  });

  it("reports evidence missing when required and notes empty", () => {
    expect(
      isEvidenceMissing(cap({ support_status: "unknown", notes: "" })),
    ).toBe(true);
    expect(
      isEvidenceMissing(cap({ support_status: "unknown", notes: "https://x/y" })),
    ).toBe(false);
    // Supported never demands evidence, so it is never "missing".
    expect(
      isEvidenceMissing(cap({ support_status: "supported", notes: "" })),
    ).toBe(false);
  });
});

describe("hasProbeReference", () => {
  it("detects a probe/run reference in evidence text", () => {
    expect(hasProbeReference(cap({ notes: "see validation run #12" }))).toBe(true);
    expect(hasProbeReference(cap({ notes: "auth.check probe passed" }))).toBe(true);
    expect(hasProbeReference(cap({ notes: "https://docs/page" }))).toBe(false);
  });
});

describe("isCustomerVisible", () => {
  it("hides unknown and unsupported capabilities from customers", () => {
    expect(isCustomerVisible(cap({ support_status: "unknown" }))).toBe(false);
    expect(isCustomerVisible(cap({ support_status: "unsupported" }))).toBe(false);
    expect(isCustomerVisible(cap({ support_status: "supported" }))).toBe(true);
    expect(isCustomerVisible(cap({ support_status: "requires_manual" }))).toBe(true);
  });
});

describe("capabilityDomain / groupByDomain", () => {
  it("derives the leading domain segment", () => {
    expect(capabilityDomain(cap({ capability_key: "supplier.catalog.read" }))).toBe(
      "supplier",
    );
    expect(capabilityDomain(cap({ capability_key: "no_dots" }))).toBe("no_dots");
    expect(capabilityDomain(cap({ capability_key: "" }))).toBe("other");
  });

  it("groups capabilities by domain, sorted", () => {
    const caps = [
      cap({ capability_key: "marketplace.offer.push" }),
      cap({ capability_key: "carrier.label.create" }),
      cap({ capability_key: "carrier.tracking.read" }),
    ];
    const grouped = groupByDomain(caps);
    expect(grouped.map((g) => g.domain)).toEqual(["carrier", "marketplace"]);
    expect(grouped[0].capabilities).toHaveLength(2);
  });
});

describe("emptyCapability", () => {
  it("defaults to an unknown support status", () => {
    const status: SupportStatus = emptyCapability().support_status;
    expect(status).toBe("unknown");
  });
});
