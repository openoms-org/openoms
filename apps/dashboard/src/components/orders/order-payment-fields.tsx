"use client";

import { useTranslations } from "next-intl";
import { FormField } from "@/components/shared/form-wrapper";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PAYMENT_METHODS } from "@/lib/constants";
import type { OrderFormValues } from "./order-form.schema";

interface OrderPaymentFieldsProps {
  paymentStatus: string;
  paymentMethod: string;
  onPaymentStatusChange: (status: string) => void;
  onPaymentMethodChange: (method: string) => void;
}

export function OrderPaymentFields({
  paymentStatus,
  paymentMethod,
  onPaymentStatusChange,
  onPaymentMethodChange,
}: OrderPaymentFieldsProps) {
  const t = useTranslations("orders");

  return (
    <div>
      <h3 className="text-sm font-medium mb-4">{t("form.payment")}</h3>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <FormField<OrderFormValues> label={t("form.paymentStatus")}>
          <Select value={paymentStatus} onValueChange={onPaymentStatusChange}>
            <SelectTrigger>
              <SelectValue placeholder={t("form.paymentStatusPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pending">{t("form.paymentPending")}</SelectItem>
              <SelectItem value="paid">{t("form.paymentPaid")}</SelectItem>
              <SelectItem value="refunded">{t("form.paymentRefunded")}</SelectItem>
              <SelectItem value="failed">{t("form.paymentFailed")}</SelectItem>
            </SelectContent>
          </Select>
        </FormField>

        <FormField<OrderFormValues> label={t("form.paymentMethod")}>
          <Select
            value={paymentMethod || "__none__"}
            onValueChange={(value) => onPaymentMethodChange(value === "__none__" ? "" : value)}
          >
            <SelectTrigger>
              <SelectValue placeholder={t("form.paymentMethodPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">{t("form.notSelected")}</SelectItem>
              {PAYMENT_METHODS.map((method) => (
                <SelectItem key={method} value={method}>
                  {method}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FormField>
      </div>
    </div>
  );
}
