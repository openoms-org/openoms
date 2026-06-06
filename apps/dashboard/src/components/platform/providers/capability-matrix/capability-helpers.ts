import type { ProviderCapability, SupportStatus } from "@/types/platform";

type BadgeTone = "success" | "warning" | "info" | "secondary" | "destructive" | "outline";

/**
 * Tone for each backend support status. Meaning is never conveyed by color alone
 * (design spec §352) — every badge carries a translated label and tooltip too.
 */
const SUPPORT_TONES: Record<SupportStatus, BadgeTone> = {
  supported: "success",
  configured: "info",
  requires_manual: "info",
  degraded: "warning",
  unsupported: "secondary",
  unknown: "destructive",
};

export function supportStatusTone(status: SupportStatus): BadgeTone {
  return SUPPORT_TONES[status] ?? "outline";
}

/**
 * A capability needs published evidence before it can be trusted/automated.
 * `unknown` always requires evidence (it blocks publication when the capability
 * is required); `degraded` requires evidence to justify the limitation. A
 * capability with no notes/evidence link is flagged so the matrix can prompt for
 * one rather than silently shipping an unverified claim.
 */
export function isEvidenceRequired(capability: ProviderCapability): boolean {
  return (
    capability.support_status === "unknown" ||
    capability.support_status === "degraded"
  );
}

/** True when evidence is required but none has been recorded (empty notes). */
export function isEvidenceMissing(capability: ProviderCapability): boolean {
  return isEvidenceRequired(capability) && capability.notes.trim() === "";
}

/**
 * Capability keys follow a `domain.area.action` convention (e.g.
 * `supplier.catalog.read`). The leading segment is the business domain used to
 * group rows in the matrix. Unprefixed keys fall back to "other".
 */
export function capabilityDomain(capability: ProviderCapability): string {
  const head = capability.capability_key.split(".")[0]?.trim();
  return head ? head : "other";
}

/** Group capabilities by their leading domain segment, domains sorted A→Z. */
export function groupByDomain(
  capabilities: ProviderCapability[],
): { domain: string; capabilities: ProviderCapability[] }[] {
  const byDomain = new Map<string, ProviderCapability[]>();
  for (const cap of capabilities) {
    const domain = capabilityDomain(cap);
    const bucket = byDomain.get(domain);
    if (bucket) {
      bucket.push(cap);
    } else {
      byDomain.set(domain, [cap]);
    }
  }
  return [...byDomain.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([domain, caps]) => ({ domain, capabilities: caps }));
}

/**
 * Whether the capability's evidence/notes reference a validation probe. Derived
 * (not a persisted column): we look for a probe/run reference in the evidence
 * text. This drives the read-only "Validation probe" indicator in the matrix.
 */
export function hasProbeReference(capability: ProviderCapability): boolean {
  const notes = capability.notes.toLowerCase();
  return /probe|validation run|run:/.test(notes);
}

/**
 * Whether a capability would be shown to customers. Derived from support state:
 * only verified/configured/limited states are customer-visible; unknown and
 * unsupported capabilities stay hidden so we never advertise an unverified or
 * absent capability.
 */
export function isCustomerVisible(capability: ProviderCapability): boolean {
  switch (capability.support_status) {
    case "supported":
    case "configured":
    case "requires_manual":
    case "degraded":
      return true;
    default:
      return false;
  }
}

/** A blank capability row with safe defaults. */
export function emptyCapability(): ProviderCapability {
  return {
    capability_key: "",
    support_status: "unknown",
    channel: "",
    mode: "",
    freshness: "",
    required_inputs: [],
    provided_outputs: [],
    latency_sla_seconds: null,
    notes: "",
  };
}
