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
import { useCompanySettings, useUpdateCompanySettings } from "@/hooks/use-settings";
import { Loader2, Save, RefreshCw, Send } from "lucide-react";
import { apiClient } from "@/lib/api-client";
import type { MailchimpSettings } from "@/types/api";

const DEFAULT_SETTINGS: MailchimpSettings = {
  api_key: "",
  list_id: "",
  enabled: false,
};

export default function MarketingSettingsPage() {
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
      // Save mailchimp settings under the tenant settings mailchimp key
      const current = companySettings || {};
      await apiClient("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify({
          ...current,
          mailchimp: form,
        }),
      });
      toast.success("Ustawienia Mailchimp zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać ustawień";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  const handleSync = async () => {
    try {
      const result = await syncCustomers.mutateAsync();
      toast.success(`Synchronizacja zakończona: ${result.synced} zsynchronizowanych, ${result.failed} błędów`);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zsynchronizować";
      toast.error(message);
    }
  };

  const handleCreateCampaign = async () => {
    try {
      const result = await createCampaign.mutateAsync(campaignForm);
      toast.success(`Kampania utworzona (ID: ${result.campaign_id})`);
      setCampaignForm({ name: "", subject: "", content: "" });
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się utworzyć kampanii";
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
          <h1 className="text-2xl font-bold">Marketing (Mailchimp)</h1>
          <p className="text-muted-foreground">
            Synchronizuj klientów z Mailchimp i tworzenie kampanii email
          </p>
        </div>

        {/* Status card */}
        <Card>
          <CardHeader>
            <CardTitle>Status integracji</CardTitle>
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
                  Mailchimp: {status?.configured ? "Skonfigurowany" : "Nieskonfigurowany"}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div
                  className={`h-3 w-3 rounded-full ${
                    status?.enabled ? "bg-success" : "bg-muted-foreground"
                  }`}
                />
                <span className="text-sm">
                  {status?.enabled ? "Włączony" : "Wyłączony"}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Configuration card */}
        <Card>
          <CardHeader>
            <CardTitle>Konfiguracja Mailchimp</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">Aktywna integracja</p>
                <p className="text-sm text-muted-foreground">
                  Włącz synchronizację klientów z Mailchimp
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
                <Label>Klucz API</Label>
                <Input
                  type="password"
                  value={form.api_key}
                  onChange={(e) =>
                    setForm({ ...form, api_key: e.target.value })
                  }
                  placeholder="Wklej klucz API Mailchimp"
                />
                <p className="text-xs text-muted-foreground">
                  Klucz API znajdziesz w Mailchimp &rarr; Account &rarr; API keys
                </p>
              </div>
              <div className="space-y-2">
                <Label>ID listy (Audience)</Label>
                <Input
                  value={form.list_id}
                  onChange={(e) =>
                    setForm({ ...form, list_id: e.target.value })
                  }
                  placeholder="np. abc1234567"
                />
                <p className="text-xs text-muted-foreground">
                  ID listy znajdziesz w Mailchimp &rarr; Audience &rarr; Settings
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
                Zapisz ustawienia
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Sync card */}
        <Card>
          <CardHeader>
            <CardTitle>Synchronizacja klientów</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              Synchronizuj wszystkich klientów z adresem email do listy Mailchimp.
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
              Synchronizuj teraz
            </Button>
          </CardContent>
        </Card>

        {/* Campaign creation card */}
        <Card>
          <CardHeader>
            <CardTitle>Nowa kampania</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Nazwa kampanii</Label>
                <Input
                  value={campaignForm.name}
                  onChange={(e) =>
                    setCampaignForm({ ...campaignForm, name: e.target.value })
                  }
                  placeholder="Nazwa kampanii"
                />
              </div>
              <div className="space-y-2">
                <Label>Temat wiadomości</Label>
                <Input
                  value={campaignForm.subject}
                  onChange={(e) =>
                    setCampaignForm({ ...campaignForm, subject: e.target.value })
                  }
                  placeholder="Temat emaila"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Treść HTML</Label>
              <Textarea
                value={campaignForm.content}
                onChange={(e) =>
                  setCampaignForm({ ...campaignForm, content: e.target.value })
                }
                placeholder="<html><body>Treść kampanii...</body></html>"
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
                Utwórz kampanię
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
