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
import { useTranslations } from "next-intl";

const EU_COUNTRY_CODES = [
  "AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR",
  "DE", "GR", "HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL",
  "PL", "PT", "RO", "SK", "SI", "ES", "SE",
] as const;

const RATE_TYPE_KEYS: { value: string; tKey: string }[] = [
  { value: "standard", tKey: "rateStandard" },
  { value: "reduced_1", tKey: "rateReduced1" },
  { value: "reduced_2", tKey: "rateReduced2" },
  { value: "super_reduced", tKey: "rateSuperReduced" },
];

const DEFAULT_CONFIG: OSSConfig = {
  enabled: false,
  home_country: "PL",
  default_vat_rate: "standard",
};

export default function VATOSSSettingsPage() {
  const t = useTranslations("settings.vatOss");
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
      toast.success(t("settingsSaved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("saveFailed");
      toast.error(message);
    }
  }, [form, updateConfig, t]);

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
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("subtitle")}
          </p>
        </div>

        {/* Threshold warning */}
        {threshold && threshold.exceeded && (
          <Card className="border-amber-300 bg-amber-50/50 dark:border-amber-800 dark:bg-amber-950/20">
            <CardContent className="py-4 flex items-start gap-3">
              <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" />
              <div>
                <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                  {t("thresholdExceeded")}
                </p>
                <p className="text-sm text-amber-700 dark:text-amber-400 mt-1">
                  {t("thresholdExceededDesc", {
                    year: currentYear,
                    amount: threshold.total_cross_border_eur.toLocaleString("pl-PL", {
                      minimumFractionDigits: 2,
                    }),
                  })}
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
                  {t("thresholdApproaching")}
                </p>
                <p className="text-sm text-yellow-700 dark:text-yellow-400 mt-1">
                  {t("thresholdApproachingDesc", {
                    remaining: threshold.remaining_eur.toLocaleString("pl-PL", {
                      minimumFractionDigits: 2,
                    }),
                  })}
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Threshold status */}
        <Card>
          <CardHeader>
            <CardTitle>{t("thresholdStatus", { year: currentYear })}</CardTitle>
            <CardDescription>
              {t("thresholdStatusDesc")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {thresholdLoading ? (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("loading")}
              </div>
            ) : threshold ? (
              <>
                <div className="flex items-center justify-between text-sm">
                  <span>{t("crossBorderSales")}</span>
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
                    ? t("thresholdExceededNote")
                    : t("remainingToThreshold", { remaining: threshold.remaining_eur.toLocaleString("pl-PL", { minimumFractionDigits: 2 }) })}
                </p>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">
                {t("noThresholdData")}
              </p>
            )}
          </CardContent>
        </Card>

        {/* OSS Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>{t("ossConfig")}</CardTitle>
            <CardDescription>
              {t("ossConfigDesc")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="oss-enabled">{t("enableVatOss")}</Label>
                <p className="text-xs text-muted-foreground">
                  {t("enableVatOssDesc")}
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
                <Label>{t("homeCountry")}</Label>
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
                    {EU_COUNTRY_CODES
                      .map((code) => ({ code, name: t(`countries.${code}`) }))
                      .sort((a, b) => a.name.localeCompare(b.name))
                      .map(({ code, name }) => (
                        <SelectItem key={code} value={code}>
                          {code} — {name}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  {t("homeCountryDesc")}
                </p>
              </div>

              <div className="space-y-2">
                <Label>{t("defaultVatRate")}</Label>
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
                    {RATE_TYPE_KEYS.map((rt) => (
                      <SelectItem key={rt.value} value={rt.value}>
                        {t(rt.tKey as never)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  {t("defaultVatRateDesc")}
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
              {t("saveSettings")}
            </Button>
          </CardContent>
        </Card>

        {/* Info card */}
        <Card>
          <CardContent className="py-4 flex items-start gap-3">
            <Info className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
            <div className="space-y-2">
              <p className="text-sm font-medium">
                {t("ossInfoTitle")}
              </p>
              <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
                <li>{t("ossInfo1")}</li>
                <li>{t("ossInfo2")}</li>
                <li>{t("ossInfo3")}</li>
                <li>{t("ossInfo4")}</li>
                <li>{t("ossInfo5")}</li>
                <li>{t("ossInfo6")}</li>
                <li>{t("ossInfo7")}</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
