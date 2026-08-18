import { readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import enStatuses from "../../../messages/en/statuses.json";
import plStatuses from "../../../messages/pl/statuses.json";
import {
  DROPSHIP_STATUSES,
  INTEGRATION_STATUSES,
  INVOICE_STATUS_MAP,
  KSEF_STATUS_MAP,
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
  ["ksef", KSEF_STATUS_MAP],
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

const STATUS_MAP_PREFIX: Record<string, string> = {
  ORDER_STATUSES: "order",
  orderStatuses: "order",
  SHIPMENT_STATUSES: "shipment",
  RETURN_STATUSES: "return",
  PURCHASE_ORDER_STATUSES: "purchaseOrder",
  SUPPLIER_STATUSES: "supplier",
  INTEGRATION_STATUSES: "integration",
  PAYMENT_STATUSES: "payment",
  INVOICE_STATUS_MAP: "invoice",
  KSEF_STATUS_MAP: "ksef",
  DROPSHIP_STATUSES: "dropship",
  LOYALTY_STATUSES: "loyalty",
  STOCKTAKE_STATUSES: "stocktake",
  RECURRING_ORDER_STATUSES: "recurringOrder",
  WAREHOUSE_DOCUMENT_STATUSES: "warehouseDocument",
  PICK_PACK_STATUSES: "pickPack",
  REPRICING_RULE_STATUSES: "repricingRule",
};

function listTsx(dir: string, acc: string[] = []): string[] {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === "__tests__" || entry.name === "node_modules") continue;
      listTsx(path, acc);
      continue;
    }
    if (!entry.name.endsWith(".tsx")) continue;
    if (entry.name.includes(".test.") || entry.name.includes(".spec.")) continue;
    acc.push(path);
  }
  return acc;
}

describe("StatusBadge call sites pass translationPrefix", () => {
  const srcRoot = join(dirname(fileURLToPath(import.meta.url)), "../..");

  it("does not render a catalogued status map as a raw enum", () => {
    const missing: string[] = [];

    for (const file of listTsx(srcRoot)) {
      const source = readFileSync(file, "utf8");
      const badges = source.match(/<StatusBadge\b[\s\S]*?\/>/g) ?? [];
      for (const badge of badges) {
        if (/\blabel=/.test(badge)) continue;
        const map = badge.match(/statusMap=\{(\w+)\}/);
        if (!map) {
          if (/statusMap=\{\{/.test(badge)) {
            missing.push(`${file}: inline statusMap instead of a catalogued constant`);
          }
          continue;
        }
        const prefix = STATUS_MAP_PREFIX[map[1]];
        if (!prefix) continue;
        const quoted = badge.match(/translationPrefix=["']([^"']+)["']/);
        if (quoted?.[1] !== prefix) {
          missing.push(`${file}: ${map[1]} needs translationPrefix="${prefix}"`);
        }
      }
    }

    expect(missing).toEqual([]);
  });
});
