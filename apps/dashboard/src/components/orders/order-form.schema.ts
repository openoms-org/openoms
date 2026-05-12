import { z } from "zod";

export interface OrderItemRow {
  id: string;
  name: string;
  sku: string;
  quantity: number;
  price: number;
}

export interface AddressFields {
  name: string;
  street: string;
  city: string;
  postal_code: string;
  country: string;
}

export type OrderPriority = "urgent" | "high" | "normal" | "low";

export const emptyAddress: AddressFields = {
  name: "",
  street: "",
  city: "",
  postal_code: "",
  country: "PL",
};

export function createOrderSchema(t: (key: string) => string) {
  return z.object({
    source: z.string().min(1, t("validation.sourceRequired")),
    customer_name: z.string().min(1, t("validation.customerNameRequired")),
    customer_email: z.union([
      z.literal(""),
      z.string().email(t("validation.invalidEmail")),
    ]).optional(),
    customer_phone: z.string().optional(),
    total_amount: z.number().min(0, t("validation.amountMin")),
    currency: z.string().min(1, t("validation.currencyRequired")),
    notes: z.string().optional(),
  });
}

export type OrderFormValues = z.infer<ReturnType<typeof createOrderSchema>>;
