"use client";

import { useState, useCallback } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SHIPMENT_PROVIDERS } from "@/lib/constants";
import { OrderSearchCombobox } from "@/components/shared/order-search-combobox";
import { PaczkomatMap } from "@/components/shared/paczkomat-map";
import type { Shipment } from "@/types/api";

const shipmentSchema = z.object({
  order_id: z.string().min(1, "ID zamówienia jest wymagane"),
  provider: z.enum(["inpost", "dhl", "dpd", "gls", "ups", "poczta_polska", "orlen_paczka", "manual"], "Wybierz dostawcę"),
  tracking_number: z.string().optional(),
  label_url: z.string().optional(),
  carrier_data: z.record(z.string(), z.unknown()).optional(),
});

type ShipmentFormValues = z.infer<typeof shipmentSchema>;

interface ShipmentFormProps {
  defaultValues?: Partial<ShipmentFormValues>;
  shipment?: Shipment;
  onSubmit: (data: ShipmentFormValues) => void;
  isLoading?: boolean;
}

export function ShipmentForm({
  defaultValues,
  shipment,
  onSubmit,
  isLoading,
}: ShipmentFormProps) {
  const existingTargetPoint =
    (shipment?.carrier_data?.target_point as string | undefined) ??
    (defaultValues?.carrier_data?.target_point as string | undefined) ??
    "";

  const [targetPoint, setTargetPoint] = useState(existingTargetPoint);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<ShipmentFormValues>({
    resolver: zodResolver(shipmentSchema),
    defaultValues: {
      order_id: shipment?.order_id ?? defaultValues?.order_id ?? "",
      provider: shipment?.provider as ShipmentFormValues["provider"] ?? defaultValues?.provider ?? undefined,
      tracking_number: shipment?.tracking_number ?? defaultValues?.tracking_number ?? "",
      label_url: shipment?.label_url ?? defaultValues?.label_url ?? "",
      carrier_data: shipment?.carrier_data as Record<string, unknown> ?? defaultValues?.carrier_data ?? undefined,
    },
  });

  const providerValue = watch("provider");

  const handlePointSelect = useCallback(
    (pointName: string) => {
      setTargetPoint(pointName);
      const current = watch("carrier_data") ?? {};
      setValue("carrier_data", { ...current, target_point: pointName });
    },
    [setValue, watch]
  );

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="space-y-2">
        <Label>Zamówienie</Label>
        <OrderSearchCombobox
          value={watch("order_id")}
          onValueChange={(id) =>
            setValue("order_id", id, { shouldValidate: true })
          }
          disabled={!!shipment}
        />
        {errors.order_id && (
          <p className="text-sm text-destructive">{errors.order_id.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="provider">Dostawca</Label>
        <Select
          value={providerValue}
          onValueChange={(value) =>
            setValue("provider", value as ShipmentFormValues["provider"], {
              shouldValidate: true,
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Wybierz dostawcę" />
          </SelectTrigger>
          <SelectContent>
            {SHIPMENT_PROVIDERS.map((provider) => (
              <SelectItem key={provider} value={provider}>
                {provider.toUpperCase()}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.provider && (
          <p className="text-sm text-destructive">{errors.provider.message}</p>
        )}
      </div>

      {providerValue === "inpost" && (
        <div className="space-y-2">
          <Label>Paczkomat docelowy</Label>
          {targetPoint && (
            <div className="flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-2">
              <MapPin className="h-4 w-4 text-primary" />
              <span className="font-medium text-sm">{targetPoint}</span>
            </div>
          )}
          <PaczkomatMap
            onSelect={handlePointSelect}
            selectedPoint={targetPoint}
          />
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="tracking_number">Numer śledzenia</Label>
        <Input
          id="tracking_number"
          placeholder="Opcjonalny numer śledzenia"
          {...register("tracking_number")}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="label_url">URL etykiety</Label>
        <Input
          id="label_url"
          placeholder="Opcjonalny URL etykiety"
          {...register("label_url")}
        />
      </div>

      <Button type="submit" disabled={isLoading}>
        {isLoading ? "Zapisywanie..." : shipment ? "Zapisz zmiany" : "Utwórz przesyłkę"}
      </Button>
    </form>
  );
}
