// Operations-dashboard cutover flag (OPE-423c).
//
// The operations dashboard home (src/app/(dashboard)/page.tsx) carries TWO
// code paths side by side (dual-read):
//   - the LEGACY heuristic view (orchestration map + operational exceptions,
//     derived from order/shipment status by use-operations-dashboard.ts), and
//   - the PROCESS-BACKED view (the OPE-419 control tower over the fulfillment
//     read model: /v1/operations/summary, /exceptions, /v1/fulfillment/*).
//
// This flag selects which one is PRIMARY on the home page. It is a build-time
// switch read straight off the environment, mirroring how readiness.ts reads
// NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE — no runtime config, no API call.
//
// DEFAULT / UNSET MUST mean the current production behavior: "heuristic". The
// actual cutover is then just flipping this flag to "process-backed" once the
// OPE-423b parity gate (process_coverage_met) is satisfied — flipping it does
// NOT delete the legacy path, so it is reversible by flipping back.

export type FulfillmentDashboardMode = "heuristic" | "process-backed";

const DEFAULT_FULFILLMENT_DASHBOARD_MODE: FulfillmentDashboardMode = "heuristic";

// getFulfillmentDashboardMode resolves the active operations-dashboard mode from
// NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD. Only the exact string
// "process-backed" opts into the process-backed primary view; every other value
// (including unset, empty, or an unknown value) falls back to "heuristic" so the
// default is always the safe, current behavior.
export function getFulfillmentDashboardMode(): FulfillmentDashboardMode {
  return process.env.NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD === "process-backed"
    ? "process-backed"
    : DEFAULT_FULFILLMENT_DASHBOARD_MODE;
}

// isProcessBackedDashboard is a small convenience over getFulfillmentDashboardMode
// for the common "should the process-backed view be primary?" check.
export function isProcessBackedDashboard(): boolean {
  return getFulfillmentDashboardMode() === "process-backed";
}
