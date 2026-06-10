import { describe, expect, it } from "vitest";
import {
  enumLabel,
  humanizeEnumCode,
  type EnumTranslator,
} from "@/lib/i18n-fallback";

function stubTranslator(known: Set<string>): EnumTranslator {
  const t = (key: string) => `translated:${key}`;
  t.has = (key: string) => known.has(key);
  return t;
}

describe("humanizeEnumCode", () => {
  it("turns snake_case into a capitalized sentence", () => {
    expect(humanizeEnumCode("supplier_order_rejected")).toBe(
      "Supplier order rejected",
    );
  });

  it("handles single-word codes", () => {
    expect(humanizeEnumCode("delay")).toBe("Delay");
  });

  it("collapses repeated separators and trims", () => {
    expect(humanizeEnumCode("__weird__code_ ")).toBe("Weird code");
  });

  it("returns the input unchanged when nothing remains", () => {
    expect(humanizeEnumCode("___")).toBe("___");
  });
});

describe("enumLabel", () => {
  it("returns the translation when the key exists", () => {
    const t = stubTranslator(
      new Set(["fulfillment.blockerCode.supplier_order_rejected"]),
    );
    expect(enumLabel(t, "fulfillment.blockerCode", "supplier_order_rejected")).toBe(
      "translated:fulfillment.blockerCode.supplier_order_rejected",
    );
  });

  it("falls back to the humanized code instead of the raw dotted key", () => {
    const t = stubTranslator(new Set());
    expect(enumLabel(t, "fulfillment.blockerCode", "brand_new_backend_code")).toBe(
      "Brand new backend code",
    );
  });
});
