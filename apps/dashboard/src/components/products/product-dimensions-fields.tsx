"use client";

import { useTranslations } from "next-intl";
import type { FieldErrors, UseFormRegister } from "react-hook-form";
import { FormField } from "@/components/shared/form-wrapper";
import { Input } from "@/components/ui/input";
import type { ProductFormValues } from "./product-form.schema";

interface ProductDimensionsFieldsProps {
  register: UseFormRegister<ProductFormValues>;
  errors: FieldErrors<ProductFormValues>;
}

export function ProductDimensionsFields({ register, errors }: ProductDimensionsFieldsProps) {
  const t = useTranslations("products");

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium">{t("form.dimensionsAndWeight")}</h3>
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <FormField<ProductFormValues> name="weight" label={t("form.weight")} error={errors.weight}>
          <Input
            id="weight"
            type="number"
            step="0.001"
            min="0"
            placeholder="0.000"
            {...register("weight", { valueAsNumber: true })}
          />
        </FormField>
        <FormField<ProductFormValues> name="width" label={t("form.width")} error={errors.width}>
          <Input
            id="width"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            {...register("width", { valueAsNumber: true })}
          />
        </FormField>
        <FormField<ProductFormValues> name="height" label={t("form.height")} error={errors.height}>
          <Input
            id="height"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            {...register("height", { valueAsNumber: true })}
          />
        </FormField>
        <FormField<ProductFormValues> name="depth" label={t("form.depth")} error={errors.depth}>
          <Input
            id="depth"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            {...register("depth", { valueAsNumber: true })}
          />
        </FormField>
      </div>
    </div>
  );
}
