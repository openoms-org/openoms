"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import { Save, AlertTriangle, Info, Loader2 } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  useOSSConfig,
  useUpdateOSSConfig,
  useOSSThreshold,
} from "@/hooks/use-vat-oss";
import { Progress } from "@/components/ui/progress";
import type { OSSConfig } from "@/types/api";

const EU_COUNTRIES: Record<string, string> = {
  AT: "Austria",
  BE: "Belgia",
  BG: "Bulgaria",
  HR: "Chorwacja",
  CY: "Cypr",
  CZ: "Czechy",
  DK: "Dania",
  EE: "Estonia",
  FI: "Finlandia",
  FR: "Francja",
  DE: "Niemcy",
  GR: "Grecja",
  HU: "Wegry",
  IE: "Irlandia",
  IT: "Wlochy",
  LV: "Lotwa",
  LT: "Litwa",
  LU: "Luksemburg",
  MT: "Malta",
  NL: "Holandia",
  PL: "Polska",
  PT: "Portugalia",
  RO: "Rumunia",
  SK: "Slowacja",
  SI: "Slowenia",
  ES: "Hiszpania",
  SE: "Szwecja",
};

const RATE_TYPES = [
  { value: "standard", label: "Standardowa" },
  { value: "reduced_1", label: "Obnizona 1" },
  { value: "reduced_2", label: "Obnizona 2" },
  { value: "super_reduced", label: "Super obnizona" },
];

const DEFAULT_CONFIG: OSSConfig = {
  enabled: false,
  home_country: "PL",
  default_vat_rate: "standard",
};

