"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
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
import type { Shipment } from "@/types/api";

const shipmentSchema = z.object({
  order_id: z.string().min(1, "ID zamowienia jest wymagane"),
  provider: z.enum(["inpost", "dhl", "dpd", "manual"], "Wybierz dostawce"),
  tracking_number: z.string().optional(),
  label_url: z.string().optional(),
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
    },
  });

  const providerValue = watch("provider");

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="order_id">ID zamowienia</Label>
        <Input
          id="order_id"
          placeholder="UUID zamowienia"
          {...register("order_id")}
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
            <SelectValue placeholder="Wybierz dostawce" />
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

      <div className="space-y-2">
        <Label htmlFor="tracking_number">Numer sledzenia</Label>
        <Input
          id="tracking_number"
          placeholder="Opcjonalny numer sledzenia"
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
        {isLoading ? "Zapisywanie..." : shipment ? "Zapisz zmiany" : "Utworz przesylke"}
      </Button>
    </form>
  );
}
