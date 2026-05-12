"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { FormWrapper } from "@/components/shared/form-wrapper";
import { Separator } from "@/components/ui/separator";
import { useCustomFields } from "@/hooks/use-custom-fields";
import { SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import type { Order, CreateOrderRequest, Address } from "@/types/api";
import { OrderCustomerFields } from "./order-customer-fields";
import { OrderItemsEditor } from "./order-items-editor";
import { OrderMetadataFields } from "./order-metadata-fields";
import { OrderPaymentFields } from "./order-payment-fields";
import { OrderShippingFields } from "./order-shipping-fields";
import {
  createOrderSchema,
  emptyAddress,
  type AddressFields,
  type OrderFormValues,
  type OrderItemRow,
  type OrderPriority,
} from "./order-form.schema";

interface OrderFormProps {
  order?: Order;
  onSubmit: (data: CreateOrderRequest) => void;
  isSubmitting?: boolean;
  onCancel?: () => void;
}

function parseAddress(addr: Address | undefined): AddressFields {
  if (!addr) return { ...emptyAddress };
  return {
    name: (addr.name as string) || "",
    street: (addr.street as string) || "",
    city: (addr.city as string) || "",
    postal_code: (addr.postal_code as string) || "",
    country: (addr.country as string) || "PL",
  };
}

function parseItems(items: Order["items"]): OrderItemRow[] {
  if (!items || items.length === 0) return [];
  return items.map((item, index) => ({
    id: `existing-${item.sku || item.name || "item"}-${index}`,
    name: item.name || "",
    sku: item.sku || "",
    quantity: item.quantity || 1,
    price: item.price || 0,
  }));
}

function normalizeOrderPriority(priority: string | undefined): OrderPriority {
  if (priority === "urgent" || priority === "high" || priority === "normal" || priority === "low") {
    return priority;
  }
  return "normal";
}

function createOrderItemRow(): OrderItemRow {
  const id = typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `new-${Date.now()}-${Math.random().toString(36).slice(2)}`;

  return { id, name: "", sku: "", quantity: 1, price: 0 };
}

function mapDeliveryMethodToProvider(deliveryMethod: string | undefined): string {
  if (!deliveryMethod) return "";
  if (deliveryMethod === "Paczkomat InPost" || deliveryMethod === "Kurier InPost") {
    return "inpost";
  }
  return Object.entries(SHIPMENT_PROVIDER_LABELS).find(([, label]) => label === deliveryMethod)?.[0] ?? "";
}

export function OrderForm({ order, onSubmit, isSubmitting = false, onCancel }: OrderFormProps) {
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const { data: customFieldsConfig } = useCustomFields();
  const [customValues, setCustomValues] = useState<Record<string, unknown>>(() =>
    order?.metadata && typeof order.metadata === "object" ? { ...order.metadata } : {}
  );
  const [tags, setTags] = useState<string[]>(order?.tags || []);
  const [orderItems, setOrderItems] = useState<OrderItemRow[]>(parseItems(order?.items));
  const [shippingAddress, setShippingAddress] = useState<AddressFields>(
    parseAddress(order?.shipping_address)
  );
  const [billingAddress, setBillingAddress] = useState<AddressFields>(
    parseAddress(order?.billing_address)
  );
  const [billingSameAsShipping, setBillingSameAsShipping] = useState(
    !order?.billing_address || Object.keys(order.billing_address).length === 0
  );
  const [paymentStatus, setPaymentStatus] = useState(order?.payment_status || "pending");
  const [paymentMethod, setPaymentMethod] = useState(order?.payment_method || "");
  const [priority, setPriority] = useState<OrderPriority>(() => normalizeOrderPriority(order?.priority));
  const [internalNotes, setInternalNotes] = useState(order?.internal_notes || "");
  const [shipmentProvider, setShipmentProvider] = useState<string>(() =>
    mapDeliveryMethodToProvider(order?.delivery_method)
  );
  const [inpostServiceType, setInpostServiceType] = useState<string>(
    order?.delivery_method === "Kurier InPost" ? "courier" : "locker"
  );
  const [autoCreateShipment, setAutoCreateShipment] = useState(false);
  const [pickupPointId, setPickupPointId] = useState(order?.pickup_point_id || "");
  const orderSchema = useMemo(() => createOrderSchema(t), [t]);
  const customFields = customFieldsConfig?.fields || [];

  const handleCustomFieldChange = (key: string, value: unknown) => {
    setCustomValues((prev) => ({ ...prev, [key]: value }));
  };

  const addItem = () => {
    setOrderItems([...orderItems, createOrderItemRow()]);
  };

  const removeItem = (index: number) => {
    setOrderItems(orderItems.filter((_, i) => i !== index));
  };

  const updateItem = (index: number, field: keyof OrderItemRow, value: string | number) => {
    setOrderItems(
      orderItems.map((item, i) =>
        i === index ? { ...item, [field]: value } : item
      )
    );
  };

  const updateShipping = (field: keyof AddressFields, value: string) => {
    setShippingAddress((prev) => ({ ...prev, [field]: value }));
  };

  const updateBilling = (field: keyof AddressFields, value: string) => {
    setBillingAddress((prev) => ({ ...prev, [field]: value }));
  };

  const handleFormSubmit = (data: OrderFormValues) => {
    for (const field of customFields) {
      if (field.required) {
        const value = customValues[field.key];
        if (value === undefined || value === null || value === "" || value === false) {
          toast.error(t("form.fieldRequired", { field: field.label }));
          return;
        }
      }
    }

    const metadata: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(customValues)) {
      if (value !== "" && value !== undefined && value !== null) {
        metadata[key] = value;
      }
    }

    const hasShipping = shippingAddress.street.trim() !== "";
    const hasBilling = !billingSameAsShipping && billingAddress.street.trim() !== "";
    const items = orderItems
      .filter((item) => item.name.trim() !== "")
      .map(({ id: _id, ...item }) => item);

    onSubmit({
      source: data.source,
      customer_name: data.customer_name,
      customer_email: data.customer_email || undefined,
      customer_phone: data.customer_phone || undefined,
      total_amount: data.total_amount,
      currency: data.currency,
      notes: data.notes || undefined,
      metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
      tags: tags.length > 0 ? tags : undefined,
      items: items.length > 0 ? items : undefined,
      shipping_address: hasShipping ? shippingAddress : undefined,
      billing_address: hasBilling ? billingAddress : undefined,
      payment_status: paymentStatus || undefined,
      payment_method: paymentMethod || undefined,
      priority,
      internal_notes: internalNotes || undefined,
      delivery_method: shipmentProvider
        ? shipmentProvider === "inpost"
          ? inpostServiceType === "locker" ? "Paczkomat InPost" : "Kurier InPost"
          : (SHIPMENT_PROVIDER_LABELS[shipmentProvider] ?? shipmentProvider)
        : undefined,
      pickup_point_id: pickupPointId || undefined,
      shipment_provider: autoCreateShipment && shipmentProvider ? shipmentProvider : undefined,
      auto_create_shipment: autoCreateShipment && !!shipmentProvider,
    });
  };

  return (
    <FormWrapper<OrderFormValues>
      schema={orderSchema}
      defaultValues={{
        source: order?.source || "manual",
        customer_name: order?.customer_name || "",
        customer_email: order?.customer_email || "",
        customer_phone: order?.customer_phone || "",
        total_amount: order?.total_amount || 0,
        currency: order?.currency || "PLN",
        notes: order?.notes || "",
      }}
      onSubmit={handleFormSubmit}
      className="space-y-6"
      submitLabel={order ? t("form.submitUpdate") : t("form.submitCreate")}
      submittingLabel={tc("saving")}
      isSubmitting={isSubmitting}
      onCancel={onCancel}
      cancelLabel={tc("cancel")}
      showErrorSummary={false}
    >
      {({ register, setValue, watch, formState: { errors } }) => {
        const [currentSource, currencyValue, totalAmountValue] = watch([
          "source",
          "currency",
          "total_amount",
        ]);
        const currency = currencyValue || "PLN";

        return (
          <>
            <OrderCustomerFields
              register={register}
              setValue={setValue}
              errors={errors}
              currentSource={currentSource}
              currency={currency}
            />

            <Separator />
            <OrderItemsEditor
              items={orderItems}
              currency={currency}
              totalAmountValue={totalAmountValue}
              errors={errors}
              register={register}
              setValue={setValue}
              onAddItem={addItem}
              onRemoveItem={removeItem}
              onUpdateItem={updateItem}
            />

            <Separator />
            <OrderPaymentFields
              paymentStatus={paymentStatus}
              paymentMethod={paymentMethod}
              onPaymentStatusChange={setPaymentStatus}
              onPaymentMethodChange={setPaymentMethod}
            />

            <Separator />
            <OrderShippingFields
              shippingAddress={shippingAddress}
              billingAddress={billingAddress}
              billingSameAsShipping={billingSameAsShipping}
              shipmentProvider={shipmentProvider}
              inpostServiceType={inpostServiceType}
              autoCreateShipment={autoCreateShipment}
              pickupPointId={pickupPointId}
              onShippingChange={updateShipping}
              onBillingChange={updateBilling}
              onBillingSameAsShippingChange={setBillingSameAsShipping}
              onShipmentProviderChange={setShipmentProvider}
              onInpostServiceTypeChange={setInpostServiceType}
              onAutoCreateShipmentChange={setAutoCreateShipment}
              onPickupPointChange={setPickupPointId}
            />

            <OrderMetadataFields
              register={register}
              errors={errors}
              customFields={customFields}
              customValues={customValues}
              tags={tags}
              priority={priority}
              internalNotes={internalNotes}
              onCustomFieldChange={handleCustomFieldChange}
              onTagsChange={setTags}
              onPriorityChange={setPriority}
              onInternalNotesChange={setInternalNotes}
            />
          </>
        );
      }}
    </FormWrapper>
  );
}
