"use client";

import { useState, useEffect } from "react";
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
import { CarrierMappingEditor } from "./carrier-mapping-editor";

const DEFAULT_ALLEGRO_MAPPING: Record<string, string> = {
  "Paczkomaty": "inpost",
  "Kurier InPost": "inpost",
  "Kurier DPD": "dpd",
  "DHL Kurier": "dhl",
  "GLS": "gls",
  "UPS": "ups",
  "Poczta Polska": "poczta_polska",
};

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
  const [autoCreate, setAutoCreate] = useState(
    (settings.auto_create_shipment as boolean) ?? false
  );
  const [autoLabel, setAutoLabel] = useState(
    (settings.auto_generate_label as boolean) ?? false
  );
  const [defaultCarrier, setDefaultCarrier] = useState(
    (settings.default_carrier as string) ?? ""
  );
  const [carrierMapping, setCarrierMapping] = useState<Record<string, string>>(
    (settings.carrier_mapping as Record<string, string>) ??
      (provider === "allegro" ? { ...DEFAULT_ALLEGRO_MAPPING } : {})
  );

  // Sync state when external settings change
  useEffect(() => {
    setAutoCreate((settings.auto_create_shipment as boolean) ?? false);
    setAutoLabel((settings.auto_generate_label as boolean) ?? false);
    setDefaultCarrier((settings.default_carrier as string) ?? "");
    setCarrierMapping(
      (settings.carrier_mapping as Record<string, string>) ??
        (provider === "allegro" ? { ...DEFAULT_ALLEGRO_MAPPING } : {})
    );
  }, [settings, provider]);

  const handleSave = () => {
    onSave({
      ...settings,
      auto_create_shipment: autoCreate,
      auto_generate_label: autoLabel,
      default_carrier: defaultCarrier || undefined,
      carrier_mapping: carrierMapping,
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Ustawienia przesyłek</CardTitle>
        <CardDescription>
          Konfiguruj automatyczne tworzenie przesyłek dla zamówień z tego marketplace.
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
            Automatycznie twórz przesyłkę dla nowych zamówień
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
            Automatycznie generuj etykietę (wkrótce)
          </Label>
        </div>

        {/* Default carrier */}
        <div className="space-y-2">
          <Label>Domyślny dostawca</Label>
          <Select
            value={defaultCarrier || "__none__"}
            onValueChange={(v) => setDefaultCarrier(v === "__none__" ? "" : v)}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Brak" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">Brak</SelectItem>
              {SHIPMENT_PROVIDERS.filter(p => p !== "manual").map((p) => (
                <SelectItem key={p} value={p}>
                  {SHIPMENT_PROVIDER_LABELS[p] ?? p}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">
            Używany gdy metoda dostawy z marketplace nie pasuje do żadnego mapowania.
          </p>
        </div>

        {/* Carrier mapping */}
        <div className="space-y-2">
          <Label>Mapowanie metod dostawy</Label>
          <p className="text-xs text-muted-foreground mb-2">
            Przypisz nazwy metod dostawy z marketplace do dostawców w OpenOMS.
            Dopasowanie działa na podstawie fragmentu nazwy (np. &quot;Paczkomaty&quot; pasuje do &quot;Paczkomaty 24/7&quot;).
          </p>
          <CarrierMappingEditor value={carrierMapping} onChange={setCarrierMapping} />
        </div>

        {/* Save button */}
        <div className="flex justify-end">
          <Button onClick={handleSave} disabled={isLoading}>
            {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
            Zapisz ustawienia przesyłek
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
