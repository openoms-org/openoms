import { describe, expect, it } from "vitest";
import enDashboard from "../../../../messages/en/dashboard.json";
import plDashboard from "../../../../messages/pl/dashboard.json";

// i18n fidelity guard (OPE-519): the backend emits these enum codes verbatim
// and the dashboard renders t(`fulfillment.blockerCode.${code}`) — a missing
// key leaks the raw dotted key to operators. The literal list below mirrors
// the authoritative Go constants; keep it in sync when the model grows:
//   apps/api-server/internal/model/fulfillment.go (Blocker* constants)
const BLOCKER_CODES = [
  "stock_sync_failed",
  "channel_stock_stale",
  "supplier_availability_stale",
  "supplier_availability_unknown",
  "supplier_preflight_required",
  "supplier_availability_insufficient",
  "manual_stock_review_required",
  "stock_write_unsupported",
  "stock_ack_missing",
  "external_status_unmapped",
  "integration_capability_missing",
  "integration_capability_degraded",
  "automation_action_failed",
  "external_workflow_timeout",
  "supplier_order_missing_data",
  "supplier_order_ambiguous_sku",
  "supplier_order_rejected",
  "supplier_payment_awaiting",
  "supplier_partial_fulfillment",
  "supplier_manual_submission_required",
] as const;

// Same guard for provider-attempt operations (OPE-520). Source of truth:
//   apps/api-server/internal/model/provider_attempt.go (ProviderOp* constants)
const PROVIDER_OPERATIONS = [
  "create_shipment",
  "generate_label",
  "download_label",
  "sync_tracking",
  "sync_tracking_to_marketplace",
  "sync_fulfillment_status",
] as const;

const LOCALES = [
  ["en", enDashboard],
  ["pl", plDashboard],
] as const;

describe("fulfillment blocker-code i18n coverage", () => {
  it.each(LOCALES)(
    "%s catalog labels every backend blocker code",
    (_locale, catalog) => {
      const labels: Record<string, string> =
        catalog.dashboard.fulfillment.blockerCode;
      for (const code of BLOCKER_CODES) {
        expect(labels[code], `missing blockerCode.${code}`).toBeTruthy();
      }
      // Exact set equality also catches stale keys for removed codes.
      expect(Object.keys(labels).sort()).toEqual([...BLOCKER_CODES].sort());
    },
  );

  it("en and pl blocker-code key sets are identical", () => {
    expect(
      Object.keys(enDashboard.dashboard.fulfillment.blockerCode).sort(),
    ).toEqual(Object.keys(plDashboard.dashboard.fulfillment.blockerCode).sort());
  });
});

describe("fulfillment provider-operation i18n coverage", () => {
  it.each(LOCALES)(
    "%s catalog labels every backend provider operation",
    (_locale, catalog) => {
      const labels: Record<string, string> =
        catalog.dashboard.fulfillment.providerOp;
      for (const operation of PROVIDER_OPERATIONS) {
        expect(labels[operation], `missing providerOp.${operation}`).toBeTruthy();
      }
      expect(Object.keys(labels).sort()).toEqual(
        [...PROVIDER_OPERATIONS].sort(),
      );
    },
  );

  it("en and pl provider-operation key sets are identical", () => {
    expect(
      Object.keys(enDashboard.dashboard.fulfillment.providerOp).sort(),
    ).toEqual(Object.keys(plDashboard.dashboard.fulfillment.providerOp).sort());
  });
});
