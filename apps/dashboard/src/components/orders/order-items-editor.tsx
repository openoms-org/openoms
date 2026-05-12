"use client";

import { useEffect } from "react";
import { Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import type { FieldErrors, UseFormRegister, UseFormSetValue } from "react-hook-form";
import { FormField } from "@/components/shared/form-wrapper";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { formatCurrency } from "@/lib/utils";
import type { OrderFormValues, OrderItemRow } from "./order-form.schema";

interface OrderItemsEditorProps {
  items: OrderItemRow[];
  currency: string;
  totalAmountValue: number;
  errors: FieldErrors<OrderFormValues>;
  register: UseFormRegister<OrderFormValues>;
  setValue: UseFormSetValue<OrderFormValues>;
  onAddItem: () => void;
  onRemoveItem: (index: number) => void;
  onUpdateItem: (index: number, field: keyof OrderItemRow, value: string | number) => void;
}

export function OrderItemsEditor({
  items,
  currency,
  totalAmountValue,
  errors,
  register,
  setValue,
  onAddItem,
  onRemoveItem,
  onUpdateItem,
}: OrderItemsEditorProps) {
  const t = useTranslations("orders");
  const itemsTotal = items.reduce((sum, item) => sum + item.quantity * item.price, 0);
  const hasItems = items.length > 0;

  useEffect(() => {
    if (hasItems) {
      setValue("total_amount", Math.round(itemsTotal * 100) / 100, {
        shouldValidate: true,
      });
    }
  }, [itemsTotal, hasItems, setValue]);

  return (
    <>
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium">{t("form.orderItems")}</h3>
          <Button type="button" variant="outline" size="sm" onClick={onAddItem}>
            <Plus className="h-4 w-4 mr-1" />
            {t("form.addItem")}
          </Button>
        </div>
        {items.length > 0 ? (
          <div className="space-y-3">
            {items.map((item, index) => (
              <div key={item.id} className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-2 items-end">
                <div className="space-y-1">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemName")}</Label>}
                  <Input
                    placeholder={t("form.itemNamePlaceholder")}
                    value={item.name}
                    onChange={(e) => onUpdateItem(index, "name", e.target.value)}
                  />
                </div>
                <div className="space-y-1 w-28">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemSku")}</Label>}
                  <Input
                    placeholder="SKU"
                    value={item.sku}
                    onChange={(e) => onUpdateItem(index, "sku", e.target.value)}
                  />
                </div>
                <div className="space-y-1 w-20">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemQuantity")}</Label>}
                  <Input
                    type="number"
                    min="1"
                    step="1"
                    value={item.quantity}
                    onChange={(e) => onUpdateItem(index, "quantity", parseInt(e.target.value) || 1)}
                  />
                </div>
                <div className="space-y-1 w-28">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemPrice")}</Label>}
                  <Input
                    type="number"
                    min="0"
                    step="0.01"
                    value={item.price}
                    onChange={(e) => onUpdateItem(index, "price", parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div className="flex items-center gap-2">
                  {index === 0 && <Label className="text-xs text-muted-foreground invisible">X</Label>}
                  <span className="text-sm text-muted-foreground w-24 text-right">
                    {formatCurrency(item.quantity * item.price, currency)}
                  </span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => onRemoveItem(index)}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </div>
            ))}
            <div className="flex items-center justify-end gap-4 pt-2 border-t">
              <span className="text-sm font-medium">
                {t("form.itemsTotal", { total: formatCurrency(itemsTotal, currency) })}
              </span>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t("form.noItems")}</p>
        )}
      </div>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <FormField<OrderFormValues>
          name="total_amount"
          label={t("form.totalAmount")}
          error={errors.total_amount}
          required
        >
          {hasItems ? (
            <Input
              id="total_amount"
              type="number"
              step="0.01"
              min="0"
              value={totalAmountValue}
              readOnly
              disabled
              className="bg-muted"
            />
          ) : (
            <Input
              id="total_amount"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
              aria-invalid={!!errors.total_amount}
              {...register("total_amount", { valueAsNumber: true })}
            />
          )}
        </FormField>
        {hasItems && (
          <p className="text-xs text-muted-foreground md:col-span-2">
            {t("form.totalAutoCalculated")}
          </p>
        )}
      </div>
    </>
  );
}
