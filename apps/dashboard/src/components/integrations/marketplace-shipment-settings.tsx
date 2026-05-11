"use client";

import { useState, useEffect, useMemo } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SHIPMENT_PROVIDERS, SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import { getSelectableCarrierShipmentProviders } from "@/lib/readiness";
import { CarrierMappingEditor } from "./carrier-mapping-editor";
import { useTranslations } from "next-intl";

const DEFAULT_ALLEGRO_MAPPING: Record<string, string> = {
  "Paczkomaty": "inpost",
  "Kurier InPost": "inpost",
  "Kurier DPD": "dpd",
  "DHL Kurier": "dhl",
  "GLS": "gls",
  "UPS": "ups",
  "Poczta Polska": "poczta_polska",
};

function getCarrierMapping(
  provider: string,
  settings: Record<string, unknown>,
): Record<string, string> {
  return (
    (settings.carrier_mapping as Record<string, string> | undefined) ??
    (provider === "allegro" ? { ...DEFAULT_ALLEGRO_MAPPING } : {})
  );
}

function filterSelectableCarrierMapping(
  mapping: Record<string, string>,
  selectableProviders: string[],
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(mapping).filter(([, mappedProvider]) =>
      selectableProviders.includes(mappedProvider),
    ),
  );
}

function normalizeSelectableProvider(
  provider: unknown,
  selectableProviders: string[],
): string {
  return typeof provider === "string" && selectableProviders.includes(provider)
    ? provider
    : "";
}

interface MarketplaceShipmentSettingsProps {
  provider: string;
  settings: Record<string, unknown>;
  onSave: (settings: Record<string, unknown>) => void;
  isLoading: boolean;
}

export function MarketplaceShipmentSettings({
  provider,
  settings,
  onSave,
  isLoading,
}: MarketplaceShipmentSettingsProps) {
  const t = useTranslations("integrations");
  const selectableShipmentProviders = useMemo(
    () => getSelectableCarrierShipmentProviders(SHIPMENT_PROVIDERS),
    [],
  );
  const [autoCreate, setAutoCreate] = useState(
    (settings.auto_create_shipment as boolean) ?? false
  );
  const [autoLabel, setAutoLabel] = useState(
    (settings.auto_generate_label as boolean) ?? false
  );
  const [defaultCarrier, setDefaultCarrier] = useState(
    normalizeSelectableProvider(settings.default_carrier, selectableShipmentProviders)
  );
  const [carrierMapping, setCarrierMapping] = useState<Record<string, string>>(
    filterSelectableCarrierMapping(
      getCarrierMapping(provider, settings),
      selectableShipmentProviders,
    )
  );

  // Sync state when external settings change
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setAutoCreate((settings.auto_create_shipment as boolean) ?? false);
    setAutoLabel((settings.auto_generate_label as boolean) ?? false);
    setDefaultCarrier(
      normalizeSelectableProvider(settings.default_carrier, selectableShipmentProviders),
    );
    setCarrierMapping(
      filterSelectableCarrierMapping(
        getCarrierMapping(provider, settings),
        selectableShipmentProviders,
      )
    );
  }, [settings, provider, selectableShipmentProviders]);

  const handleSave = () => {
    onSave({
      ...settings,
      auto_create_shipment: autoCreate,
      auto_generate_label: autoLabel,
      default_carrier: defaultCarrier || undefined,
      carrier_mapping: filterSelectableCarrierMapping(
        carrierMapping,
        selectableShipmentProviders,
      ),
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("shipmentSettings")}</CardTitle>
        <CardDescription>
          {t("shipmentAutoCreateDesc")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Auto-create shipment */}
        <div className="flex items-center space-x-2">
          <Checkbox
            id="auto_create_shipment"
            checked={autoCreate}
            onCheckedChange={(checked) => {
              setAutoCreate(checked === true);
              if (!checked) setAutoLabel(false);
            }}
          />
          <Label htmlFor="auto_create_shipment" className="font-normal cursor-pointer">
            {t("shipmentAutoCreate")}
          </Label>
        </div>

        {/* Auto-generate label */}
        <div className="flex items-center space-x-2">
          <Checkbox
            id="auto_generate_label"
            checked={autoLabel}
            disabled={!autoCreate}
            onCheckedChange={(checked) => setAutoLabel(checked === true)}
          />
          <Label
            htmlFor="auto_generate_label"
            className={`font-normal cursor-pointer ${!autoCreate ? "text-muted-foreground" : ""}`}
          >
            {t("autoGenerateLabelSoon")}
          </Label>
        </div>

        {/* Default carrier */}
        <div className="space-y-2">
          <Label>{t("defaultCarrier")}</Label>
          <Select
            value={defaultCarrier || "__none__"}
            onValueChange={(v) => setDefaultCarrier(v === "__none__" ? "" : v)}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder={t("noneOption")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">{t("noneOption")}</SelectItem>
              {selectableShipmentProviders.map((p) => (
                <SelectItem key={p} value={p}>
                  {SHIPMENT_PROVIDER_LABELS[p] ?? p}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">
            {t("defaultCarrierHint")}
          </p>
        </div>

        {/* Carrier mapping */}
        <div className="space-y-2">
          <Label>{t("deliveryMethodMapping")}</Label>
          <p className="text-xs text-muted-foreground mb-2">
            {t("carrierMappingDesc")}
          </p>
          <CarrierMappingEditor value={carrierMapping} onChange={setCarrierMapping} />
        </div>

        {/* Save button */}
        <div className="flex justify-end">
          <Button onClick={handleSave} disabled={isLoading}>
            {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("saveShipmentSettings")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
