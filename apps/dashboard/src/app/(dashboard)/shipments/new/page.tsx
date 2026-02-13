"use client";

import { useState, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ShipmentForm } from "@/components/shipments/shipment-form";
import { RateShopping } from "@/components/shipping/rate-shopping";
import { useCreateShipment } from "@/hooks/use-shipments";
import type { ShippingRate } from "@/types/api";

type ProviderValue = "inpost" | "dhl" | "dpd" | "gls" | "ups" | "poczta_polska" | "orlen_paczka" | "fedex" | "manual";

const VALID_PROVIDERS: ProviderValue[] = [
  "inpost", "dhl", "dpd", "gls", "ups", "poczta_polska", "orlen_paczka", "fedex", "manual",
];

export default function NewShipmentPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const defaultOrderId = searchParams.get("order_id") ?? undefined;
  const carrierParam = searchParams.get("carrier") ?? undefined;
  const createShipment = useCreateShipment();

  const initialCarrier =
    carrierParam && VALID_PROVIDERS.includes(carrierParam as ProviderValue)
      ? (carrierParam as ProviderValue)
      : undefined;

  const [selectedCarrier, setSelectedCarrier] = useState<ProviderValue | undefined>(initialCarrier);

  const handleRateSelect = useCallback((rate: ShippingRate) => {
    if (VALID_PROVIDERS.includes(rate.carrier_code as ProviderValue)) {
      setSelectedCarrier(rate.carrier_code as ProviderValue);
    }
    toast.success(
      `Wybrano: ${rate.carrier_name} - ${rate.service_name} (${rate.price.toFixed(2)} ${rate.currency})`
    );
  }, []);

  const handleSubmit = (data: Parameters<typeof createShipment.mutate>[0]) => {
    createShipment.mutate(data, {
      onSuccess: (shipment) => {
        toast.success("Przesyłka została utworzona");
        router.push(`/shipments/${shipment.id}`);
      },
      onError: (error) => {
        toast.error(error.message || "Nie udało się utworzyć przesyłki");
      },
    });
  };

  const formDefaults: Record<string, unknown> = {};
  if (defaultOrderId) formDefaults.order_id = defaultOrderId;
  if (selectedCarrier) formDefaults.provider = selectedCarrier;

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/shipments">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Nowa przesyłka</h1>
          <p className="text-muted-foreground">
            Utwórz nową przesyłkę dla zamówienia
          </p>
        </div>
      </div>

      <RateShopping onSelectRate={handleRateSelect} />

      <Card>
        <CardHeader>
          <CardTitle>Dane przesyłki</CardTitle>
        </CardHeader>
        <CardContent>
          <ShipmentForm
            defaultValues={
              Object.keys(formDefaults).length > 0
                ? (formDefaults as Parameters<typeof ShipmentForm>[0]["defaultValues"])
                : undefined
            }
            key={selectedCarrier ?? "default"}
            onSubmit={handleSubmit}
            isLoading={createShipment.isPending}
          />
        </CardContent>
      </Card>
    </div>
  );
}
