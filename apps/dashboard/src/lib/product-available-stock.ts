import type { Product } from "@/types/products";

/**
 * Canonical available stock for UI prefills and listing stock overrides.
 * Same hybrid read as product list/detail: warehouse available, with a
 * products.stock_quantity fallback only when the API computed it that way.
 */
export function productAvailableStock(
  product: Pick<Product, "available_stock">
): number {
  return product.available_stock;
}
