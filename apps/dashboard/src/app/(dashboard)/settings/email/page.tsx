"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
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
import {
  useEmailSettings,
  useUpdateEmailSettings,
  useSendTestEmail,
} from "@/hooks/use-settings";
import { Loader2, Send, Save } from "lucide-react";
import type { EmailSettings } from "@/types/api";

const NOTIFICATION_STATUSES = [
  { value: "confirmed", label: "Potwierdzenie zamowienia" },
  { value: "shipped", label: "Wysylka zamowienia" },
  { value: "delivered", label: "Dostarczenie zamowienia" },
  { value: "cancelled", label: "Anulowanie zamowienia" },
  { value: "refunded", label: "Zwrot srodkow" },
];

const DEFAULT_SETTINGS: EmailSettings = {
  enabled: false,
  smtp_host: "",
  smtp_port: 587,
  smtp_user: "",
  smtp_pass: "",
  from_email: "",
  from_name: "",
  notify_on: ["confirmed", "shipped", "delivered", "cancelled", "refunded"],
};

export default function EmailSettingsPage() {
  const { data: settings, isLoading } = useEmailSettings();
  const updateSettings = useUpdateEmailSettings();
  const sendTest = useSendTestEmail();

  const [form, setForm] = useState<EmailSettings>(DEFAULT_SETTINGS);
  const [testEmail, setTestEmail] = useState("");

  useEffect(() => {
    if (settings) setForm(settings);
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success("Ustawienia zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udalo sie zapisac ustawien";
      toast.error(message);
    }
  };

  const handleTestEmail = async () => {
    try {
      await sendTest.mutateAsync(testEmail);
      toast.success("Testowy email wyslany");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udalo sie wyslac testowego emaila";
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
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Powiadomienia email</h1>
        <p className="text-muted-foreground">
          Konfiguracja wysylki emaili do klientow
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
              <p className="font-medium">Powiadomienia email</p>
              <p className="text-sm text-muted-foreground">
                Wysylaj automatyczne emaile przy zmianie statusu zamowienia
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

      {/* SMTP Configuration card */}
      <Card>
        <CardHeader>
          <CardTitle>Konfiguracja SMTP</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Host SMTP</Label>
              <Input
                value={form.smtp_host}
                onChange={(e) =>
                  setForm({ ...form, smtp_host: e.target.value })
                }
                placeholder="smtp.gmail.com"
              />
            </div>
            <div className="space-y-2">
              <Label>Port</Label>
              <Input
                type="number"
                value={form.smtp_port}
                onChange={(e) =>
                  setForm({
                    ...form,
                    smtp_port: parseInt(e.target.value) || 587,
                  })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>Uzytkownik</Label>
              <Input
                value={form.smtp_user}
                onChange={(e) =>
                  setForm({ ...form, smtp_user: e.target.value })
                }
                placeholder="user@example.com"
              />
            </div>
            <div className="space-y-2">
              <Label>Haslo</Label>
              <Input
                type="password"
                value={form.smtp_pass}
                onChange={(e) =>
                  setForm({ ...form, smtp_pass: e.target.value })
                }
                placeholder="••••••"
              />
            </div>
            <div className="space-y-2">
              <Label>Email nadawcy</Label>
              <Input
                value={form.from_email}
                onChange={(e) =>
                  setForm({ ...form, from_email: e.target.value })
                }
                placeholder="zamowienia@firma.pl"
              />
            </div>
            <div className="space-y-2">
              <Label>Nazwa nadawcy</Label>
              <Input
                value={form.from_name}
                onChange={(e) =>
                  setForm({ ...form, from_name: e.target.value })
                }
                placeholder="Moja Firma"
              />
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
            Wybierz przy jakich zmianach statusu wysylac email do klienta
          </p>
          <div className="space-y-3">
            {NOTIFICATION_STATUSES.map(({ value, label }) => (
              <div key={value} className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id={`notify-${value}`}
                  checked={form.notify_on.includes(value)}
                  onChange={(e) => {
                    setForm({
                      ...form,
                      notify_on: e.target.checked
                        ? [...form.notify_on, value]
                        : form.notify_on.filter((s) => s !== value),
                    });
                  }}
                  className="h-4 w-4 rounded border-gray-300"
                />
                <label htmlFor={`notify-${value}`} className="text-sm">
                  {label}
                </label>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Test email card */}
      <Card>
        <CardHeader>
          <CardTitle>Testowy email</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2 max-w-md">
            <Input
              value={testEmail}
              onChange={(e) => setTestEmail(e.target.value)}
              placeholder="test@example.com"
              type="email"
            />
            <Button
              variant="outline"
              onClick={handleTestEmail}
              disabled={!testEmail || sendTest.isPending}
            >
              {sendTest.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
              Wyslij
            </Button>
          </div>
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
  );
}
