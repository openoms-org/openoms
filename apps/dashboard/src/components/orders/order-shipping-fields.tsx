"use client";

import { useTranslations } from "next-intl";
import { PaczkomatSelector } from "@/components/shared/paczkomat-selector";
import { FormField } from "@/components/shared/form-wrapper";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SHIPMENT_PROVIDERS, SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import { getSelectableCarrierShipmentProviders } from "@/lib/readiness";
import type { AddressFields, OrderFormValues } from "./order-form.schema";

interface OrderShippingFieldsProps {
  shippingAddress: AddressFields;
  billingAddress: AddressFields;
  billingSameAsShipping: boolean;
  shipmentProvider: string;
  inpostServiceType: string;
  autoCreateShipment: boolean;
  pickupPointId: string;
  onShippingChange: (field: keyof AddressFields, value: string) => void;
  onBillingChange: (field: keyof AddressFields, value: string) => void;
  onBillingSameAsShippingChange: (same: boolean) => void;
  onShipmentProviderChange: (provider: string) => void;
  onInpostServiceTypeChange: (serviceType: string) => void;
  onAutoCreateShipmentChange: (enabled: boolean) => void;
  onPickupPointChange: (pointId: string) => void;
}

export function OrderShippingFields({
  shippingAddress,
  billingAddress,
  billingSameAsShipping,
  shipmentProvider,
  inpostServiceType,
  autoCreateShipment,
  pickupPointId,
  onShippingChange,
  onBillingChange,
  onBillingSameAsShippingChange,
  onShipmentProviderChange,
  onInpostServiceTypeChange,
  onAutoCreateShipmentChange,
  onPickupPointChange,
}: OrderShippingFieldsProps) {
  const t = useTranslations("orders");
  const selectableShipmentProviders = getSelectableCarrierShipmentProviders(
    SHIPMENT_PROVIDERS,
  );

  return (
    <>
      <div>
        <h3 className="text-sm font-medium mb-4">{t("form.shippingAddress")}</h3>
        <AddressEditor
          address={shippingAddress}
          nameLabel={t("form.fullName")}
          namePlaceholder={t("form.customerNamePlaceholder")}
          onChange={onShippingChange}
        />
      </div>

      <div>
        <h3 className="text-sm font-medium mb-4">{t("form.deliveryMethod")}</h3>
        <div className="space-y-4">
          <FormField<OrderFormValues> label={t("form.shipmentProvider")}>
            <Select
              value={shipmentProvider || "__none__"}
              onValueChange={(value) => {
                onShipmentProviderChange(value === "__none__" ? "" : value);
                if (value === "__none__") onAutoCreateShipmentChange(false);
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("form.shipmentProviderPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">{t("form.noShipment")}</SelectItem>
                {selectableShipmentProviders.map((provider) => (
                  <SelectItem key={provider} value={provider}>
                    {SHIPMENT_PROVIDER_LABELS[provider] ?? provider}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          {shipmentProvider === "inpost" && (
            <>
              <FormField<OrderFormValues> label={t("form.inpostServiceType")}>
                <Select
                  value={inpostServiceType}
                  onValueChange={(value) => {
                    onInpostServiceTypeChange(value);
                    if (value !== "locker") onPickupPointChange("");
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="locker">{t("form.inpostLocker")}</SelectItem>
                    <SelectItem value="courier">{t("form.inpostCourier")}</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>

              {inpostServiceType === "locker" && (
                <div className="space-y-2">
                  <Label>{t("form.targetLocker")}</Label>
                  {pickupPointId && (
                    <div className="flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-2 mb-2">
                      <span className="font-medium text-sm">{pickupPointId}</span>
                    </div>
                  )}
                  <PaczkomatSelector
                    mode="inline"
                    value={pickupPointId}
                    onPointSelect={onPickupPointChange}
                  />
                </div>
              )}
            </>
          )}

          {shipmentProvider && (
            <div className="flex items-center space-x-2">
              <Checkbox
                id="auto_create_shipment"
                checked={autoCreateShipment}
                onCheckedChange={(checked) => onAutoCreateShipmentChange(checked === true)}
              />
              <Label htmlFor="auto_create_shipment" className="font-normal cursor-pointer">
                {t("form.autoCreateShipment")}
              </Label>
            </div>
          )}
        </div>
      </div>

      <div>
        <div className="flex items-center gap-2 mb-4">
          <Checkbox
            id="billing_same"
            checked={billingSameAsShipping}
            onCheckedChange={(checked) => onBillingSameAsShippingChange(checked === true)}
          />
          <Label htmlFor="billing_same" className="cursor-pointer">
            {t("form.billingSameAsShipping")}
          </Label>
        </div>
        {!billingSameAsShipping && (
          <AddressEditor
            address={billingAddress}
            nameLabel={t("form.billingNameCompany")}
            namePlaceholder={t("form.customerNamePlaceholder")}
            onChange={onBillingChange}
          />
        )}
      </div>
    </>
  );
}

interface AddressEditorProps {
  address: AddressFields;
  nameLabel: string;
  namePlaceholder: string;
  onChange: (field: keyof AddressFields, value: string) => void;
}

function AddressEditor({ address, nameLabel, namePlaceholder, onChange }: AddressEditorProps) {
  const t = useTranslations("orders");

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <FormField<OrderFormValues> label={nameLabel}>
        <Input
          placeholder={namePlaceholder}
          value={address.name}
          onChange={(e) => onChange("name", e.target.value)}
        />
      </FormField>
      <FormField<OrderFormValues> label={t("form.street")}>
        <Input
          placeholder={t("form.streetPlaceholder")}
          value={address.street}
          onChange={(e) => onChange("street", e.target.value)}
        />
      </FormField>
      <FormField<OrderFormValues> label={t("form.city")}>
        <Input
          placeholder={t("form.cityPlaceholder")}
          value={address.city}
          onChange={(e) => onChange("city", e.target.value)}
        />
      </FormField>
      <FormField<OrderFormValues> label={t("form.postalCode")}>
        <Input
          placeholder={t("form.postalCodePlaceholder")}
          value={address.postal_code}
          onChange={(e) => onChange("postal_code", e.target.value)}
        />
      </FormField>
      <FormField<OrderFormValues> label={t("form.country")}>
        <Input
          placeholder="PL"
          value={address.country}
          onChange={(e) => onChange("country", e.target.value)}
        />
      </FormField>
    </div>
  );
}
