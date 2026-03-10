"use client";

import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { useCustomFields } from "@/hooks/use-custom-fields";
import { TagInput } from "@/components/shared/tag-input";
import { PaczkomatSelector } from "@/components/shared/paczkomat-selector";
import { PAYMENT_METHODS, ORDER_PRIORITIES, SHIPMENT_PROVIDERS, SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import { formatCurrency } from "@/lib/utils";
import type { Order, CreateOrderRequest, CustomFieldDef, Address } from "@/types/api";

interface OrderItemRow {
  name: string;
  sku: string;
  quantity: number;
  price: number;
}

interface AddressFields {
  name: string;
  street: string;
  city: string;
  postal_code: string;
  country: string;
}

const emptyAddress: AddressFields = {
  name: "",
  street: "",
  city: "",
  postal_code: "",
  country: "PL",
};

function createOrderSchema(t: (key: string) => string) {
  return z.object({
    source: z.string().min(1, t("validation.sourceRequired")),
    customer_name: z.string().min(1, t("validation.customerNameRequired")),
    customer_email: z.string().email(t("validation.invalidEmail")).optional().or(z.literal("")),
    customer_phone: z.string().optional(),
    total_amount: z.number().min(0, t("validation.amountMin")),
    currency: z.string().min(1, t("validation.currencyRequired")),
    notes: z.string().optional(),
  });
}

type OrderFormValues = z.infer<ReturnType<typeof createOrderSchema>>;

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
  return items.map((item) => ({
    name: item.name || "",
    sku: item.sku || "",
    quantity: item.quantity || 1,
    price: item.price || 0,
  }));
}

