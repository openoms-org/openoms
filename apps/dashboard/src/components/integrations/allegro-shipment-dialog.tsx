"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Loader2, Download, Package, CheckCircle2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import {
  useAllegroDeliveryServices,
  useCreateAllegroShipment,
  downloadAllegroLabel,
} from "@/hooks/use-allegro";
import type { Order } from "@/types/api";
import { getErrorMessage } from "@/lib/api-client";

interface AllegroShipmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: Order;
}

type Step = "select-service" | "package-details" | "result";

export function AllegroShipmentDialog({
  open,
  onOpenChange,
  order,
}: AllegroShipmentDialogProps) {
  const [step, setStep] = useState<Step>("select-service");
  const [selectedServiceId, setSelectedServiceId] = useState("");
  const [weight, setWeight] = useState("1");
  const [length, setLength] = useState("30");
  const [width, setWidth] = useState("20");
  const [height, setHeight] = useState("15");
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(
    null
  );
  const [isDownloading, setIsDownloading] = useState(false);

  const { data: deliveryData, isLoading: isLoadingServices } =
    useAllegroDeliveryServices();
  const createShipment = useCreateAllegroShipment();

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setStep("select-service");
      setSelectedServiceId("");
      setWeight("1");
      setLength("30");
      setWidth("20");
      setHeight("15");
      setCreatedShipmentId(null);
    }
  }, [open]);

  const deliveryServices = deliveryData?.delivery_services ?? [];

  const handleCreateShipment = async () => {
    if (!selectedServiceId) {
      toast.error("Wybierz usługę dostawy");
      return;
    }

    const shippingAddr = order.shipping_address;

    try {
      const result = await createShipment.mutateAsync({
        commandId: crypto.randomUUID(),
        input: {
          deliveryMethodId: selectedServiceId,
          sender: {
            name: "",
            street: "",
            city: "",
            zipCode: "",
            countryCode: "PL",
          },
          receiver: {
            name: shippingAddr?.name || order.customer_name || "",
            street: shippingAddr?.street || "",
            city: shippingAddr?.city || "",
            zipCode: shippingAddr?.postal_code || "",
            countryCode: shippingAddr?.country || "PL",
            phone: order.customer_phone || "",
            email: order.customer_email || "",
          },
          packages: [
            {
              weight: {
                value: parseFloat(weight) || 1,
                unit: "KILOGRAMS",
              },
              length: {
                value: parseFloat(length) || 30,
                unit: "CENTIMETERS",
              },
              width: {
                value: parseFloat(width) || 20,
                unit: "CENTIMETERS",
              },
              height: {
                value: parseFloat(height) || 15,
                unit: "CENTIMETERS",
              },
            },
          ],
        },
      });

      setCreatedShipmentId(result.shipmentId);
      setStep("result");
      toast.success("Przesyłka została utworzona w Allegro");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDownloadLabel = async () => {
    if (!createdShipmentId) return;
    setIsDownloading(true);
    try {
      await downloadAllegroLabel(createdShipmentId);
      toast.success("Etykieta została pobrana");
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setIsDownloading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Package className="h-5 w-5" />
            Wyślij z Allegro
          </DialogTitle>
          <DialogDescription>
            Utwórz przesyłkę i wygeneruj etykietę kurierską przez Allegro.
          </DialogDescription>
        </DialogHeader>

        {step === "select-service" && (
          <div className="space-y-4">
            <div>
              <Label>Usługa dostawy</Label>
              {isLoadingServices ? (
                <div className="flex items-center gap-2 mt-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Ładowanie usług dostawy...
                </div>
              ) : deliveryServices.length === 0 ? (
                <p className="mt-2 text-sm text-muted-foreground">
                  Brak dostępnych usług dostawy. Sprawdź konfigurację Allegro.
                </p>
              ) : (
                <Select
                  value={selectedServiceId}
                  onValueChange={setSelectedServiceId}
                >
                  <SelectTrigger className="mt-1">
                    <SelectValue placeholder="Wybierz usługę dostawy..." />
                  </SelectTrigger>
                  <SelectContent>
                    {deliveryServices.map((svc) => (
                      <SelectItem key={svc.id} value={svc.id}>
                        {svc.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Anuluj
              </Button>
              <Button
                onClick={() => setStep("package-details")}
                disabled={!selectedServiceId}
              >
                Dalej
              </Button>
            </DialogFooter>
          </div>
        )}

        {step === "package-details" && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Waga (kg)</Label>
                <Input
                  type="number"
                  step="0.1"
                  min="0.1"
                  value={weight}
                  onChange={(e) => setWeight(e.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label>Długość (cm)</Label>
                <Input
                  type="number"
                  step="1"
                  min="1"
                  value={length}
                  onChange={(e) => setLength(e.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label>Szerokość (cm)</Label>
                <Input
                  type="number"
                  step="1"
                  min="1"
                  value={width}
                  onChange={(e) => setWidth(e.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label>Wysokość (cm)</Label>
                <Input
                  type="number"
                  step="1"
                  min="1"
                  value={height}
                  onChange={(e) => setHeight(e.target.value)}
                  className="mt-1"
                />
              </div>
            </div>

            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              <p className="font-medium">Dane odbiorcy</p>
              <p className="text-muted-foreground">
                {order.shipping_address?.name || order.customer_name}
              </p>
              <p className="text-muted-foreground">
                {order.shipping_address?.street}
              </p>
              <p className="text-muted-foreground">
                {order.shipping_address?.postal_code}{" "}
                {order.shipping_address?.city}
              </p>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setStep("select-service")}
              >
                Wstecz
              </Button>
              <Button
                onClick={handleCreateShipment}
                disabled={createShipment.isPending}
              >
                {createShipment.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Tworzenie...
                  </>
                ) : (
                  "Utwórz przesyłkę"
                )}
              </Button>
            </DialogFooter>
          </div>
        )}

        {step === "result" && createdShipmentId && (
          <div className="space-y-4">
            <div className="flex flex-col items-center gap-3 py-4">
              <CheckCircle2 className="h-12 w-12 text-green-500" />
              <p className="text-lg font-medium">Przesyłka utworzona</p>
              <p className="text-sm text-muted-foreground text-center">
                ID przesyłki: {createdShipmentId}
              </p>
            </div>

            <Button
              className="w-full"
              onClick={handleDownloadLabel}
              disabled={isDownloading}
            >
              {isDownloading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Download className="mr-2 h-4 w-4" />
              )}
              Pobierz etykietę
            </Button>

            <DialogFooter>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Zamknij
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
