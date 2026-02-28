"use client";

import { MapPin } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PaczkomatSelector } from "@/components/shared/paczkomat-selector";

export interface CarrierFieldValues {
  service_type?: string;
  target_point?: string;
  parcel_size?: string;
  sending_method?: string;
  weight_kg?: number;
  width_cm?: number;
  height_cm?: number;
  depth_cm?: number;
  cod_amount?: number;
  insured_value?: number;
  [key: string]: unknown;
}

interface CarrierFieldsProps {
  provider: string;
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}

export function CarrierFields({ provider, values, onChange }: CarrierFieldsProps) {
  switch (provider) {
    case "inpost":
      return <InPostFields values={values} onChange={onChange} />;
    case "dhl":
      return <DHLFields values={values} onChange={onChange} />;
    case "dpd":
      return <DPDFields values={values} onChange={onChange} />;
    case "gls":
      return <GLSFields values={values} onChange={onChange} />;
    case "ups":
      return <UPSFields values={values} onChange={onChange} />;
    case "fedex":
      return <FedExFields values={values} onChange={onChange} />;
    case "poczta_polska":
      return <PocztaPolskaFields values={values} onChange={onChange} />;
    case "orlen_paczka":
      return <OrlenPaczkaFields values={values} onChange={onChange} />;
    default:
      return null;
  }
}

function ParcelDimensionFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label>Waga (kg)</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          placeholder="np. 1.5"
          value={values.weight_kg ?? ""}
          onChange={(e) =>
            onChange("weight_kg", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
      <div className="space-y-2">
        <Label>Szerokość (cm)</Label>
        <Input
          type="number"
          step="1"
          min="0"
          placeholder="np. 30"
          value={values.width_cm ?? ""}
          onChange={(e) =>
            onChange("width_cm", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
      <div className="space-y-2">
        <Label>Wysokość (cm)</Label>
        <Input
          type="number"
          step="1"
          min="0"
          placeholder="np. 20"
          value={values.height_cm ?? ""}
          onChange={(e) =>
            onChange("height_cm", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
      <div className="space-y-2">
        <Label>Głębokość (cm)</Label>
        <Input
          type="number"
          step="1"
          min="0"
          placeholder="np. 15"
          value={values.depth_cm ?? ""}
          onChange={(e) =>
            onChange("depth_cm", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
    </div>
  );
}

function InPostFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  const serviceType = values.service_type ?? "inpost_locker_standard";
  const isLocker = serviceType === "inpost_locker_standard";
  const targetPoint = (values.target_point as string) ?? "";

  return (
    <>
      {/* Service type */}
      <div className="space-y-2">
        <Label>Typ usługi</Label>
        <div className="flex flex-col gap-2">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="service_type"
              value="inpost_locker_standard"
              checked={serviceType === "inpost_locker_standard"}
              onChange={() => onChange("service_type", "inpost_locker_standard")}
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
              onChange={() => onChange("service_type", "inpost_courier_standard")}
              className="accent-primary h-4 w-4"
            />
            <span className="text-sm">Kurier InPost</span>
          </label>
        </div>
      </div>

      {/* Target point (only for locker) */}
      {isLocker && (
        <div className="space-y-2">
          <Label>Paczkomat docelowy</Label>
          {targetPoint && (
            <div className="flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-2">
              <MapPin className="h-4 w-4 text-primary" />
              <span className="font-medium text-sm">{targetPoint}</span>
            </div>
          )}
          <PaczkomatSelector
            mode="inline"
            value={targetPoint}
            onPointSelect={(name) => onChange("target_point", name)}
          />
        </div>
      )}

      {/* Sending method */}
      <div className="space-y-2">
        <Label>Metoda nadania</Label>
        <Select
          value={(values.sending_method as string) ?? "__default__"}
          onValueChange={(v) => onChange("sending_method", v === "__default__" ? undefined : v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Domyślna (z ustawień integracji)" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__default__">Domyślna (z ustawień integracji)</SelectItem>
            <SelectItem value="dispatch_order">Kurier odbierze (zlecenie odbioru)</SelectItem>
            <SelectItem value="parcel_locker">Nadam w paczkomacie</SelectItem>
            <SelectItem value="pop">Nadam w PaczkoPunkcie (POP)</SelectItem>
            <SelectItem value="any_point">Dowolny punkt (paczkomat/POP)</SelectItem>
            <SelectItem value="pok">Punkt Obsługi Klienta (POK)</SelectItem>
            <SelectItem value="branch">Oddział InPost</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">
          Jak paczka trafi do sieci InPost. Pozostaw puste, aby użyć domyślnej metody z ustawień integracji.
        </p>
      </div>

      {/* Parcel size */}
      <div className="space-y-2">
        <Label>Rozmiar paczki</Label>
        <Select
          value={(values.parcel_size as string) ?? "small"}
          onValueChange={(v) => onChange("parcel_size", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="small">Mały (A: 8×38×64cm)</SelectItem>
            <SelectItem value="medium">Średni (B: 19×38×64cm)</SelectItem>
            <SelectItem value="large">Duży (C: 41×38×64cm)</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </>
  );
}

function DHLFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label>Typ usługi</Label>
        <Select
          value={(values.service_type as string) ?? "dhl_parcel"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="dhl_parcel">DHL Parcel</SelectItem>
            <SelectItem value="dhl_courier">DHL Courier</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}

function DPDFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label>Typ usługi</Label>
        <Select
          value={(values.service_type as string) ?? "dpd_classic"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="dpd_classic">DPD Classic</SelectItem>
            <SelectItem value="dpd_pickup">DPD Pickup</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
    </>
  );
}

function CODAndInsuranceFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label>Kwota pobrania (PLN)</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          placeholder="Opcjonalnie"
          value={values.cod_amount ?? ""}
          onChange={(e) =>
            onChange("cod_amount", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
      <div className="space-y-2">
        <Label>Wartość ubezpieczenia (PLN)</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          placeholder="Opcjonalnie"
          value={values.insured_value ?? ""}
          onChange={(e) =>
            onChange("insured_value", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
    </div>
  );
}

function GLSFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}

function UPSFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label>Typ usługi</Label>
        <Select
          value={(values.service_type as string) ?? "11"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="11">UPS Standard</SelectItem>
            <SelectItem value="65">UPS Express Saver</SelectItem>
            <SelectItem value="07">UPS Express</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}

function FedExFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label>Typ usługi</Label>
        <Select
          value={(values.service_type as string) ?? "FEDEX_INTERNATIONAL_PRIORITY"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="FEDEX_INTERNATIONAL_PRIORITY">FedEx International Priority</SelectItem>
            <SelectItem value="INTERNATIONAL_ECONOMY">FedEx International Economy</SelectItem>
            <SelectItem value="FEDEX_GROUND">FedEx Ground</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}

function PocztaPolskaFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label>Typ usługi</Label>
        <Select
          value={(values.service_type as string) ?? "POCZTEX_KURIER_48"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="POCZTEX_KURIER_48">Pocztex Kurier 48h</SelectItem>
            <SelectItem value="POCZTEX_KURIER_24">Pocztex Kurier 24h</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}

function OrlenPaczkaFields({
  values,
  onChange,
}: {
  values: CarrierFieldValues;
  onChange: (field: string, value: unknown) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label>Automat Orlen Paczka</Label>
        <Input
          type="text"
          placeholder="np. PL10001"
          value={(values.target_point as string) ?? ""}
          onChange={(e) => onChange("target_point", e.target.value || undefined)}
        />
        <p className="text-xs text-muted-foreground">
          Identyfikator automatu docelowego Orlen Paczka.
        </p>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}