export default function VATOSSSettingsPage() {
  const { data: config, isLoading: configLoading } = useOSSConfig();
  const updateConfig = useUpdateOSSConfig();
  const currentYear = new Date().getFullYear();
  const { data: threshold, isLoading: thresholdLoading } =
    useOSSThreshold(currentYear);

  const [form, setForm] = useState<OSSConfig>(DEFAULT_CONFIG);

  useEffect(() => {
    if (config) {
      setForm({ ...DEFAULT_CONFIG, ...config });
    }
  }, [config]);

  const handleSave = useCallback(async () => {
    try {
      await updateConfig.mutateAsync(form);
      toast.success("Ustawienia VAT OSS zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udalo sie zapisac ustawien";
      toast.error(message);
    }
  }, [form, updateConfig]);

  const thresholdPercent = threshold
    ? Math.min(
        (threshold.total_cross_border_eur / threshold.threshold_eur) * 100,
        100
      )
    : 0;

  if (configLoading) {
    return (
      <AdminGuard>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      </AdminGuard>
    );
  }

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">VAT OSS</h1>
          <p className="text-muted-foreground mt-1">
            Konfiguracja procedury One Stop Shop dla transgranicznej sprzedazy
            w UE
          </p>
        </div>

        {/* Threshold warning */}
        {threshold && threshold.exceeded && (
          <Card className="border-amber-300 bg-amber-50/50 dark:border-amber-800 dark:bg-amber-950/20">
            <CardContent className="py-4 flex items-start gap-3">
              <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" />
              <div>
                <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                  Prog 10 000 EUR przekroczony
                </p>
                <p className="text-sm text-amber-700 dark:text-amber-400 mt-1">
                  Twoja sprzedaz transgraniczna w {currentYear} roku wynosi{" "}
                  {threshold.total_cross_border_eur.toLocaleString("pl-PL", {
                    minimumFractionDigits: 2,
                  })}{" "}
                  EUR. Musisz stosowac stawki VAT kraju przeznaczenia i skladac
                  deklaracje OSS kwartalnie.
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {threshold && !threshold.exceeded && thresholdPercent >= 80 && (
          <Card className="border-yellow-300 bg-yellow-50/50 dark:border-yellow-800 dark:bg-yellow-950/20">
            <CardContent className="py-4 flex items-start gap-3">
              <Info className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mt-0.5 shrink-0" />
              <div>
                <p className="text-sm font-medium text-yellow-800 dark:text-yellow-300">
                  Zblizasz sie do progu 10 000 EUR
                </p>
                <p className="text-sm text-yellow-700 dark:text-yellow-400 mt-1">
                  Pozostalo{" "}
                  {threshold.remaining_eur.toLocaleString("pl-PL", {
                    minimumFractionDigits: 2,
                  })}{" "}
                  EUR do progu. Po jego przekroczeniu konieczne bedzie
                  stosowanie stawek VAT krajow przeznaczenia.
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Threshold status */}
        <Card>
          <CardHeader>
            <CardTitle>Status progu na rok {currentYear}</CardTitle>
            <CardDescription>
              Roczny prog sprzedazy transgranicznej w UE
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {thresholdLoading ? (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Ladowanie...
              </div>
            ) : threshold ? (
              <>
                <div className="flex items-center justify-between text-sm">
                  <span>Sprzedaz transgraniczna</span>
                  <span className="font-medium">
                    {threshold.total_cross_border_eur.toLocaleString("pl-PL", {
                      minimumFractionDigits: 2,
                    })}{" "}
                    /{" "}
                    {threshold.threshold_eur.toLocaleString("pl-PL", {
                      minimumFractionDigits: 0,
                    })}{" "}
                    EUR
                  </span>
                </div>
                <Progress value={thresholdPercent} className="h-3" />
                <p className="text-xs text-muted-foreground">
                  {threshold.exceeded
                    ? "Prog przekroczony — obowiazek stosowania VAT kraju przeznaczenia"
                    : `Pozostalo ${threshold.remaining_eur.toLocaleString("pl-PL", { minimumFractionDigits: 2 })} EUR do progu`}
                </p>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">
                Brak danych o progu
              </p>
            )}
          </CardContent>
        </Card>

        {/* OSS Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>Konfiguracja OSS</CardTitle>
            <CardDescription>
              Wlacz lub wylacz procedure One Stop Shop i ustaw parametry
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="oss-enabled">Wlacz VAT OSS</Label>
                <p className="text-xs text-muted-foreground">
                  Aktywuje automatyczne naliczanie VAT wg kraju przeznaczenia
                </p>
              </div>
              <Switch
                id="oss-enabled"
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  setForm((prev) => ({ ...prev, enabled: checked }))
                }
              />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>Kraj macierzysty</Label>
                <Select
                  value={form.home_country}
                  onValueChange={(value) =>
                    setForm((prev) => ({ ...prev, home_country: value }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(EU_COUNTRIES)
                      .sort(([, a], [, b]) => a.localeCompare(b))
                      .map(([code, name]) => (
                        <SelectItem key={code} value={code}>
                          {code} — {name}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Kraj rejestracji Twojej firmy
                </p>
              </div>

              <div className="space-y-2">
                <Label>Domyslna stawka VAT</Label>
                <Select
                  value={form.default_vat_rate}
                  onValueChange={(value) =>
                    setForm((prev) => ({ ...prev, default_vat_rate: value }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {RATE_TYPES.map((rt) => (
                      <SelectItem key={rt.value} value={rt.value}>
                        {rt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Stawka stosowana do obliczen VAT
                </p>
              </div>
            </div>

            <Button
              onClick={handleSave}
              disabled={updateConfig.isPending}
              className="w-full sm:w-auto"
            >
              {updateConfig.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Save className="h-4 w-4" />
              )}
              Zapisz ustawienia
            </Button>
          </CardContent>
        </Card>

        {/* Info card */}
        <Card>
          <CardContent className="py-4 flex items-start gap-3">
            <Info className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
            <div className="space-y-2">
              <p className="text-sm font-medium">
                Informacje o procedurze OSS
              </p>
              <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
                <li>
                  Prog roczny: 10 000 EUR sprzedazy transgranicznej w UE
                </li>
                <li>
                  Ponizej progu mozesz stosowac stawke VAT kraju macierzystego
                </li>
                <li>
                  Powyzej progu musisz stosowac stawke VAT kraju przeznaczenia
                </li>
                <li>
                  Deklaracje OSS skladane sa kwartalnie (do konca miesiaca po
                  zakonczeniu kwartalu)
                </li>
                <li>
                  Q1: styczen-marzec (termin: 30 kwietnia), Q2: kwiecien-czerwiec
                  (termin: 31 lipca)
                </li>
                <li>
                  Q3: lipiec-wrzesien (termin: 31 pazdziernika), Q4:
                  pazdziernik-grudzien (termin: 31 stycznia)
                </li>
                <li>
                  Raporty generowane w OpenOMS sa informacyjne — deklaracje
                  skladasz przez portal urzedu skarbowego
                </li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
