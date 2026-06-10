import { describe, expect, it } from "vitest";
import {
  canViewSensitiveEvidence,
  evidenceTimeline,
  hasRedactableContent,
  redactObservation,
  shortHash,
} from "@/components/platform/providers/shared/evidence-helpers";
import type { ProviderValidationResult } from "@/types/platform";

describe("canViewSensitiveEvidence", () => {
  it("requires the providers:secrets permission", () => {
    expect(canViewSensitiveEvidence(["providers:read"])).toBe(false);
    expect(canViewSensitiveEvidence(["providers:read", "providers:secrets"])).toBe(
      true,
    );
    expect(canViewSensitiveEvidence(undefined)).toBe(false);
    expect(canViewSensitiveEvidence([])).toBe(false);
  });
});

describe("redactObservation", () => {
  it("masks bearer tokens and key=value secrets when not permitted to reveal", () => {
    const raw = "Authorization: Bearer abcdef1234567890 password=hunter2";
    const masked = redactObservation(raw, {
      canViewSensitive: false,
      reveal: false,
    });
    expect(masked).not.toContain("abcdef1234567890");
    expect(masked).not.toContain("hunter2");
    expect(masked).toContain("[redacted]");
  });

  it("masks email PII", () => {
    const masked = redactObservation("contact alice@example.com now", {
      canViewSensitive: false,
      reveal: false,
    });
    expect(masked).not.toContain("alice@example.com");
    expect(masked).toContain("[redacted-email]");
  });

  it("keeps masking even for a permitted viewer until they explicitly reveal", () => {
    const raw = "secret=topsecretvalue";
    expect(
      redactObservation(raw, { canViewSensitive: true, reveal: false }),
    ).toContain("[redacted]");
    // Only an explicit reveal by a permitted viewer shows the recorded text.
    expect(redactObservation(raw, { canViewSensitive: true, reveal: true })).toBe(
      raw,
    );
  });

  it("does not reveal for an unpermitted viewer even if reveal is true", () => {
    const raw = "secret=topsecretvalue";
    expect(
      redactObservation(raw, { canViewSensitive: false, reveal: true }),
    ).toContain("[redacted]");
  });

  it("leaves benign observations untouched", () => {
    const raw = "connectivity ok, 200 in 120ms";
    expect(redactObservation(raw, { canViewSensitive: false, reveal: false })).toBe(
      raw,
    );
    expect(hasRedactableContent(raw)).toBe(false);
  });
});

describe("shortHash", () => {
  it("truncates long hashes and leaves short ones", () => {
    expect(shortHash("abc")).toBe("abc");
    expect(shortHash("0123456789abcdef0123")).toBe("0123456789ab…");
  });
});

describe("evidenceTimeline", () => {
  it("sorts results newest-first and tolerates undefined", () => {
    expect(evidenceTimeline(undefined)).toEqual([]);
    const results: ProviderValidationResult[] = [
      mkResult("a", "2026-06-01T00:00:00Z"),
      mkResult("b", "2026-06-05T00:00:00Z"),
    ];
    expect(evidenceTimeline(results).map((r) => r.id)).toEqual(["b", "a"]);
  });
});

function mkResult(id: string, createdAt: string): ProviderValidationResult {
  return {
    id,
    run_id: "run",
    probe_type: "auth_check",
    label: id,
    status: "passed",
    observation: "",
    payload_hash: "",
    findings: "",
    created_at: createdAt,
  };
}
