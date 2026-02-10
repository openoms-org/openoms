"use client";

import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { useCustomFields } from "@/hooks/use-custom-fields";
import { TagInput } from "@/components/shared/tag-input";
import type { Order, CreateOrderRequest, CustomFieldDef } from "@/types/api";

const orderSchema = z.object({
  source: z.string().min(1, "Zrodlo jest wymagane"),
  customer_name: z.string().min(1, "Nazwa klienta jest wymagana"),
  customer_email: z.string().email("Nieprawidlowy adres email").optional().or(z.literal("")),
  customer_phone: z.string().optional(),
  total_amount: z.number().min(0, "Kwota musi byc >= 0"),
  currency: z.string().min(1, "Waluta jest wymagana"),
  notes: z.string().optional(),
});

type OrderFormValues = z.infer<typeof orderSchema>;

interface OrderFormProps {
  order?: Order;
  onSubmit: (data: CreateOrderRequest) => void;
  isSubmitting?: boolean;
  onCancel?: () => void;
}

export function OrderForm({ order, onSubmit, isSubmitting = false, onCancel }: OrderFormProps) {
  const { data: customFieldsConfig } = useCustomFields();
  const [customValues, setCustomValues] = useState<Record<string, unknown>>({});
  const [tags, setTags] = useState<string[]>(order?.tags || []);

  useEffect(() => {
    if (order?.metadata && typeof order.metadata === "object") {
      setCustomValues({ ...order.metadata });
    }
  }, [order?.metadata]);

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

  const customFields = customFieldsConfig?.fields || [];

  const handleCustomFieldChange = (key: string, value: unknown) => {
    setCustomValues((prev) => ({ ...prev, [key]: value }));
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
              <SelectValue placeholder="Wybierz..." />
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
    const metadata: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(customValues)) {
      if (value !== "" && value !== undefined && value !== null) {
        metadata[key] = value;
      }
    }

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
    });
  };

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="source">Zrodlo</Label>
          <Select
            value={currentSource}
            onValueChange={(value) => setValue("source", value)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Wybierz zrodlo" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="manual">Reczne</SelectItem>
              <SelectItem value="allegro">Allegro</SelectItem>
              <SelectItem value="woocommerce">WooCommerce</SelectItem>
            </SelectContent>
          </Select>
          {errors.source && (
            <p className="text-sm text-destructive">{errors.source.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="customer_name">Nazwa klienta</Label>
          <Input
            id="customer_name"
            placeholder="Jan Kowalski"
            {...register("customer_name")}
          />
          {errors.customer_name && (
            <p className="text-sm text-destructive">{errors.customer_name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="customer_email">Email klienta</Label>
          <Input
            id="customer_email"
            type="email"
            placeholder="jan@example.com"
            {...register("customer_email")}
          />
          {errors.customer_email && (
            <p className="text-sm text-destructive">{errors.customer_email.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="customer_phone">Telefon klienta</Label>
          <Input
            id="customer_phone"
            placeholder="+48 123 456 789"
            {...register("customer_phone")}
          />
          {errors.customer_phone && (
            <p className="text-sm text-destructive">{errors.customer_phone.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="total_amount">Kwota</Label>
          <Input
            id="total_amount"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            {...register("total_amount", { valueAsNumber: true })}
          />
          {errors.total_amount && (
            <p className="text-sm text-destructive">{errors.total_amount.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="currency">Waluta</Label>
          <Input
            id="currency"
            placeholder="PLN"
            {...register("currency")}
          />
          {errors.currency && (
            <p className="text-sm text-destructive">{errors.currency.message}</p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="notes">Notatki</Label>
        <Textarea
          id="notes"
          placeholder="Dodatkowe uwagi do zamowienia..."
          rows={3}
          {...register("notes")}
        />
        {errors.notes && (
          <p className="text-sm text-destructive">{errors.notes.message}</p>
        )}
      </div>

      {customFields.length > 0 && (
        <>
          <Separator />
          <div>
            <h3 className="text-sm font-medium mb-4">Pola dodatkowe</h3>
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

      <div className="space-y-2">
        <Label>Tagi</Label>
        <TagInput tags={tags} onChange={setTags} />
      </div>

      <div className="flex items-center gap-4">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting
            ? "Zapisywanie..."
            : order
              ? "Zapisz zmiany"
              : "Utworz zamowienie"}
        </Button>
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            Anuluj
          </Button>
        )}
      </div>
    </form>
  );
}
