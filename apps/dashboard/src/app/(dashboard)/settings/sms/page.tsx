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
  useSMSSettings,
  useUpdateSMSSettings,
  useSendTestSMS,
} from "@/hooks/use-sms-settings";
import { Loader2, Send, Save } from "lucide-react";
import type { SMSSettings } from "@/types/api";

const NOTIFICATION_STATUSES = [
  { value: "shipped", label: "Wysyłka zamówienia" },
  { value: "delivered", label: "Dostarczenie zamówienia" },
  { value: "out_for_delivery", label: "W doręczeniu" },
  { value: "in_transit", label: "W transporcie" },
  { value: "cancelled", label: "Anulowanie zamówienia" },
];

const DEFAULT_TEMPLATES: Record<string, string> = {
  shipped:
    "Twoje zamowienie {{.OrderNumber}} zostalo wyslane. Numer przesylki: {{.TrackingNumber}}",
  delivered: "Twoje zamowienie {{.OrderNumber}} zostalo dostarczone.",
  out_for_delivery:
    "Twoje zamowienie {{.OrderNumber}} jest w doreczeniu. Sledz: {{.TrackingURL}}",
  in_transit:
    "Twoje zamowienie {{.OrderNumber}} jest w transporcie. Numer przesylki: {{.TrackingNumber}}",
  cancelled: "Twoje zamowienie {{.OrderNumber}} zostalo anulowane.",
};

const DEFAULT_SETTINGS: SMSSettings = {
  enabled: false,
  api_token: "",
  from: "",
  notify_on: ["shipped", "delivered", "out_for_delivery"],
  templates: { ...DEFAULT_TEMPLATES },
};

export default function SMSSettingsPage() {
  const { data: settings, isLoading } = useSMSSettings();
  const updateSettings = useUpdateSMSSettings();
  const sendTest = useSendTestSMS();

  const [form, setForm] = useState<SMSSettings>(DEFAULT_SETTINGS);
  const [testPhone, setTestPhone] = useState("");

  useEffect(() => {
    if (settings) {
      setForm({
        ...settings,
        templates: { ...DEFAULT_TEMPLATES, ...settings.templates },
      });
    }
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success("Ustawienia SMS zapisane");
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Nie udalo sie zapisac ustawien SMS";
      toast.error(message);
    }
  };

  const handleTestSMS = async () => {
    try {
      await sendTest.mutateAsync(testPhone);
      toast.success("Testowy SMS wyslany");
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Nie udalo sie wyslac testowego SMS";
      toast.error(message);
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
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Powiadomienia SMS</h1>
        <p className="text-muted-foreground">
          Konfiguracja wysylki SMS do klientow przez SMSAPI.pl
        </p>
      </div>

      {/* Master toggle card */}
      <Card>
        <CardHeader>
          <CardTitle>Status</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Powiadomienia SMS</p>
              <p className="text-sm text-muted-foreground">
                Wysylaj automatyczne SMS-y przy zmianie statusu zamowienia
              </p>
            </div>
            <Switch
              checked={form.enabled}
              onCheckedChange={(checked) =>
                setForm({ ...form, enabled: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      {/* SMSAPI Configuration card */}
      <Card>
        <CardHeader>
          <CardTitle>Konfiguracja SMSAPI.pl</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Token API</Label>
              <Input
                type="password"
                value={form.api_token}
                onChange={(e) =>
                  setForm({ ...form, api_token: e.target.value })
                }
                placeholder="Wklej token z panelu SMSAPI.pl"
              />
              <p className="text-xs text-muted-foreground">
                Token znajdziesz w panelu SMSAPI.pl &rarr; Ustawienia &rarr; API
              </p>
            </div>
            <div className="space-y-2">
              <Label>Nazwa nadawcy</Label>
              <Input
                value={form.from}
                onChange={(e) => setForm({ ...form, from: e.target.value })}
                placeholder="OpenOMS"
              />
              <p className="text-xs text-muted-foreground">
                Nazwa zarejestrowana w panelu SMSAPI.pl
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Notification triggers card */}
      <Card>
        <CardHeader>
          <CardTitle>Wyzwalacze</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            Wybierz przy jakich zmianach statusu wysylac SMS do klienta
          </p>
          <div className="space-y-3">
            {NOTIFICATION_STATUSES.map(({ value, label }) => (
              <div key={value} className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id={`sms-notify-${value}`}
                  checked={form.notify_on.includes(value)}
                  onChange={(e) => {
                    setForm({
                      ...form,
                      notify_on: e.target.checked
                        ? [...form.notify_on, value]
                        : form.notify_on.filter((s) => s !== value),
                    });
                  }}
                  className="h-4 w-4 rounded border-border"
                />
                <label htmlFor={`sms-notify-${value}`} className="text-sm">
                  {label}
                </label>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Templates card */}
      <Card>
        <CardHeader>
          <CardTitle>Szablony wiadomosci</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Dostepne zmienne:{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded dark:bg-muted/50">
              {"{{.OrderNumber}}"}
            </code>{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded dark:bg-muted/50">
              {"{{.Status}}"}
            </code>{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded dark:bg-muted/50">
              {"{{.TrackingNumber}}"}
            </code>{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded dark:bg-muted/50">
              {"{{.TrackingURL}}"}
            </code>{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded dark:bg-muted/50">
              {"{{.CustomerName}}"}
            </code>
          </p>
          {NOTIFICATION_STATUSES.map(({ value, label }) => (
            <div key={value} className="space-y-2">
              <Label>{label}</Label>
              <Textarea
                value={form.templates[value] || ""}
                onChange={(e) =>
                  setForm({
                    ...form,
                    templates: {
                      ...form.templates,
                      [value]: e.target.value,
                    },
                  })
                }
                placeholder={DEFAULT_TEMPLATES[value] || ""}
                rows={2}
                className="font-mono text-sm"
              />
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Test SMS card */}
      <Card>
        <CardHeader>
          <CardTitle>Testowy SMS</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2 max-w-md">
            <Input
              value={testPhone}
              onChange={(e) => setTestPhone(e.target.value)}
              placeholder="48123456789"
              type="tel"
            />
            <Button
              variant="outline"
              onClick={handleTestSMS}
              disabled={!testPhone || sendTest.isPending}
            >
              {sendTest.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
              Wyslij
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            Numer telefonu w formacie E.164 (np. 48123456789)
          </p>
        </CardContent>
      </Card>

      {/* Save button */}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateSettings.isPending}>
          {updateSettings.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          Zapisz ustawienia
        </Button>
      </div>
    </div>
    </AdminGuard>
  );
}
