"use client";

import { useState, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/shared/form-layout";
import { ShipmentForm } from "@/components/shipments/shipment-form";
import { RateShopping } from "@/components/shipping/rate-shopping";
import { useCreateShipment } from "@/hooks/use-shipments";
import { isShipmentProviderSelectable } from "@/lib/readiness";
import type { ShippingRate } from "@/types/api";
import { useTranslations } from "next-intl";

type ProviderValue = "inpost" | "dhl" | "dpd" | "gls" | "ups" | "poczta_polska" | "orlen_paczka" | "fedex" | "manual";

const VALID_PROVIDERS: ProviderValue[] = [
  "inpost", "dhl", "dpd", "gls", "ups", "poczta_polska", "orlen_paczka", "fedex", "manual",
];

function isSelectableProvider(provider: string | undefined): provider is ProviderValue {
  return (
    !!provider &&
    VALID_PROVIDERS.includes(provider as ProviderValue) &&
    isShipmentProviderSelectable(provider)
  );
}

export default function NewShipmentPage() {
  const t = useTranslations("shipments");
  const router = useRouter();
  const searchParams = useSearchParams();
  const defaultOrderId = searchParams.get("order_id") ?? undefined;
  const carrierParam = searchParams.get("carrier") ?? undefined;
  const createShipment = useCreateShipment();

  const initialCarrier = isSelectableProvider(carrierParam) ? carrierParam : undefined;

  const [selectedCarrier, setSelectedCarrier] = useState<ProviderValue | undefined>(initialCarrier);

  const handleRateSelect = useCallback((rate: ShippingRate) => {
    if (isSelectableProvider(rate.carrier_code)) {
      setSelectedCarrier(rate.carrier_code as ProviderValue);
    }
    toast.success(
      t("rateSelected", { carrier: rate.carrier_name, service: rate.service_name, price: rate.price.toFixed(2), currency: rate.currency })
    );
  }, [t]);

  const handleSubmit = (data: Parameters<typeof createShipment.mutate>[0]) => {
    createShipment.mutate(data, {
      onSuccess: (shipment) => {
        toast.success(t("shipmentCreated"));
        router.push(`/shipments/${shipment.id}`);
      },
      onError: (error) => {
        toast.error(error.message || t("shipmentCreateError"));
      },
    });
  };

  const formDefaults: Record<string, unknown> = {};
  if (defaultOrderId) formDefaults.order_id = defaultOrderId;
  if (selectedCarrier) formDefaults.provider = selectedCarrier;

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-start gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/shipments">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-semibold tracking-normal">{t("newShipment")}</h1>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">
            {t("createNewShipmentForOrder")}
          </p>
        </div>
      </div>

      <RateShopping onSelectRate={handleRateSelect} />

      <FormSection title={t("shipmentData")} description={t("shipmentDataDescription")}>
        <ShipmentForm
          defaultValues={
            Object.keys(formDefaults).length > 0
              ? (formDefaults as Parameters<typeof ShipmentForm>[0]["defaultValues"])
              : undefined
          }
          key={selectedCarrier ?? "default"}
          onSubmit={handleSubmit}
          isPending={createShipment.isPending}
        />
      </FormSection>
    </div>
  );
}
