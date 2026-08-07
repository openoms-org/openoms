import { describe, expect, it } from "vitest";

import enStatuses from "../../../messages/en/statuses.json";
import plStatuses from "../../../messages/pl/statuses.json";
import {
  DROPSHIP_STATUSES,
  INTEGRATION_STATUSES,
  INVOICE_STATUS_MAP,
  LOYALTY_STATUSES,
  ORDER_STATUSES,
  PAYMENT_STATUSES,
  PICK_PACK_STATUSES,
  PURCHASE_ORDER_STATUSES,
  RECURRING_ORDER_STATUSES,
  REPRICING_RULE_STATUSES,
  RETURN_STATUSES,
  SHIPMENT_STATUSES,
  STOCKTAKE_STATUSES,
  SUPPLIER_STATUSES,
  WAREHOUSE_DOCUMENT_STATUSES,
} from "@/lib/constants";

/**
 * StatusBadge resolves its label as `statuses.<translationPrefix>.<status>`, falling back
 * to the raw status key when the message is missing. That fallback is silent, so a status
 * present in the constant but absent from the catalogue ships as an untranslated key —
 * exactly how `purchaseOrder` lost `confirmed` and `shipped`, both of which the supplier
 * portal sets, while the badge rendered them bare.
 *
 * Pair each constant with the prefix the pages pass so the two cannot drift apart again.
 */
const FAMILIES: Array<[string, Record<string, { label: string; color: string }>]> = [
  ["order", ORDER_STATUSES],
  ["shipment", SHIPMENT_STATUSES],
  ["return", RETURN_STATUSES],
  ["purchaseOrder", PURCHASE_ORDER_STATUSES],
  ["supplier", SUPPLIER_STATUSES],
  ["integration", INTEGRATION_STATUSES],
  ["payment", PAYMENT_STATUSES],
  ["invoice", INVOICE_STATUS_MAP],
  ["dropship", DROPSHIP_STATUSES],
  ["loyalty", LOYALTY_STATUSES],
  ["stocktake", STOCKTAKE_STATUSES],
  ["recurringOrder", RECURRING_ORDER_STATUSES],
  ["warehouseDocument", WAREHOUSE_DOCUMENT_STATUSES],
  ["pickPack", PICK_PACK_STATUSES],
  ["repricingRule", REPRICING_RULE_STATUSES],
];

const catalogues = {
  pl: plStatuses.statuses as Record<string, Record<string, string>>,
  en: enStatuses.statuses as Record<string, Record<string, string>>,
};

describe("status constants have translations", () => {
  it.each(FAMILIES)("%s covers every status in both locales", (prefix, statusMap) => {
    const expected = Object.keys(statusMap).sort();

    for (const [locale, catalogue] of Object.entries(catalogues)) {
      const family = catalogue[prefix];
      expect(family, `statuses.${prefix} missing from messages/${locale}`).toBeDefined();

      const missing = expected.filter((status) => !(status in family));
      expect(missing, `messages/${locale}: statuses.${prefix} missing ${missing.join(", ")}`).toEqual([]);
    }
  });

  it("keeps pl and en in step for every status family", () => {
    expect(Object.keys(catalogues.pl).sort()).toEqual(Object.keys(catalogues.en).sort());

    for (const family of Object.keys(catalogues.pl)) {
      expect(Object.keys(catalogues.pl[family]).sort(), `family ${family}`).toEqual(
        Object.keys(catalogues.en[family]).sort()
      );
    }
  });

  it("translates the purchase order states the supplier portal sets", () => {
    // Regression guard for the gap this test was written after.
    for (const status of ["confirmed", "shipped"]) {
      expect(PURCHASE_ORDER_STATUSES[status]).toBeDefined();
      expect(catalogues.pl.purchaseOrder[status]).toBeTruthy();
      expect(catalogues.en.purchaseOrder[status]).toBeTruthy();
    }
  });
});
