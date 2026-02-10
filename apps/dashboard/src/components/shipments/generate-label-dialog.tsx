"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useGenerateLabel } from "@/hooks/use-shipments";
import { CarrierFields } from "@/components/shipments/carrier-fields";
import { SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import type { Order, GenerateLabelRequest } from "@/types/api";

interface GenerateLabelDialogProps {
  shipmentId: string;
  provider: string;
  order: Order | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const DEFAULT_VALUES: Record<string, Record<string, any>> = {
  inpost: { service_type: "inpost_locker_standard", parcel_size: "small" },
  dhl: { service_type: "dhl_parcel" },
  dpd: { service_type: "dpd_classic" },
};

function getDialogTitle(provider: string): string {
  const label = SHIPMENT_PROVIDER_LABELS[provider];
  if (label) return `Generuj etykietę ${label}`;
  return `Generuj etykietę — ${provider.toUpperCase()}`;
}

export function GenerateLabelDialog({
  shipmentId,
  provider,
  order,
  open,
  onOpenChange,
}: GenerateLabelDialogProps) {
  const generateLabel = useGenerateLabel(shipmentId);

  const [carrierValues, setCarrierValues] = useState<Record<string, any>>(
    () => DEFAULT_VALUES[provider] ?? {}
  );
  const [labelFormat, setLabelFormat] = useState<string>("pdf");

  const handleFieldChange = (field: string, value: any) => {
    setCarrierValues((prev) => ({ ...prev, [field]: value }));
  };

  const isLocker =
    provider === "inpost" &&
    carrierValues.service_type === "inpost_locker_standard";
  const hasGeowidgetToken = !!process.env.NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN;
  const isSubmitDisabled =
    generateLabel.isPending ||
    (isLocker && !(carrierValues.target_point ?? "").trim());

  const handleSubmit = () => {
    const data: GenerateLabelRequest = {
      service_type: carrierValues.service_type ?? provider,
      label_format: labelFormat,
    };

    if (carrierValues.parcel_size) data.parcel_size = carrierValues.parcel_size;
    if (carrierValues.target_point) data.target_point = carrierValues.target_point.trim();
    if (carrierValues.weight_kg != null) data.weight_kg = carrierValues.weight_kg;
    if (carrierValues.width_cm != null) data.width_cm = carrierValues.width_cm;
    if (carrierValues.height_cm != null) data.height_cm = carrierValues.height_cm;
    if (carrierValues.depth_cm != null) data.depth_cm = carrierValues.depth_cm;
    if (carrierValues.cod_amount != null) data.cod_amount = carrierValues.cod_amount;
    if (carrierValues.insured_value != null) data.insured_value = carrierValues.insured_value;

    generateLabel.mutate(data, {
      onSuccess: () => {
        toast.success("Etykieta wygenerowana");
        onOpenChange(false);
      },
      onError: (error) => {
        toast.error(error.message || "Nie udało się wygenerować etykiety");
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={
          isLocker && hasGeowidgetToken ? "max-w-3xl" : ""
        }
      >
        <DialogHeader>
          <DialogTitle>{getDialogTitle(provider)}</DialogTitle>
          <DialogDescription>
            Wypełnij dane przesyłki, aby wygenerować etykietę.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Carrier-specific fields */}
          <CarrierFields
            provider={provider}
            values={carrierValues}
            onChange={handleFieldChange}
          />

          {/* Label format — shared across all carriers */}
          <div className="space-y-2">
            <Label>Format etykiety</Label>
            <Select
              value={labelFormat}
              onValueChange={setLabelFormat}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pdf">PDF</SelectItem>
                <SelectItem value="zpl">ZPL (drukarka termiczna)</SelectItem>
                <SelectItem value="epl">EPL (drukarka termiczna)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Receiver preview */}
          {order && (
            <div className="space-y-2 rounded-md border p-3">
              <Label>Odbiorca</Label>
              <div className="space-y-1 text-sm">
                <p>
                  <span className="text-muted-foreground">Imię i nazwisko: </span>
                  {order.customer_name}
                </p>
                <p>
                  <span className="text-muted-foreground">Telefon: </span>
                  {order.customer_phone ?? "-"}
                </p>
                <p>
                  <span className="text-muted-foreground">Email: </span>
                  {order.customer_email ?? "-"}
                </p>
              </div>
              <p className="text-xs text-muted-foreground">
                Dane odbiorcy pobrane z zamówienia
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitDisabled}
          >
            {generateLabel.isPending && (
              <Loader2 className="h-4 w-4 animate-spin" />
            )}
            Generuj etykietę
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
