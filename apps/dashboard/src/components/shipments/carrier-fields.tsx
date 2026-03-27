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
import { useTranslations } from "next-intl";

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
  const t = useTranslations("shipments");
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label>{t("weightKg")}</Label>
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
        <Label>{t("addPackage.width")}</Label>
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
        <Label>{t("addPackage.height")}</Label>
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
        <Label>{t("form.depth")}</Label>
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
  const t = useTranslations("shipments");
  const serviceType = values.service_type ?? "inpost_locker_standard";
  const isLocker = serviceType === "inpost_locker_standard";
  const targetPoint = (values.target_point as string) ?? "";

  return (
    <>
      {/* Service type */}
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
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
            <span className="text-sm">{t("inpostLocker")}</span>
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
            <span className="text-sm">{t("inpostCourier")}</span>
          </label>
        </div>
      </div>

      {/* Target point (only for locker) */}
      {isLocker && (
        <div className="space-y-2">
          <Label>{t("targetLocker")}</Label>
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
        <Label>{t("sendingMethod")}</Label>
        <Select
          value={(values.sending_method as string) ?? "__default__"}
          onValueChange={(v) => onChange("sending_method", v === "__default__" ? undefined : v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder={t("domyslnaZUstawienIntegracji")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__default__">{t("domyslnaZUstawienIntegracji")}</SelectItem>
            <SelectItem value="dispatch_order">{t("inpost.sendingMethods.dispatchOrder")}</SelectItem>
            <SelectItem value="parcel_locker">{t("inpost.sendingMethods.parcelLocker")}</SelectItem>
            <SelectItem value="pop">{t("inpost.sendingMethods.pop")}</SelectItem>
            <SelectItem value="any_point">{t("inpost.sendingMethods.anyPoint")}</SelectItem>
            <SelectItem value="pok">{t("inpost.sendingMethods.pok")}</SelectItem>
            <SelectItem value="branch">{t("inpost.sendingMethods.branch")}</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">
          {t("jakPaczkaTrafiDoSieciInpostPozostaw")}
        </p>
      </div>

      {/* Parcel size */}
      <div className="space-y-2">
        <Label>{t("parcelSize")}</Label>
        <Select
          value={(values.parcel_size as string) ?? "small"}
          onValueChange={(v) => onChange("parcel_size", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="small">{t("smallA83864cm")}</SelectItem>
            <SelectItem value="medium">{t("sredniB193864cm")}</SelectItem>
            <SelectItem value="large">{t("duzyC413864cm")}</SelectItem>
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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
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
  const t = useTranslations("shipments");
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label>{t("codAmountPln")}</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          placeholder={t("optional")}
          value={values.cod_amount ?? ""}
          onChange={(e) =>
            onChange("cod_amount", e.target.value ? parseFloat(e.target.value) : undefined)
          }
        />
      </div>
      <div className="space-y-2">
        <Label>{t("wartoscUbezpieczeniaPln")}</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          placeholder={t("optional")}
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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
        <Select
          value={(values.service_type as string) ?? "standard"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="standard">GLS Standard</SelectItem>
            <SelectItem value="express_10">GLS Express 10:00</SelectItem>
            <SelectItem value="express_12">GLS Express 12:00</SelectItem>
          </SelectContent>
        </Select>
      </div>

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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
        <Select
          value={(values.service_type as string) ?? "11"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="11">UPS Standard</SelectItem>
            <SelectItem value="65">UPS Worldwide Saver</SelectItem>
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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
        <Select
          value={(values.service_type as string) ?? "FEDEX_INTERNATIONAL_PRIORITY"}
          onValueChange={(v) => onChange("service_type", v)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="FEDEX_INTERNATIONAL_PRIORITY">FedEx International Priority</SelectItem>
            <SelectItem value="FEDEX_INTERNATIONAL_ECONOMY">FedEx International Economy</SelectItem>
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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("serviceType")}</Label>
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
            <SelectItem value="POCZTEX_2_0">Pocztex 2.0</SelectItem>
            <SelectItem value="PACZKA_POCZTOWA">Paczka Pocztowa</SelectItem>
            <SelectItem value="EMS">EMS (Express Mail Service)</SelectItem>
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
  const t = useTranslations("shipments");
  return (
    <>
      <div className="space-y-2">
        <Label>{t("orlenPaczkaLocker")}</Label>
        <Input
          type="text"
          placeholder="np. PL10001"
          value={(values.target_point as string) ?? ""}
          onChange={(e) => onChange("target_point", e.target.value || undefined)}
        />
        <p className="text-xs text-muted-foreground">
          {t("orlenPaczkaIdHelp")}
        </p>
      </div>

      <ParcelDimensionFields values={values} onChange={onChange} />
      <CODAndInsuranceFields values={values} onChange={onChange} />
    </>
  );
}
