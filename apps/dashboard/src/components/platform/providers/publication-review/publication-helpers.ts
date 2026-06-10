import {
  ALLOWED_PROVIDER_TRANSITIONS,
  EMERGENCY_DISABLE_FROM_STATES,
  type ProviderCapability,
  type ProviderFieldSchema,
  type ProviderIntegrationGap,
  type ProviderPublicationState,
  type ProviderValidationRun,
  type ProviderVersion,
  type RunVerdict,
} from "@/types/platform";

/** The nine production-readiness gates (production-readiness spec §452). */
export type GateId =
  | "G1"
  | "G2"
  | "G3"
  | "G4"
  | "G5"
  | "G6"
  | "G7"
  | "G8"
  | "G9";

export const GATE_IDS: GateId[] = [
  "G1",
  "G2",
  "G3",
  "G4",
  "G5",
  "G6",
  "G7",
  "G8",
  "G9",
];

export type GateStatus = "pass" | "warn" | "blocked" | "manual";

export interface GateState {
  id: GateId;
  status: GateStatus;
  /** True when the status is derived client-side rather than from a backend gate API. */
  derived: boolean;
  /** Stable message key suffix under platform.publicationReview.gateReason.* */
  reasonKey: string;
}

export interface VersionReadinessInput {
  version: ProviderVersion;
  schema: ProviderFieldSchema | undefined;
  capabilities: ProviderCapability[];
  gaps: ProviderIntegrationGap[];
  runs: ProviderValidationRun[];
}

/** Open (unresolved) gaps. */
export function openGaps(
  gaps: ProviderIntegrationGap[],
): ProviderIntegrationGap[] {
  return gaps.filter((g) => g.status !== "resolved");
}

/** Open gaps whose severity blocks publication. */
export function blockingGaps(
  gaps: ProviderIntegrationGap[],
): ProviderIntegrationGap[] {
  return openGaps(gaps).filter(
    (g) => g.severity === "action_required" || g.severity === "system_error",
  );
}

/** Open gaps that are warnings (non-blocking but surfaced). */
export function warningGaps(
  gaps: ProviderIntegrationGap[],
): ProviderIntegrationGap[] {
  return openGaps(gaps).filter(
    (g) => g.severity === "warning" || g.severity === "info",
  );
}

/** The most recent run by started_at, or undefined when there are none. */
export function lastRun(
  runs: ProviderValidationRun[],
): ProviderValidationRun | undefined {
  if (runs.length === 0) return undefined;
  return [...runs].sort(
    (a, b) =>
      new Date(b.started_at).getTime() - new Date(a.started_at).getTime(),
  )[0];
}

/** The most recent run that finished with a definite verdict (passed/failed/error). */
export function lastCompletedRun(
  runs: ProviderValidationRun[],
): ProviderValidationRun | undefined {
  return [...runs]
    .filter((r) => r.verdict !== "pending")
    .sort(
      (a, b) =>
        new Date(b.started_at).getTime() - new Date(a.started_at).getTime(),
    )[0];
}

function schemaHasFields(schema: ProviderFieldSchema | undefined): boolean {
  return !!schema && schema.groups.some((g) => g.fields.length > 0);
}

function capabilitiesComplete(capabilities: ProviderCapability[]): boolean {
  if (capabilities.length === 0) return false;
  // A capability left "unknown" means there is no evidence yet — it blocks.
  return capabilities.every((c) => c.support_status !== "unknown");
}

/**
 * Derive the G1-G9 gate checklist for a version. Most gates have NO dedicated
 * backend endpoint, so they are derived from version state, schema completeness,
 * capabilities, the last validation verdict and open gaps. Every derived gate is
 * flagged `derived: true` and gates that can only be confirmed by a human
 * (operational readiness, security review, certification) are reported as
 * `manual` rather than asserted as automatically passing.
 */
