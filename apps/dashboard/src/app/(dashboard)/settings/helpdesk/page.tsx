"use client";

import { useState } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { useAllTickets } from "@/hooks/use-helpdesk";
import { useCompanySettings } from "@/hooks/use-settings";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Loader2, Save } from "lucide-react";
import { DevelopmentBanner } from "@/components/shared/development-banner";
import { apiClient } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import type { FreshdeskSettings } from "@/types/api";
import { useTranslations } from "next-intl";

const DEFAULT_SETTINGS: FreshdeskSettings = {
  domain: "",
  api_key: "",
  enabled: false,
};

export default function HelpdeskSettingsPage() {
  const t = useTranslations("settings");
  const { data: ticketsData, isLoading: ticketsLoading } = useAllTickets();
  const { data: companySettings } = useCompanySettings();

  const FRESHDESK_STATUS_LABELS: Record<number, string> = {
    2: t("helpdesk.statusOpen"),
    3: t("helpdesk.statusPending"),
    4: t("helpdesk.statusResolved"),
    5: t("helpdesk.statusClosed"),
  };

  const FRESHDESK_PRIORITY_LABELS: Record<number, string> = {
    1: t("helpdesk.priorityLow"),
    2: t("helpdesk.priorityMedium"),
    3: t("helpdesk.priorityHigh"),
    4: t("helpdesk.priorityUrgent"),
  };

  const [form, setForm] = useState<FreshdeskSettings>(DEFAULT_SETTINGS);
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      const current = companySettings || {};
      await apiClient("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify({
          ...current,
          freshdesk: form,
        }),
      });
      toast.success(t("helpdesk.freshdeskSaved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("settingsSaveError");
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">{t("helpdesk.title")}</h1>
          <p className="text-muted-foreground">
            {t("freshdeskIntegrationForCustomerTickets")}
          </p>
        </div>

        <DevelopmentBanner />

        {/* Configuration card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("helpdesk.freshdeskConfig")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">{t("helpdesk.activeIntegration")}</p>
                <p className="text-sm text-muted-foreground">
                  {t("enableFreshdeskTicketCreation")}
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
                <Label>{t("helpdesk.freshdeskDomain")}</Label>
                <Input
                  value={form.domain}
                  onChange={(e) =>
                    setForm({ ...form, domain: e.target.value })
                  }
                  placeholder={t("helpdesk.domainPlaceholder")}
                />
                <p className="text-xs text-muted-foreground">
                  {t("helpdesk.domainHint")}
                </p>
              </div>
              <div className="space-y-2">
                <Label>{t("helpdesk.apiKey")}</Label>
                <Input
                  type="password"
                  value={form.api_key}
                  onChange={(e) =>
                    setForm({ ...form, api_key: e.target.value })
                  }
                  placeholder={t("helpdesk.apiKeyPlaceholder")}
                />
                <p className="text-xs text-muted-foreground">
                  {t("helpdesk.apiKeyHint")}
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

        {/* Recent tickets card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("recentTickets")}</CardTitle>
          </CardHeader>
          <CardContent>
            {ticketsLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : ticketsData?.tickets && ticketsData.tickets.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t("helpdesk.columns.subject")}</TableHead>
                    <TableHead>{t("helpdesk.columns.status")}</TableHead>
                    <TableHead>{t("helpdesk.columns.priority")}</TableHead>
                    <TableHead>{t("helpdesk.columns.createdAt")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {ticketsData.tickets.map((ticket) => (
                    <TableRow key={ticket.id}>
                      <TableCell className="font-mono text-sm">
                        #{ticket.id}
                      </TableCell>
                      <TableCell className="font-medium">
                        {ticket.subject}
                      </TableCell>
                      <TableCell>
                        <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium">
                          {FRESHDESK_STATUS_LABELS[ticket.status] || `Status ${ticket.status}`}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs">
                          {FRESHDESK_PRIORITY_LABELS[ticket.priority] || `Priority ${ticket.priority}`}
                        </span>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(ticket.created_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">
                {t("noTicketsConfigureFreshdesk")}
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
