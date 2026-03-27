"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
  useKSeFSettings,
  useUpdateKSeFSettings,
  useTestKSeFConnection,
} from "@/hooks/use-ksef";
import { Loader2, Save, TestTube, CheckCircle, XCircle } from "lucide-react";
import { DevelopmentBanner } from "@/components/shared/development-banner";
import { KSEF_ENVIRONMENTS } from "@/lib/constants";
import type { KSeFSettings } from "@/types/api";
import { useTranslations } from "next-intl";

const DEFAULT_SETTINGS: KSeFSettings = {
  enabled: false,
  environment: "test",
  nip: "",
  token: "",
  auto_send: false,
  company_name: "",
  company_street: "",
  company_city: "",
  company_postal: "",
  company_country: "PL",
};

export default function KSeFSettingsPage() {
  const t = useTranslations("ksef");
  const ts = useTranslations("settings");
  const { data: settings, isLoading } = useKSeFSettings();
  const updateSettings = useUpdateKSeFSettings();
  const testConnection = useTestKSeFConnection();

  const [form, setForm] = useState<KSeFSettings>(DEFAULT_SETTINGS);
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  useEffect(() => {
    if (settings) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setForm({
        ...DEFAULT_SETTINGS,
        ...settings,
      });
    }
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success(t("settingsSaved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("ksefSettingsSaveError");
      toast.error(message);
    }
  };

  const handleTestConnection = async () => {
    setTestResult(null);
    try {
      const result = await testConnection.mutateAsync();
      setTestResult(result);
      if (result.success) {
        toast.success(result.message);
      } else {
        toast.error(result.message);
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("ksefConnectionTestError");
      toast.error(message);
      setTestResult({ success: false, message });
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-muted-foreground">
            {t("subtitle")}
          </p>
        </div>

        <DevelopmentBanner />

        {/* Enable/disable */}
        <Card>
          <CardHeader>
            <CardTitle>{t("integrationStatus")}</CardTitle>
            <CardDescription>
              {t("enableOrDisableSendingInvoicesToKsef")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <Switch
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  setForm({ ...form, enabled: checked })
                }
              />
              <span className="text-sm">
                {form.enabled ? t("ksefEnabled") : t("ksefDisabled")}
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Auto-send */}
        <Card>
          <CardHeader>
            <CardTitle>{t("automaticSending")}</CardTitle>
            <CardDescription>
              {t("autoSendNewInvoicesToKsef")}
              {t("autoSendRetry")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <Switch
                checked={form.auto_send}
                onCheckedChange={(checked) =>
                  setForm({ ...form, auto_send: checked })
                }
                disabled={!form.enabled}
              />
              <span className="text-sm">
                {form.auto_send ? t("autosendEnabled") : t("autosendDisabled")}
              </span>
            </div>
            {!form.enabled && form.auto_send && (
              <p className="text-xs text-muted-foreground mt-2">
                {t("enableKsefForAutoSending")}
              </p>
            )}
          </CardContent>
        </Card>

        {/* Environment */}
        <Card>
          <CardHeader>
            <CardTitle>{t("srodowisko")}</CardTitle>
            <CardDescription>
              {t("environmentDescription")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-w-sm">
              <Label>{t("srodowisko")}</Label>
              <Select
                value={form.environment || "test"}
                onValueChange={(value) =>
                  setForm({ ...form, environment: value })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("wybierzSrodowisko")} />
                </SelectTrigger>
                <SelectContent>
                  {KSEF_ENVIRONMENTS.map((env) => (
                    <SelectItem key={env.value} value={env.value}>
                      {env.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Authentication */}
        <Card>
          <CardHeader>
            <CardTitle>{t("authData")}</CardTitle>
            <CardDescription>
              {t("authDescription")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>NIP</Label>
                <Input
                  value={form.nip}
                  onChange={(e) => setForm({ ...form, nip: e.target.value })}
                  placeholder="1234567890"
                  maxLength={10}
                />
                <p className="text-xs text-muted-foreground">
                  {t("nipHint")}
                </p>
              </div>
              <div className="space-y-2">
                <Label>{t("authToken")}</Label>
                <Input
                  type="password"
                  value={form.token}
                  onChange={(e) => setForm({ ...form, token: e.target.value })}
                  placeholder={t("authTokenPlaceholder")}
                />
                <p className="text-xs text-muted-foreground">
                  {t("generateTokenHint", {
                    portal: form.environment === "production"
                      ? "ksef.mf.gov.pl"
                      : "ksef-test.mf.gov.pl"
                  })}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Company details */}
        <Card>
          <CardHeader>
            <CardTitle>{t("companyData")}</CardTitle>
            <CardDescription>
              {t("daneFirmyUzywaneWFakturachStrukturalnychKsef")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>{t("companyName")}</Label>
              <Input
                value={form.company_name}
                onChange={(e) =>
                  setForm({ ...form, company_name: e.target.value })
                }
                placeholder={t("companyNamePlaceholder")}
              />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("street")}</Label>
                <Input
                  value={form.company_street}
                  onChange={(e) =>
                    setForm({ ...form, company_street: e.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>{t("city")}</Label>
                <Input
                  value={form.company_city}
                  onChange={(e) =>
                    setForm({ ...form, company_city: e.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>{t("postalCode")}</Label>
                <Input
                  value={form.company_postal}
                  onChange={(e) =>
                    setForm({ ...form, company_postal: e.target.value })
                  }
                  placeholder="00-001"
                />
              </div>
              <div className="space-y-2">
                <Label>{t("country")}</Label>
                <Input
                  value={form.company_country}
                  onChange={(e) =>
                    setForm({ ...form, company_country: e.target.value })
                  }
                  placeholder="PL"
                  maxLength={2}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Test connection */}
        <Card>
          <CardHeader>
            <CardTitle>{t("ksefConnectionTest")}</CardTitle>
            <CardDescription>
              {t("checkKsefConnectionWorks")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Button
              variant="outline"
              onClick={handleTestConnection}
              disabled={testConnection.isPending || !form.enabled}
            >
              {testConnection.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <TestTube className="mr-2 h-4 w-4" />
              )}
              {ts("testujPołaczenie")}
            </Button>

            {testResult && (
              <div
                className={`flex items-center gap-2 rounded-md border p-3 ${
                  testResult.success
                    ? "border-success/30 bg-success/15"
                    : "border-destructive/30 bg-destructive/15"
                }`}
              >
                {testResult.success ? (
                  <CheckCircle className="h-5 w-5 text-success" />
                ) : (
                  <XCircle className="h-5 w-5 text-destructive" />
                )}
                <span className="text-sm">{testResult.message}</span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Save */}
        <div className="flex justify-end">
          <Button onClick={handleSave} disabled={updateSettings.isPending}>
            {updateSettings.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            {ts("saveSettings")}
          </Button>
        </div>
      </div>
    </AdminGuard>
  );
}
