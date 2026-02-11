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
import { apiClient } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import type { FreshdeskSettings } from "@/types/api";

const DEFAULT_SETTINGS: FreshdeskSettings = {
  domain: "",
  api_key: "",
  enabled: false,
};

const FRESHDESK_STATUS_LABELS: Record<number, string> = {
  2: "Otwarty",
  3: "Oczekujacy",
  4: "Rozwiazany",
  5: "Zamkniety",
};

const FRESHDESK_PRIORITY_LABELS: Record<number, string> = {
  1: "Niski",
  2: "Sredni",
  3: "Wysoki",
  4: "Pilny",
};

export default function HelpdeskSettingsPage() {
  const { data: ticketsData, isLoading: ticketsLoading } = useAllTickets();
  const { data: companySettings } = useCompanySettings();

  const [form, setForm] = useState<FreshdeskSettings>(DEFAULT_SETTINGS);
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      // Save freshdesk settings under the tenant settings freshdesk key,
      // spreading existing company settings to avoid overwriting other fields.
      const current = companySettings || {};
      await apiClient("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify({
          ...current,
          freshdesk: form,
        }),
      });
      toast.success("Ustawienia Freshdesk zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udalo sie zapisac ustawien";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Helpdesk (Freshdesk)</h1>
          <p className="text-muted-foreground">
            Integracja z Freshdesk do obslugi zgloszen klientow
          </p>
        </div>

        {/* Configuration card */}
        <Card>
          <CardHeader>
            <CardTitle>Konfiguracja Freshdesk</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">Aktywna integracja</p>
                <p className="text-sm text-muted-foreground">
                  Wlacz tworzenie zgloszen w Freshdesk z poziomu zamowien
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
                <Label>Domena Freshdesk</Label>
                <Input
                  value={form.domain}
                  onChange={(e) =>
                    setForm({ ...form, domain: e.target.value })
                  }
                  placeholder="mojafirma"
                />
                <p className="text-xs text-muted-foreground">
                  Domena bez .freshdesk.com (np. &quot;mojafirma&quot; dla mojafirma.freshdesk.com)
                </p>
              </div>
              <div className="space-y-2">
                <Label>Klucz API</Label>
                <Input
                  type="password"
                  value={form.api_key}
                  onChange={(e) =>
                    setForm({ ...form, api_key: e.target.value })
                  }
                  placeholder="Wklej klucz API Freshdesk"
                />
                <p className="text-xs text-muted-foreground">
                  Klucz API znajdziesz w Freshdesk &rarr; Profil &rarr; API key
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

        {/* Recent tickets card */}
        <Card>
          <CardHeader>
            <CardTitle>Ostatnie zgloszenia</CardTitle>
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
                    <TableHead>Temat</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Priorytet</TableHead>
                    <TableHead>Utworzono</TableHead>
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
                          {FRESHDESK_PRIORITY_LABELS[ticket.priority] || `Priorytet ${ticket.priority}`}
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
                Brak zgloszen. Skonfiguruj Freshdesk aby zobaczyc zgloszenia.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
