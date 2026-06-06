import { describe, expect, it } from "vitest";
import {
  availableTransitions,
  blockedGates,
  blockingGaps,
  canEmergencyDisable,
  deriveGates,
  evaluatePublishAction,
  lastCompletedRun,
  warningGaps,
  type VersionReadinessInput,
} from "@/components/platform/providers/publication-review/publication-helpers";
import type {
  ProviderCapability,
  ProviderFieldSchema,
  ProviderIntegrationGap,
  ProviderPublicationState,
  ProviderValidationRun,
  ProviderVersion,
} from "@/types/platform";

function version(state: ProviderPublicationState): ProviderVersion {
  return {
    id: "v1",
    version: "1.0.0",
    publication_state: state,
    changelog: "",
    created_at: "2026-06-06T00:00:00Z",
  };
}

function schemaWithFields(): ProviderFieldSchema {
  return {
    id: "s",
    provider_version_id: "v1",
    groups: [
      {
        key: "secret_credentials",
        label: "Secrets",
        fields: [
          {
            key: "token",
            label: "Token",
            type: "password",
            required: true,
            secret: true,
            validation: {},
          },
        ],
      },
    ],
    created_at: "",
    updated_at: "",
  };
}

function cap(status: ProviderCapability["support_status"]): ProviderCapability {
  return {
    capability_key: "marketplace.order.read",
    support_status: status,
    channel: "",
    mode: "",
    freshness: "",
    required_inputs: [],
    provided_outputs: [],
    latency_sla_seconds: null,
    notes: "",
  };
}

function gap(
  severity: ProviderIntegrationGap["severity"],
  status: ProviderIntegrationGap["status"] = "open",
): ProviderIntegrationGap {
  return {
    id: `g-${severity}-${status}-${Math.random()}`,
    provider_version_id: "v1",
    gap_type: "missing_status_mapping",
    severity,
    status,
    description: "x",
    created_at: "",
    updated_at: "",
  };
}

function run(
  verdict: ProviderValidationRun["verdict"],
  withResults = true,
): ProviderValidationRun {
  return {
    id: `run-${verdict}-${Math.random()}`,
    provider_version_id: "v1",
    environment: "sandbox",
    verdict,
    started_at: "2026-06-06T00:00:00Z",
    finished_at: "2026-06-06T01:00:00Z",
    notes: "",
    results: withResults
      ? [
          {
            id: "res1",
            run_id: "run",
            probe_type: "auth_check",
            label: "auth",
            status: verdict === "passed" ? "passed" : "failed",
            observation: "ok",
            payload_hash: "abc",
            findings: "",
            created_at: "2026-06-06T00:30:00Z",
          },
        ]
      : [],
  };
}

function fullyReady(state: ProviderPublicationState): VersionReadinessInput {
  return {
    version: version(state),
    schema: schemaWithFields(),
    capabilities: [cap("supported")],
    gaps: [],
    runs: [run("passed")],
  };
}

describe("gap selectors", () => {
  it("splits blocking vs warning gaps and ignores resolved", () => {
    const gaps = [
      gap("action_required"),
      gap("system_error"),
      gap("warning"),
      gap("info"),
      gap("action_required", "resolved"),
    ];
    expect(blockingGaps(gaps)).toHaveLength(2);
    expect(warningGaps(gaps)).toHaveLength(2);
  });
});

