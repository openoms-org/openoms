import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { productAvailableStock } from "@/lib/product-available-stock";

const srcRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");

describe("productAvailableStock", () => {
  it("returns available_stock when it differs from the legacy column", () => {
    const product = {
      available_stock: 7,
      stock_quantity: 100,
    };

    expect(productAvailableStock(product)).toBe(7);
    expect(productAvailableStock(product)).not.toBe(product.stock_quantity);
  });

  it("returns 0 when warehouse rows exist but are empty", () => {
    expect(productAvailableStock({ available_stock: 0 })).toBe(0);
  });
});

describe("leftover stock_quantity prefills", () => {
  it("listing dialogs and product form prefill from productAvailableStock", () => {
    const listings = readFileSync(
      resolve(srcRoot, "app/(dashboard)/products/[id]/listings/page.tsx"),
      "utf8"
    );
    const form = readFileSync(
      resolve(srcRoot, "components/products/product-form.tsx"),
      "utf8"
    );

    expect(listings).not.toMatch(/product\.stock_quantity/);
    expect(listings).toContain("productAvailableStock(product)");
    expect(form).not.toMatch(/product\?\.stock_quantity/);
    expect(form).toContain("productAvailableStock(product)");
  });
});

