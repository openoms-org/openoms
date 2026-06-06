import type {
  MappingConfidence,
  ProviderIntegrationGap,
  ProviderStatusMapping,
  StatusDomain,
} from "@/types/platform";

/** A canonical order status that means the order is done with no further work. */
const TERMINAL_ORDER_STATUSES = new Set([
  "delivered",
  "completed",
  "cancelled",
  "refunded",
]);

/**
 * Canonical statuses that belong to the commercial *order* lifecycle. A shipment
 * mapping must not silently steer the order into one of these (design spec §390,
 * builder template: "Shipment status must not automatically overwrite commercial
 * order status").
 */
const COMMERCIAL_ORDER_STATUSES = new Set([
  "new",
  "confirmed",
  "processing",
  "submitted",
  "awaiting_payment",
  "on_hold",
  "completed",
  "cancelled",
  "refunded",
]);

export type MappingWarningCode =
  | "terminalLowConfidence"
  | "shipmentOverwritesOrder"
  | "unknownNeedsGap";

export interface MappingWarning {
  code: MappingWarningCode;
  domain: StatusDomain;
  rawStatus: string;
  /** True for action_required problems that should block safe automation. */
  blocking: boolean;
}

/**
 * A raw status whose canonical target is empty or literally "unknown" is an
 * unmapped gap — it must NOT silently resolve to a success status.
 */
export function isUnknownMapping(mapping: ProviderStatusMapping): boolean {
  const canonical = mapping.canonical_status.trim().toLowerCase();
  return canonical === "" || canonical === "unknown";
}

/** True when a mapping points at a terminal canonical order status. */
export function isTerminalCanonical(mapping: ProviderStatusMapping): boolean {
  return (
    mapping.is_terminal ||
    TERMINAL_ORDER_STATUSES.has(mapping.canonical_status.trim().toLowerCase())
  );
}

function lowConfidence(confidence: MappingConfidence): boolean {
  return confidence === "low" || confidence === "medium";
}

/**
 * Whether an open/acknowledged gap already covers this unknown raw status, so
 * the workbench doesn't double-warn once the operator has filed the gap.
 */
function hasOpenGapForUnknown(gaps: ProviderIntegrationGap[]): boolean {
  return gaps.some(
    (g) =>
      g.gap_type === "missing_status_mapping" &&
      (g.status === "open" || g.status === "acknowledged"),
  );
}

/**
 * Compute the safety warnings for a set of status mappings. Three rules
 * (design spec §389-390, builder template §324-331):
 *   1. terminalLowConfidence — a terminal status with confidence below "high".
 *   2. shipmentOverwritesOrder — a shipment-domain mapping targeting a
 *      commercial order status (would let a shipment update overwrite the order).
 *   3. unknownNeedsGap — an unknown/empty canonical mapping that is not yet
 *      covered by an open missing_status_mapping gap. Unknowns never silently
 *      map to success; they must be filed as gaps.
 */
export function computeMappingWarnings(
  mappings: ProviderStatusMapping[],
  gaps: ProviderIntegrationGap[] = [],
): MappingWarning[] {
  const warnings: MappingWarning[] = [];
  const gapFiled = hasOpenGapForUnknown(gaps);

  for (const m of mappings) {
    if (isUnknownMapping(m)) {
      // An unmapped raw status is a gap, not a usable mapping. Warn until a gap
      // is filed; never treat it as a successful mapping.
      if (!gapFiled) {
        warnings.push({
          code: "unknownNeedsGap",
          domain: m.status_domain,
          rawStatus: m.raw_status,
          blocking: true,
        });
      }
      continue;
    }

    if (isTerminalCanonical(m) && lowConfidence(m.confidence)) {
      warnings.push({
        code: "terminalLowConfidence",
        domain: m.status_domain,
        rawStatus: m.raw_status,
        blocking: true,
      });
    }

    if (
      m.status_domain === "shipment" &&
      COMMERCIAL_ORDER_STATUSES.has(m.canonical_status.trim().toLowerCase())
    ) {
      warnings.push({
        code: "shipmentOverwritesOrder",
        domain: m.status_domain,
        rawStatus: m.raw_status,
        blocking: true,
      });
    }
  }

  return warnings;
}

/** Index warnings by `${domain}::${rawStatus}` for per-row lookup. */
export function warningsByRow(
  warnings: MappingWarning[],
): Map<string, MappingWarning[]> {
  const map = new Map<string, MappingWarning[]>();
  for (const w of warnings) {
    const key = rowKey(w.domain, w.rawStatus);
    const bucket = map.get(key);
    if (bucket) {
      bucket.push(w);
    } else {
      map.set(key, [w]);
    }
  }
  return map;
}

export function rowKey(domain: StatusDomain, rawStatus: string): string {
  return `${domain}::${rawStatus}`;
}

/** Mappings for a single domain tab. */
export function mappingsForDomain(
  mappings: ProviderStatusMapping[],
  domain: StatusDomain,
): ProviderStatusMapping[] {
  return mappings.filter((m) => m.status_domain === domain);
}

/** A blank mapping row for a given domain. */
export function emptyMapping(domain: StatusDomain): ProviderStatusMapping {
  return {
    status_domain: domain,
    raw_status: "",
    canonical_status: "",
    canonical_event_type: "",
    canonical_step_key: "",
    confidence: "low",
    is_terminal: false,
    notes: "",
  };
}
