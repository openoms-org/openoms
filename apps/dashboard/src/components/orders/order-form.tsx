"use client";

import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Order, CreateOrderRequest } from "@/types/api";

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

  const handleFormSubmit = (data: OrderFormValues) => {
    onSubmit({
      source: data.source,
      customer_name: data.customer_name,
      customer_email: data.customer_email || undefined,
      customer_phone: data.customer_phone || undefined,
      total_amount: data.total_amount,
      currency: data.currency,
      notes: data.notes || undefined,
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
