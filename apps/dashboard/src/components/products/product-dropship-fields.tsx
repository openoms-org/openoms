"use client";

import { useTranslations } from "next-intl";
import { FormField } from "@/components/shared/form-wrapper";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useAllSuppliers } from "@/hooks/use-suppliers";
import type { ProductFormValues } from "./product-form.schema";

interface ProductDropshipFieldsProps {
  isDropship: boolean;
  dropshipSupplierId: string;
  onDropshipChange: (enabled: boolean) => void;
  onSupplierChange: (supplierId: string) => void;
}

export function ProductDropshipFields({
  isDropship,
  dropshipSupplierId,
  onDropshipChange,
  onSupplierChange,
}: ProductDropshipFieldsProps) {
  const t = useTranslations("products");
  const { data: suppliersData } = useAllSuppliers();
  const switchId = "dropship-product";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor={switchId}>{t("form.dropshipProduct")}</Label>
          <p className="text-xs text-muted-foreground">
            {t("form.dropshipDescription")}
          </p>
        </div>
        <Switch id={switchId} checked={isDropship} onCheckedChange={onDropshipChange} />
      </div>
      {isDropship && (
        <FormField<ProductFormValues> label={t("form.dropshipSupplier")}>
          <Select value={dropshipSupplierId} onValueChange={onSupplierChange}>
            <SelectTrigger>
              <SelectValue placeholder={t("form.dropshipSupplierPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {suppliersData?.items?.map((supplier) => (
                <SelectItem key={supplier.id} value={supplier.id}>
                  {supplier.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FormField>
      )}
    </div>
  );
}
