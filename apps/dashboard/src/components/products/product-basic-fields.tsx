"use client";

import { useTranslations } from "next-intl";
import type { FieldErrors, UseFormRegister, UseFormSetValue } from "react-hook-form";
import { CategoryTreePicker } from "@/components/shared/category-tree-picker";
import { FormField } from "@/components/shared/form-wrapper";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PRODUCT_SOURCES, type ProductFormValues } from "./product-form.schema";

interface ProductBasicFieldsProps {
  register: UseFormRegister<ProductFormValues>;
  setValue: UseFormSetValue<ProductFormValues>;
  errors: FieldErrors<ProductFormValues>;
  sourceValue: ProductFormValues["source"];
  selectedCategoryId: string | undefined;
  onCategoryChange: (categoryId: string | undefined) => void;
}

export function ProductBasicFields({
  register,
  setValue,
  errors,
  sourceValue,
  selectedCategoryId,
  onCategoryChange,
}: ProductBasicFieldsProps) {
  const t = useTranslations("products");

  return (
    <>
      <FormField<ProductFormValues>
        name="name"
        label={t("form.productName")}
        error={errors.name}
        required
      >
        <Input
          id="name"
          placeholder={t("form.productNamePlaceholder")}
          aria-invalid={!!errors.name}
          {...register("name")}
        />
      </FormField>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <FormField<ProductFormValues> name="sku" label={t("form.sku")}>
          <Input
            id="sku"
            placeholder={t("form.skuPlaceholder")}
            {...register("sku")}
          />
        </FormField>

        <FormField<ProductFormValues> name="ean" label={t("form.ean")}>
          <Input
            id="ean"
            placeholder={t("form.eanPlaceholder")}
            {...register("ean")}
          />
        </FormField>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <FormField<ProductFormValues>
          name="price"
          label={t("form.price")}
          error={errors.price}
          required
        >
          <Input
            id="price"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            aria-invalid={!!errors.price}
            {...register("price", { valueAsNumber: true })}
          />
        </FormField>

        <FormField<ProductFormValues>
          name="stock_quantity"
          label={t("form.stockQuantity")}
          error={errors.stock_quantity}
          required
        >
          <Input
            id="stock_quantity"
            type="number"
            step="1"
            min="0"
            placeholder="0"
            aria-invalid={!!errors.stock_quantity}
            {...register("stock_quantity", { valueAsNumber: true })}
          />
        </FormField>
      </div>

      <FormField<ProductFormValues>
        name="source"
        label={t("form.source")}
        error={errors.source}
        required
      >
        <Select
          value={sourceValue}
          onValueChange={(value) =>
            setValue("source", value as ProductFormValues["source"], {
              shouldValidate: true,
            })
          }
        >
          <SelectTrigger className="w-full" aria-invalid={!!errors.source}>
            <SelectValue placeholder={t("form.sourcePlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {PRODUCT_SOURCES.map((source) => (
              <SelectItem key={source} value={source}>
                {t(`form.sourceOptions.${source}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormField>

      <FormField<ProductFormValues> label={t("form.category")}>
        <CategoryTreePicker
          value={selectedCategoryId}
          onChange={onCategoryChange}
          placeholder={t("form.categoryPlaceholder")}
          className="w-full"
        />
      </FormField>
    </>
  );
}
