"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  useMarketingStatus,
  useSyncCustomers,
  useCreateCampaign,
} from "@/hooks/use-marketing";
import { useCompanySettings } from "@/hooks/use-settings";
import { Loader2, Save, RefreshCw, Send } from "lucide-react";
import { DevelopmentBanner } from "@/components/shared/development-banner";
import { apiClient } from "@/lib/api-client";
import type { MailchimpSettings } from "@/types/api";
import { useTranslations } from "next-intl";

const DEFAULT_SETTINGS: MailchimpSettings = {
  api_key: "",
  list_id: "",
  enabled: false,
};

export default function MarketingSettingsPage() {
  const t = useTranslations("settings");
  const tc = useTranslations("common");
  const { data: status, isLoading: statusLoading } = useMarketingStatus();
  const syncCustomers = useSyncCustomers();
  const createCampaign = useCreateCampaign();

  const { data: companySettings } = useCompanySettings();

  const [form, setForm] = useState<MailchimpSettings>(DEFAULT_SETTINGS);
  const [campaignForm, setCampaignForm] = useState({
    name: "",
    subject: "",
    content: "",
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const settings = companySettings as Record<string, unknown> | undefined;
    const mc = settings?.mailchimp as MailchimpSettings | undefined;
    if (mc) {
      setForm({
        api_key: mc.api_key ?? "",
        list_id: mc.list_id ?? "",
        enabled: mc.enabled ?? false,
      });
    }
  }, [companySettings]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const current = companySettings || {};
      await apiClient("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify({
          ...current,
          mailchimp: form,
        }),
      });
      toast.success(t("marketing.mailchimpSaved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("nieUdałoSieZapisacUstawien");
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  const handleSync = async () => {
    try {
      const result = await syncCustomers.mutateAsync();
      toast.success(t("marketing.syncCompleted", { synced: result.synced, failed: result.failed }));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("nieUdałoSieZsynchronizowac");
      toast.error(message);
    }
  };

  const handleCreateCampaign = async () => {
    try {
      const result = await createCampaign.mutateAsync(campaignForm);
      toast.success(t("marketing.campaignCreated", { id: result.campaign_id }));
      setCampaignForm({ name: "", subject: "", content: "" });
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("nieUdałoSieUtworzycKampanii");
      toast.error(message);
    }
  };

  if (statusLoading) {
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
          <h1 className="text-2xl font-bold">{t("marketing.title")}</h1>
          <p className="text-muted-foreground">
            {t("synchronizujKlientowZMailchimpITworzenieKampanii")}
          </p>
        </div>

        <DevelopmentBanner />

        {/* Status card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("marketing.integrationStatus")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div
                  className={`h-3 w-3 rounded-full ${
                    status?.configured ? "bg-success" : "bg-destructive"
                  }`}
                />
                <span className="text-sm">
                  Mailchimp: {status?.configured ? t("marketing.configured") : t("marketing.notConfigured")}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div
                  className={`h-3 w-3 rounded-full ${
                    status?.enabled ? "bg-success" : "bg-muted-foreground"
                  }`}
                />
                <span className="text-sm">
                  {status?.enabled ? t("właczony") : t("wyłaczony1")}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Configuration card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("marketing.mailchimpConfig")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">{t("marketing.activeIntegration")}</p>
                <p className="text-sm text-muted-foreground">
                  {t("właczSynchronizacjeKlientowZMailchimp")}
                </p>
              </div>
              <Switch
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  setForm({ ...form, enabled: checked })
                }
              />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("marketing.apiKey")}</Label>
                <Input
                  type="password"
                  value={form.api_key}
                  onChange={(e) =>
                    setForm({ ...form, api_key: e.target.value })
                  }
                  placeholder={t("marketing.apiKeyPlaceholder")}
                />
                <p className="text-xs text-muted-foreground">
                  {t("marketing.apiKeyHint")}
                </p>
              </div>
              <div className="space-y-2">
                <Label>{t("marketing.listId")}</Label>
                <Input
                  value={form.list_id}
                  onChange={(e) =>
                    setForm({ ...form, list_id: e.target.value })
                  }
                  placeholder="e.g. abc1234567"
                />
                <p className="text-xs text-muted-foreground">
                  {t("marketing.listIdHint")}
                </p>
              </div>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleSave} disabled={saving}>
                {saving ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                {t("saveSettings")}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Sync card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("synchronizacjaKlientow")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              {t("synchronizujWszystkichKlientowZAdresemEmailDo")}
            </p>
            <Button
              onClick={handleSync}
              disabled={syncCustomers.isPending}
              variant="outline"
            >
              {syncCustomers.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              {t("marketing.syncNow")}
            </Button>
          </CardContent>
        </Card>

        {/* Campaign creation card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("marketing.newCampaign")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("marketing.campaignName")}</Label>
                <Input
                  value={campaignForm.name}
                  onChange={(e) =>
                    setCampaignForm({ ...campaignForm, name: e.target.value })
                  }
                  placeholder={t("marketing.campaignName")}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("marketing.emailSubject")}</Label>
                <Input
                  value={campaignForm.subject}
                  onChange={(e) =>
                    setCampaignForm({ ...campaignForm, subject: e.target.value })
                  }
                  placeholder={t("marketing.emailSubject")}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("trescHtml")}</Label>
              <Textarea
                value={campaignForm.content}
                onChange={(e) =>
                  setCampaignForm({ ...campaignForm, content: e.target.value })
                }
                placeholder="<html><body>...</body></html>"
                rows={6}
                className="font-mono text-sm"
              />
            </div>
            <div className="flex justify-end">
              <Button
                onClick={handleCreateCampaign}
                disabled={
                  createCampaign.isPending ||
                  !campaignForm.name ||
                  !campaignForm.subject ||
                  !campaignForm.content
                }
              >
                {createCampaign.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
                {t("utworzKampanie")}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
