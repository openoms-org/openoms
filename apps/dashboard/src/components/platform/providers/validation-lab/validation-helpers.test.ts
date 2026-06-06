import { describe, expect, it } from "vitest";
import {
  groupProbesBySafety,
  hasDestructiveProbe,
  isRunOpen,
  probeSafetyClass,
  resultStatusTone,
  safetyClassTone,
  verdictFromResults,
  verdictTone,
} from "@/components/platform/providers/validation-lab/validation-helpers";
import type {
  ProbeType,
  ProviderValidationProbe,
  ProviderValidationResult,
} from "@/types/platform";

function probe(
  type: ProbeType,
  overrides: Partial<ProviderValidationProbe> = {},
): ProviderValidationProbe {
  return {
    id: `${type}-${overrides.label ?? ""}`,
    provider_version_id: "v",
    probe_type: type,
    label: overrides.label ?? type,
    destructive: false,
    required: false,
    created_at: "2026-06-06T00:00:00Z",
    updated_at: "2026-06-06T00:00:00Z",
    ...overrides,
  };
}

function result(
  status: ProviderValidationResult["status"],
): ProviderValidationResult {
  return {
    id: `r-${status}-${Math.random()}`,
    run_id: "run",
    probe_type: "auth_check",
    label: "l",
    status,
    observation: "",
    payload_hash: "",
    findings: "",
    created_at: "2026-06-06T00:00:00Z",
  };
}

describe("probeSafetyClass", () => {
  it("classifies a read-only probe", () => {
    expect(probeSafetyClass(probe("auth_check"), "sandbox")).toBe("read_only");
    expect(probeSafetyClass(probe("sample_catalog_read"), "production")).toBe(
      "read_only",
    );
  });

  it("classifies a sandbox write vs production write by environment", () => {
    expect(probeSafetyClass(probe("sandbox_order_create"), "sandbox")).toBe(
      "sandbox_write",
    );
    expect(probeSafetyClass(probe("sandbox_order_create"), "production")).toBe(
      "production_write",
    );
  });

  it("destructive flag overrides type and environment", () => {
    expect(
      probeSafetyClass(probe("auth_check", { destructive: true }), "sandbox"),
    ).toBe("destructive");
    expect(
      probeSafetyClass(
        probe("sandbox_order_create", { destructive: true }),
        "production",
      ),
    ).toBe("destructive");
  });
});

describe("groupProbesBySafety", () => {
  it("orders groups safest-first and separates destructive", () => {
    const probes = [
      probe("sandbox_order_create", { label: "write" }),
      probe("auth_check", { label: "read" }),
      probe("order_status_read", { label: "destroy", destructive: true }),
    ];
    const grouped = groupProbesBySafety(probes, "sandbox");
    expect(grouped.map((g) => g.safety)).toEqual([
      "read_only",
      "sandbox_write",
      "destructive",
    ]);
  });
});

describe("hasDestructiveProbe", () => {
  it("detects any destructive probe", () => {
    expect(hasDestructiveProbe([probe("auth_check")])).toBe(false);
    expect(
      hasDestructiveProbe([probe("auth_check", { destructive: true })]),
    ).toBe(true);
  });
});

describe("verdictFromResults", () => {
  it("returns error when there are no results", () => {
    expect(verdictFromResults([])).toBe("error");
  });
  it("any error wins over failed/passed", () => {
    expect(verdictFromResults([result("passed"), result("error")])).toBe("error");
  });
  it("any failed (no errors) yields failed", () => {
    expect(verdictFromResults([result("passed"), result("failed")])).toBe("failed");
  });
  it("all passed yields passed", () => {
    expect(verdictFromResults([result("passed"), result("skipped")])).toBe(
      "passed",
    );
  });
});

describe("tones and run state", () => {
  it("maps tones for verdict and result status", () => {
    expect(verdictTone("passed")).toBe("success");
    expect(verdictTone("failed")).toBe("destructive");
    expect(resultStatusTone("skipped")).toBe("secondary");
    expect(safetyClassTone("destructive")).toBe("destructive");
    expect(safetyClassTone("read_only")).toBe("secondary");
  });
  it("isRunOpen only for pending verdict", () => {
    expect(isRunOpen({ verdict: "pending" })).toBe(true);
    expect(isRunOpen({ verdict: "passed" })).toBe(false);
  });
});