export function OrderForm({ order, onSubmit, isSubmitting = false, onCancel }: OrderFormProps) {
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const { data: customFieldsConfig } = useCustomFields();
  const [customValues, setCustomValues] = useState<Record<string, unknown>>({});
  const [tags, setTags] = useState<string[]>(order?.tags || []);

  const [orderItems, setOrderItems] = useState<OrderItemRow[]>(
    parseItems(order?.items)
  );

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
  const [priority, setPriority] = useState<string>(order?.priority || "normal");
  const [internalNotes, setInternalNotes] = useState(order?.internal_notes || "");

  const [shipmentProvider, setShipmentProvider] = useState<string>(order?.delivery_method ? "" : "");
  const [inpostServiceType, setInpostServiceType] = useState<string>("locker");
  const [autoCreateShipment, setAutoCreateShipment] = useState(false);
  const [pickupPointId, setPickupPointId] = useState(order?.pickup_point_id || "");

  useEffect(() => {
    if (order?.metadata && typeof order.metadata === "object") {
      setCustomValues({ ...order.metadata });
    }
  }, [order?.metadata]);

  const orderSchema = createOrderSchema(t);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<OrderFormValues>({
    resolver: zodResolver(orderSchema),
    defaultValues: {
      source: order?.source || "manual",
      customer_name: order?.customer_name || "",
      customer_email: order?.customer_email || "",
      customer_phone: order?.customer_phone || "",
      total_amount: order?.total_amount || 0,
      currency: order?.currency || "PLN",
      notes: order?.notes || "",
    },
  });

  const currentSource = watch("source");
  const currency = watch("currency") || "PLN";

  const customFields = customFieldsConfig?.fields || [];

  const handleCustomFieldChange = (key: string, value: unknown) => {
    setCustomValues((prev) => ({ ...prev, [key]: value }));
  };

  // Order items
  const addItem = () => {
    setOrderItems([...orderItems, { name: "", sku: "", quantity: 1, price: 0 }]);
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

  const itemsTotal = orderItems.reduce(
    (sum, item) => sum + item.quantity * item.price,
    0
  );

  const hasItems = orderItems.length > 0;

  // Auto-calculate total when items change
  useEffect(() => {
    if (hasItems) {
      setValue("total_amount", Math.round(itemsTotal * 100) / 100);
    }
  }, [itemsTotal, hasItems, setValue]);

  // Shipping address handlers
  const updateShipping = (field: keyof AddressFields, value: string) => {
    setShippingAddress((prev) => ({ ...prev, [field]: value }));
  };

  const updateBilling = (field: keyof AddressFields, value: string) => {
    setBillingAddress((prev) => ({ ...prev, [field]: value }));
  };

  const renderCustomField = (field: CustomFieldDef) => {
    switch (field.type) {
      case "text":
        return (
          <Input
            id={`cf_${field.key}`}
            value={(customValues[field.key] as string) || ""}
            onChange={(e) => handleCustomFieldChange(field.key, e.target.value)}
          />
        );
      case "number":
        return (
          <Input
            id={`cf_${field.key}`}
            type="number"
            step="any"
            value={(customValues[field.key] as string | number) ?? ""}
            onChange={(e) =>
              handleCustomFieldChange(
                field.key,
                e.target.value === "" ? "" : Number(e.target.value)
              )
            }
          />
        );
      case "select":
        return (
          <Select
            value={(customValues[field.key] as string) || ""}
            onValueChange={(v) => handleCustomFieldChange(field.key, v)}
          >
            <SelectTrigger>
              <SelectValue placeholder={tc("selectPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {(field.options || []).map((opt) => (
                <SelectItem key={opt} value={opt}>
                  {opt}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );
      case "date":
        return (
          <Input
            id={`cf_${field.key}`}
            type="date"
            value={(customValues[field.key] as string) || ""}
            onChange={(e) => handleCustomFieldChange(field.key, e.target.value)}
          />
        );
      case "checkbox":
        return (
          <div className="flex items-center gap-2 pt-1">
            <input
              id={`cf_${field.key}`}
              type="checkbox"
              checked={!!customValues[field.key]}
              onChange={(e) => handleCustomFieldChange(field.key, e.target.checked)}
              className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
            />
          </div>
        );
      default:
        return null;
    }
  };

  const handleFormSubmit = (data: OrderFormValues) => {
    // Validate required custom fields
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

    const items = orderItems.filter((item) => item.name.trim() !== "");

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
      priority: priority as "urgent" | "high" | "normal" | "low",
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
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      {/* Basic info */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="source">{t("form.source")} <span className="text-destructive">*</span></Label>
          <Select
            value={currentSource}
            onValueChange={(value) => setValue("source", value)}
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
          {errors.source && (
            <p className="text-destructive text-xs mt-1">{errors.source.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="customer_name">{t("form.customerName")} <span className="text-destructive">*</span></Label>
          <Input
            id="customer_name"
            placeholder={t("form.customerNamePlaceholder")}
            aria-invalid={!!errors.customer_name}
            {...register("customer_name")}
          />
          {errors.customer_name && (
            <p className="text-destructive text-xs mt-1">{errors.customer_name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="customer_email">{t("form.customerEmail")}</Label>
          <Input
            id="customer_email"
            type="email"
            placeholder="jan@example.com"
            aria-invalid={!!errors.customer_email}
            {...register("customer_email")}
          />
          {errors.customer_email && (
            <p className="text-destructive text-xs mt-1">{errors.customer_email.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="customer_phone">{t("form.customerPhone")}</Label>
          <Input
            id="customer_phone"
            placeholder="+48 123 456 789"
            {...register("customer_phone")}
          />
          {errors.customer_phone && (
            <p className="text-destructive text-xs mt-1">{errors.customer_phone.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="currency">{t("form.currency")} <span className="text-destructive">*</span></Label>
          <Input
            id="currency"
            value={currency}
            readOnly
            disabled
            className="bg-muted"
          />
        </div>
      </div>

      {/* Order items */}
      <Separator />
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium">{t("form.orderItems")}</h3>
          <Button type="button" variant="outline" size="sm" onClick={addItem}>
            <Plus className="h-4 w-4 mr-1" />
            {t("form.addItem")}
          </Button>
        </div>
        {orderItems.length > 0 ? (
          <div className="space-y-3">
            {orderItems.map((item, index) => (
              <div key={index} className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-2 items-end">
                <div className="space-y-1">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemName")}</Label>}
                  <Input
                    placeholder={t("form.itemNamePlaceholder")}
                    value={item.name}
                    onChange={(e) => updateItem(index, "name", e.target.value)}
                  />
                </div>
                <div className="space-y-1 w-28">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemSku")}</Label>}
                  <Input
                    placeholder="SKU"
                    value={item.sku}
                    onChange={(e) => updateItem(index, "sku", e.target.value)}
                  />
                </div>
                <div className="space-y-1 w-20">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemQuantity")}</Label>}
                  <Input
                    type="number"
                    min="1"
                    step="1"
                    value={item.quantity}
                    onChange={(e) => updateItem(index, "quantity", parseInt(e.target.value) || 1)}
                  />
                </div>
                <div className="space-y-1 w-28">
                  {index === 0 && <Label className="text-xs text-muted-foreground">{t("form.itemPrice")}</Label>}
                  <Input
                    type="number"
                    min="0"
                    step="0.01"
                    value={item.price}
                    onChange={(e) => updateItem(index, "price", parseFloat(e.target.value) || 0)}
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
                    onClick={() => removeItem(index)}
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
          <p className="text-sm text-muted-foreground">
            {t("form.noItems")}
          </p>
        )}
      </div>

      {/* Total amount */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="total_amount">{t("form.totalAmount")} <span className="text-destructive">*</span></Label>
          {hasItems ? (
            <Input
              id="total_amount"
              type="number"
              step="0.01"
              min="0"
              value={watch("total_amount")}
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
          {hasItems && (
            <p className="text-xs text-muted-foreground">
              {t("form.totalAutoCalculated")}
            </p>
          )}
          {errors.total_amount && (
            <p className="text-destructive text-xs mt-1">{errors.total_amount.message}</p>
          )}
        </div>
      </div>

      {/* Payment */}
      <Separator />
      <div>
        <h3 className="text-sm font-medium mb-4">{t("form.payment")}</h3>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t("form.paymentStatus")}</Label>
            <Select value={paymentStatus} onValueChange={setPaymentStatus}>
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
          </div>
          <div className="space-y-2">
            <Label>{t("form.paymentMethod")}</Label>
            <Select value={paymentMethod || "__none__"} onValueChange={(v) => setPaymentMethod(v === "__none__" ? "" : v)}>
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
          </div>
        </div>
      </div>

      {/* Shipping address */}
      <Separator />
      <div>
        <h3 className="text-sm font-medium mb-4">{t("form.shippingAddress")}</h3>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t("form.fullName")}</Label>
            <Input
              placeholder={t("form.customerNamePlaceholder")}
              value={shippingAddress.name}
              onChange={(e) => updateShipping("name", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("form.street")}</Label>
            <Input
              placeholder={t("form.streetPlaceholder")}
              value={shippingAddress.street}
              onChange={(e) => updateShipping("street", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("form.city")}</Label>
            <Input
              placeholder={t("form.cityPlaceholder")}
              value={shippingAddress.city}
              onChange={(e) => updateShipping("city", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("form.postalCode")}</Label>
            <Input
              placeholder={t("form.postalCodePlaceholder")}
              value={shippingAddress.postal_code}
              onChange={(e) => updateShipping("postal_code", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("form.country")}</Label>
            <Input
              placeholder="PL"
              value={shippingAddress.country}
              onChange={(e) => updateShipping("country", e.target.value)}
            />
          </div>
        </div>
      </div>

      {/* Delivery method / Carrier selection */}
      <Separator />
      <div>
        <h3 className="text-sm font-medium mb-4">{t("form.deliveryMethod")}</h3>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>{t("form.shipmentProvider")}</Label>
            <Select
              value={shipmentProvider || "__none__"}
              onValueChange={(v) => {
                setShipmentProvider(v === "__none__" ? "" : v);
                if (v === "__none__") setAutoCreateShipment(false);
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("form.shipmentProviderPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">{t("form.noShipment")}</SelectItem>
                {SHIPMENT_PROVIDERS.filter(p => p !== "manual").map((p) => (
                  <SelectItem key={p} value={p}>
                    {SHIPMENT_PROVIDER_LABELS[p] ?? p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {shipmentProvider === "inpost" && (
            <>
              <div className="space-y-2">
                <Label>{t("form.inpostServiceType")}</Label>
                <Select
                  value={inpostServiceType}
                  onValueChange={(v) => {
                    setInpostServiceType(v);
                    if (v !== "locker") setPickupPointId("");
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
              </div>

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
                    onPointSelect={(name) => setPickupPointId(name)}
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
                onCheckedChange={(checked) => setAutoCreateShipment(checked === true)}
              />
              <Label htmlFor="auto_create_shipment" className="font-normal cursor-pointer">
                {t("form.autoCreateShipment")}
              </Label>
            </div>
          )}
        </div>
      </div>

      {/* Billing address */}
      <Separator />
      <div>
        <div className="flex items-center gap-2 mb-4">
          <input
            id="billing_same"
            type="checkbox"
            checked={billingSameAsShipping}
            onChange={(e) => setBillingSameAsShipping(e.target.checked)}
            className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
          />
          <Label htmlFor="billing_same" className="cursor-pointer">
            {t("form.billingSameAsShipping")}
          </Label>
        </div>
        {!billingSameAsShipping && (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("form.billingNameCompany")}</Label>
              <Input
                placeholder={t("form.customerNamePlaceholder")}
                value={billingAddress.name}
                onChange={(e) => updateBilling("name", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("form.street")}</Label>
              <Input
                placeholder={t("form.streetPlaceholder")}
                value={billingAddress.street}
                onChange={(e) => updateBilling("street", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("form.city")}</Label>
              <Input
                placeholder={t("form.cityPlaceholder")}
                value={billingAddress.city}
                onChange={(e) => updateBilling("city", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("form.postalCode")}</Label>
              <Input
                placeholder={t("form.postalCodePlaceholder")}
                value={billingAddress.postal_code}
                onChange={(e) => updateBilling("postal_code", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("form.country")}</Label>
              <Input
                placeholder="PL"
                value={billingAddress.country}
                onChange={(e) => updateBilling("country", e.target.value)}
              />
            </div>
          </div>
        )}
      </div>

      {/* Notes */}
      <div className="space-y-2">
        <Label htmlFor="notes">{tc("notes")}</Label>
        <Textarea
          id="notes"
          placeholder={t("form.notesPlaceholder")}
          rows={3}
          {...register("notes")}
        />
        {errors.notes && (
          <p className="text-destructive text-xs mt-1">{errors.notes.message}</p>
        )}
      </div>

      {/* Priority */}
      <div className="space-y-2">
        <Label>{t("form.priority")}</Label>
        <Select value={priority} onValueChange={setPriority}>
          <SelectTrigger>
            <SelectValue placeholder={t("form.priorityPlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(ORDER_PRIORITIES).map(([key, { label }]) => (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Internal notes */}
      <div className="space-y-2">
        <Label htmlFor="internal_notes">{t("form.internalNotes")}</Label>
        <Textarea
          id="internal_notes"
          placeholder={t("form.internalNotesPlaceholder")}
          rows={3}
          value={internalNotes}
          onChange={(e) => setInternalNotes(e.target.value)}
          className="border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-950/30"
        />
      </div>

      {/* Custom fields */}
      {customFields.length > 0 && (
        <>
          <Separator />
          <div>
            <h3 className="text-sm font-medium mb-4">{t("form.customFields")}</h3>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {[...customFields]
                .sort((a, b) => a.position - b.position)
                .map((field) => (
                  <div key={field.key} className="space-y-2">
                    <Label htmlFor={`cf_${field.key}`}>
                      {field.label}
                      {field.required && <span className="text-destructive ml-1">*</span>}
                    </Label>
                    {renderCustomField(field)}
                  </div>
                ))}
            </div>
          </div>
        </>
      )}

      {/* Tags */}
      <div className="space-y-2">
        <Label>{tc("tags")}</Label>
        <TagInput tags={tags} onChange={setTags} />
      </div>

      {/* Actions */}
      <div className="flex items-center gap-4">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting
            ? tc("saving")
            : order
              ? t("form.submitUpdate")
              : t("form.submitCreate")}
        </Button>
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            {tc("cancel")}
          </Button>
        )}
      </div>
    </form>
  );
}