describe("deriveGates", () => {
  it("passes the readiness-driven gates when fully ready", () => {
    const gates = deriveGates(fullyReady("internal_validation"));
    const byId = Object.fromEntries(gates.map((g) => [g.id, g]));
    expect(byId.G2.status).toBe("pass");
    expect(byId.G3.status).toBe("pass");
    expect(byId.G5.status).toBe("pass");
    expect(byId.G6.status).toBe("pass");
    // Human-sign-off gates stay manual, never auto-pass.
    expect(byId.G7.status).toBe("manual");
    expect(byId.G8.status).toBe("manual");
    expect(byId.G9.status).toBe("manual");
    // Every gate is flagged as derived.
    expect(gates.every((g) => g.derived)).toBe(true);
  });

  it("blocks G2 with no schema fields", () => {
    const input = fullyReady("internal_validation");
    input.schema = undefined;
    const g2 = deriveGates(input).find((g) => g.id === "G2")!;
    expect(g2.status).toBe("blocked");
    expect(g2.reasonKey).toBe("g2NoSchema");
  });

  it("blocks G3 with no completed run and again when last run failed", () => {
    const noRun = { ...fullyReady("internal_validation"), runs: [] };
    expect(deriveGates(noRun).find((g) => g.id === "G3")!.status).toBe("blocked");

    const failed = { ...fullyReady("internal_validation"), runs: [run("failed")] };
    expect(deriveGates(failed).find((g) => g.id === "G3")!.reasonKey).toBe(
      "g3LastRunFailed",
    );
  });

  it("blocks G3 when there are open blocking gaps even after a pass", () => {
    const input = {
      ...fullyReady("internal_validation"),
      gaps: [gap("action_required")],
    };
    const g3 = deriveGates(input).find((g) => g.id === "G3")!;
    expect(g3.status).toBe("blocked");
    expect(g3.reasonKey).toBe("g3OpenBlockingGaps");
  });

  it("warns G6 when a capability is unknown", () => {
    const input = { ...fullyReady("internal_validation"), capabilities: [cap("unknown")] };
    expect(deriveGates(input).find((g) => g.id === "G6")!.status).toBe("warn");
  });
});

describe("evaluatePublishAction", () => {
  it("allows exposure-increasing transition only when no blocked gates or blocking gaps", () => {
    const gates = deriveGates(fullyReady("internal_validation"));
    const decision = evaluatePublishAction(
      "internal_validation",
      "private_beta",
      gates,
      [],
    );
    expect(decision.allowed).toBe(true);
    expect(decision.reasonKeys).toEqual([]);
  });

  it("disables exposure-increasing transition and explains blocked gates", () => {
    const input = fullyReady("internal_validation");
    input.schema = undefined; // forces G2 blocked
    const gates = deriveGates(input);
    const decision = evaluatePublishAction(
      "internal_validation",
      "private_beta",
      gates,
      input.gaps,
    );
    expect(decision.allowed).toBe(false);
    expect(decision.reasonKeys).toContain("blockedGates");
    expect(blockedGates(gates).length).toBeGreaterThan(0);
  });

  it("disables exposure-increasing transition and explains blocking gaps", () => {
    const input = {
      ...fullyReady("private_beta"),
      gaps: [gap("system_error")],
    };
    const gates = deriveGates(input);
    const decision = evaluatePublishAction(
      "private_beta",
      "available",
      gates,
      input.gaps,
    );
    expect(decision.allowed).toBe(false);
    expect(decision.reasonKeys).toContain("blockingGaps");
  });

  it("allows rollback / exposure-reducing transitions without gate checks", () => {
    const input = fullyReady("available");
    input.schema = undefined; // gates would block an increase
    const gates = deriveGates(input);
    // Deprecate (exposure-reducing) is allowed even with blocked gates.
    expect(
      evaluatePublishAction("available", "deprecated", gates, input.gaps).allowed,
    ).toBe(true);
    // Roll back to internal validation is allowed.
    expect(
      evaluatePublishAction("available", "internal_validation", gates, input.gaps)
        .allowed,
    ).toBe(true);
  });

  it("rejects an illegal transition outright", () => {
    const gates = deriveGates(fullyReady("research"));
    const decision = evaluatePublishAction("research", "available", gates, []);
    expect(decision.allowed).toBe(false);
    expect(decision.reasonKeys).toEqual(["illegalTransition"]);
  });
});

describe("lifecycle helpers", () => {
  it("mirrors the backend transition graph", () => {
    expect(availableTransitions("internal_validation")).toEqual([
      "private_beta",
      "designed",
    ]);
    expect(availableTransitions("retired")).toEqual([]);
  });

  it("allows emergency disable only from private_beta and available", () => {
    expect(canEmergencyDisable("private_beta")).toBe(true);
    expect(canEmergencyDisable("available")).toBe(true);
    expect(canEmergencyDisable("internal_validation")).toBe(false);
    expect(canEmergencyDisable("research")).toBe(false);
  });

  it("picks the most recent completed run", () => {
    const older = {
      ...run("failed"),
      started_at: "2026-06-01T00:00:00Z",
    };
    const newer = {
      ...run("passed"),
      started_at: "2026-06-05T00:00:00Z",
    };
    const pending = {
      ...run("pending", false),
      started_at: "2026-06-06T00:00:00Z",
    };
    expect(lastCompletedRun([older, newer, pending])?.verdict).toBe("passed");
  });
});