export function deriveGates(input: VersionReadinessInput): GateState[] {
  const { version, schema, capabilities, gaps, runs } = input;
  const blocking = blockingGaps(gaps);
  const completed = lastCompletedRun(runs);
  const lastVerdict: RunVerdict | undefined = completed?.verdict;

  // G1 Platform security — the studio is only reachable behind platform-admin
  // auth + per-route permission middleware. If this surface rendered at all, the
  // boundary held; backend remains authoritative.
  const g1: GateState = { id: "G1", status: "pass", derived: true, reasonKey: "g1Ok" };

  // G2 Data foundation — registry/version/schema/capabilities exist for this
  // version. Blocked until a credential schema has fields.
  const g2: GateState = schemaHasFields(schema)
    ? { id: "G2", status: "pass", derived: true, reasonKey: "g2Ok" }
    : { id: "G2", status: "blocked", derived: true, reasonKey: "g2NoSchema" };

  // G3 Validation engine — a completed run that passed.
  let g3: GateState;
  if (!completed) {
    g3 = { id: "G3", status: "blocked", derived: true, reasonKey: "g3NoRun" };
  } else if (lastVerdict === "passed") {
    g3 = { id: "G3", status: "pass", derived: true, reasonKey: "g3Ok" };
  } else {
    g3 = { id: "G3", status: "blocked", derived: true, reasonKey: "g3LastRunFailed" };
  }

  // G4 Publication control — visibility is backend-controlled by publication
  // state. Always satisfied by architecture (informational).
  const g4: GateState = { id: "G4", status: "pass", derived: true, reasonKey: "g4Ok" };

  // G5 Evidence safety — redaction/hashing are enforced by the validation engine;
  // we confirm runs actually recorded evidence (results with payload hashes).
  const hasEvidence = runs.some((r) => (r.results?.length ?? 0) > 0);
  const g5: GateState = hasEvidence
    ? { id: "G5", status: "pass", derived: true, reasonKey: "g5Ok" }
    : { id: "G5", status: "warn", derived: true, reasonKey: "g5NoEvidence" };

  // G6 Migration safety — capability completeness stands in for "existing
  // providers map cleanly". Unknown capabilities mean the mapping is incomplete.
  const g6: GateState = capabilitiesComplete(capabilities)
    ? { id: "G6", status: "pass", derived: true, reasonKey: "g6Ok" }
    : { id: "G6", status: "warn", derived: true, reasonKey: "g6CapabilitiesIncomplete" };

  // G7 Operational readiness — runbooks/dashboards/alerts are a human sign-off
  // with no backend signal here.
  const g7: GateState = { id: "G7", status: "manual", derived: true, reasonKey: "g7Manual" };

  // G8 Certification — class certification is a human/portfolio decision.
  const g8: GateState = { id: "G8", status: "manual", derived: true, reasonKey: "g8Manual" };

  // G9 Security review — threat model / ASVS review is a human sign-off.
  const g9: GateState = { id: "G9", status: "manual", derived: true, reasonKey: "g9Manual" };

  // Any blocking gap forces the data/validation gates that own gaps to blocked.
  if (blocking.length > 0) {
    if (g3.status === "pass") {
      g3 = { id: "G3", status: "blocked", derived: true, reasonKey: "g3OpenBlockingGaps" };
    }
  }

  return [g1, g2, g3, g4, g5, g6, g7, g8, g9];
}

/** Gates that hard-block a publish action (status === "blocked"). */
export function blockedGates(gates: GateState[]): GateState[] {
  return gates.filter((g) => g.status === "blocked");
}

/** Available lifecycle transitions from a state (mirror of the backend graph). */
export function availableTransitions(
  state: ProviderPublicationState,
): ProviderPublicationState[] {
  return ALLOWED_PROVIDER_TRANSITIONS[state] ?? [];
}

/** True when emergency disable applies to a version's current state. */
export function canEmergencyDisable(state: ProviderPublicationState): boolean {
  return EMERGENCY_DISABLE_FROM_STATES.has(state);
}

export interface PublishActionGate {
  /** Whether the publish-to-target action is allowed. */
  allowed: boolean;
  /** Stable reason keys (under publicationReview.disabledReason.*) when not allowed. */
  reasonKeys: string[];
  /** The blocked gates contributing to the decision (for explicit messaging). */
  blocked: GateState[];
}

/**
 * Decide whether a publish transition to `target` is allowed and, when not,
 * explain the concrete missing gates/gaps. Promotion to private_beta or
 * available is gated on readiness; rollback-style transitions
 * (-> internal_validation, -> designed, deprecate, retire) are NOT gate-blocked
 * because they reduce or end exposure.
 */
export function evaluatePublishAction(
  from: ProviderPublicationState,
  target: ProviderPublicationState,
  gates: GateState[],
  gaps: ProviderIntegrationGap[],
): PublishActionGate {
  const reasonKeys: string[] = [];

  if (!ALLOWED_PROVIDER_TRANSITIONS[from]?.includes(target)) {
    reasonKeys.push("illegalTransition");
    return { allowed: false, reasonKeys, blocked: [] };
  }

  // Transitions that increase customer exposure must satisfy readiness gates.
  const exposureIncreasing =
    target === "private_beta" || target === "available";

  if (!exposureIncreasing) {
    return { allowed: true, reasonKeys: [], blocked: [] };
  }

  const blocked = blockedGates(gates);
  if (blocked.length > 0) {
    reasonKeys.push("blockedGates");
  }
  const blocking = blockingGaps(gaps);
  if (blocking.length > 0) {
    reasonKeys.push("blockingGaps");
  }

  return { allowed: reasonKeys.length === 0, reasonKeys, blocked };
}
