"use client";

import { useTranslations } from "next-intl";
import type { FieldErrors, UseFormRegister, UseFormSetValue } from "react-hook-form";
import { FormField } from "@/components/shared/form-wrapper";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { OrderFormValues } from "./order-form.schema";

interface OrderCustomerFieldsProps {
  register: UseFormRegister<OrderFormValues>;
  setValue: UseFormSetValue<OrderFormValues>;
  errors: FieldErrors<OrderFormValues>;
  currentSource: string;
  currency: string;
}

export function OrderCustomerFields({
  register,
  setValue,
  errors,
  currentSource,
  currency,
}: OrderCustomerFieldsProps) {
  const t = useTranslations("orders");

  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
      <FormField<OrderFormValues>
        name="source"
        label={t("form.source")}
        error={errors.source}
        required
      >
        <Select
          value={currentSource}
          onValueChange={(value) => setValue("source", value, { shouldValidate: true })}
        >
          <SelectTrigger aria-invalid={!!errors.source}>
            <SelectValue placeholder={t("form.sourcePlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="manual">{t("form.manual")}</SelectItem>
            <SelectItem value="allegro">Allegro</SelectItem>
            <SelectItem value="amazon">Amazon</SelectItem>
            <SelectItem value="ebay">eBay</SelectItem>
            <SelectItem value="erli">Erli</SelectItem>
            <SelectItem value="woocommerce">WooCommerce</SelectItem>
            <SelectItem value="shopify">Shopify</SelectItem>
            <SelectItem value="olx">OLX</SelectItem>
            <SelectItem value="other">{t("form.other")}</SelectItem>
          </SelectContent>
        </Select>
      </FormField>

      <FormField<OrderFormValues>
        name="customer_name"
        label={t("form.customerName")}
        error={errors.customer_name}
        required
      >
        <Input
          id="customer_name"
          placeholder={t("form.customerNamePlaceholder")}
          aria-invalid={!!errors.customer_name}
          {...register("customer_name")}
        />
      </FormField>

      <FormField<OrderFormValues>
        name="customer_email"
        label={t("form.customerEmail")}
        error={errors.customer_email}
      >
        <Input
          id="customer_email"
          type="email"
          placeholder="jan@example.com"
          aria-invalid={!!errors.customer_email}
          {...register("customer_email")}
        />
      </FormField>

      <FormField<OrderFormValues>
        name="customer_phone"
        label={t("form.customerPhone")}
        error={errors.customer_phone}
      >
        <Input
          id="customer_phone"
          placeholder="+48 123 456 789"
          {...register("customer_phone")}
        />
      </FormField>

      <FormField<OrderFormValues> name="currency" label={t("form.currency")} required>
        <Input
          id="currency"
          value={currency}
          readOnly
          disabled
          className="bg-muted"
        />
      </FormField>
    </div>
  );
}
