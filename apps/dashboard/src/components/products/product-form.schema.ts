import { z } from "zod";

export const PRODUCT_SOURCES = ["manual", "allegro", "woocommerce"] as const;

const optionalDimension = z.preprocess((value) => {
  if (value === "" || value === null || Number.isNaN(value)) {
    return undefined;
  }
  return value;
}, z.number().min(0).optional());

export function createProductSchema(t: (key: string) => string) {
  return z.object({
    name: z.string().min(1, t("nameRequired")),
    sku: z.string().optional(),
    ean: z.string().optional(),
    price: z.number().min(0, t("priceMin")),
    stock_quantity: z
      .number()
      .int(t("quantityInteger"))
      .min(0, t("quantityMin")),
    source: z.enum(PRODUCT_SOURCES),
    description_short: z.string().optional(),
    description_long: z.string().optional(),
    weight: optionalDimension,
    width: optionalDimension,
    height: optionalDimension,
    depth: optionalDimension,
    image_url: z.string(),
  });
}

export type ProductFormValues = z.infer<ReturnType<typeof createProductSchema>>;
