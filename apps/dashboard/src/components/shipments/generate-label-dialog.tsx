"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useGenerateLabel } from "@/hooks/use-shipments";
import type { Order, GenerateLabelRequest } from "@/types/api";

interface GenerateLabelDialogProps {
  shipmentId: string;
  order: Order | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function GenerateLabelDialog({
  shipmentId,
  order,
  open,
  onOpenChange,
}: GenerateLabelDialogProps) {
  const generateLabel = useGenerateLabel(shipmentId);

  const [serviceType, setServiceType] = useState<
    "inpost_locker_standard" | "inpost_courier_standard"
  >("inpost_locker_standard");
  const [targetPoint, setTargetPoint] = useState("");
  const [parcelSize, setParcelSize] = useState<"small" | "medium" | "large">(
    "small"
  );
  const [labelFormat, setLabelFormat] = useState<"pdf" | "zpl" | "epl">("pdf");

  const isLocker = serviceType === "inpost_locker_standard";
  const isSubmitDisabled =
    generateLabel.isPending || (isLocker && targetPoint.trim() === "");

  const handleSubmit = () => {
    const data: GenerateLabelRequest = {
      service_type: serviceType,
      parcel_size: parcelSize,
      label_format: labelFormat,
    };

    if (isLocker) {
      data.target_point = targetPoint.trim();
    }

    generateLabel.mutate(data, {
      onSuccess: () => {
        toast.success("Etykieta wygenerowana");
        onOpenChange(false);
      },
      onError: (error) => {
        toast.error(error.message || "Nie udalo sie wygenerowac etykiety");
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generuj etykiete InPost</DialogTitle>
          <DialogDescription>
            Wypelnij dane przesylki, aby wygenerowac etykiete.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Service type */}
          <div className="space-y-2">
            <Label>Typ uslugi</Label>
            <div className="flex flex-col gap-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="service_type"
                  value="inpost_locker_standard"
                  checked={serviceType === "inpost_locker_standard"}
                  onChange={() => setServiceType("inpost_locker_standard")}
                  className="accent-primary h-4 w-4"
                />
                <span className="text-sm">Paczkomat InPost</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="service_type"
                  value="inpost_courier_standard"
                  checked={serviceType === "inpost_courier_standard"}
                  onChange={() => setServiceType("inpost_courier_standard")}
                  className="accent-primary h-4 w-4"
                />
                <span className="text-sm">Kurier InPost</span>
              </label>
            </div>
          </div>

          {/* Target point (only for locker) */}
          {isLocker && (
            <div className="space-y-2">
              <Label htmlFor="target_point">Paczkomat docelowy</Label>
              <Input
                id="target_point"
                placeholder="np. WAW123M"
                value={targetPoint}
                onChange={(e) => setTargetPoint(e.target.value)}
              />
            </div>
          )}

          {/* Parcel size */}
          <div className="space-y-2">
            <Label>Rozmiar paczki</Label>
            <Select
              value={parcelSize}
              onValueChange={(v) =>
                setParcelSize(v as "small" | "medium" | "large")
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="small">Maly (A: 8×38×64cm)</SelectItem>
                <SelectItem value="medium">Sredni (B: 19×38×64cm)</SelectItem>
                <SelectItem value="large">Duzy (C: 41×38×64cm)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Label format */}
          <div className="space-y-2">
            <Label>Format etykiety</Label>
            <Select
              value={labelFormat}
              onValueChange={(v) =>
                setLabelFormat(v as "pdf" | "zpl" | "epl")
              }
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
                  <span className="text-muted-foreground">Imie i nazwisko: </span>
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
                Dane odbiorcy pobrane z zamowienia
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
            Generuj etykiete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
