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
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  useEmailSettings,
  useUpdateEmailSettings,
  useSendTestEmail,
} from "@/hooks/use-settings";
import {
  useSMSSettings,
  useUpdateSMSSettings,
  useSendTestSMS,
} from "@/hooks/use-sms-settings";
import { Loader2, Send, Save } from "lucide-react";
import type { EmailSettings } from "@/types/api";
import type { SMSSettings } from "@/types/api";

const EMAIL_NOTIFICATION_STATUSES = [
  { value: "confirmed", label: "Potwierdzenie zamówienia" },
  { value: "shipped", label: "Wysyłka zamówienia" },
  { value: "delivered", label: "Dostarczenie zamówienia" },
  { value: "cancelled", label: "Anulowanie zamówienia" },
  { value: "refunded", label: "Zwrot środków" },
];

const SMS_NOTIFICATION_STATUSES = [
  { value: "shipped", label: "Wysyłka zamówienia" },
  { value: "delivered", label: "Dostarczenie zamówienia" },
  { value: "out_for_delivery", label: "W doręczeniu" },
  { value: "in_transit", label: "W transporcie" },
  { value: "cancelled", label: "Anulowanie zamówienia" },
];

const DEFAULT_EMAIL_SETTINGS: EmailSettings = {
  enabled: false,
  smtp_host: "",
  smtp_port: 587,
  smtp_user: "",
  smtp_pass: "",
  from_email: "",
  from_name: "",
  notify_on: ["confirmed", "shipped", "delivered", "cancelled", "refunded"],
};

const DEFAULT_TEMPLATES: Record<string, string> = {
  shipped:
    "Twoje zamówienie {{.OrderNumber}} zostało wysłane. Numer przesyłki: {{.TrackingNumber}}",
  delivered: "Twoje zamówienie {{.OrderNumber}} zostało dostarczone.",
  out_for_delivery:
    "Twoje zamówienie {{.OrderNumber}} jest w doręczeniu. Śledź: {{.TrackingURL}}",
  in_transit:
    "Twoje zamówienie {{.OrderNumber}} jest w transporcie. Numer przesyłki: {{.TrackingNumber}}",
  cancelled: "Twoje zamówienie {{.OrderNumber}} zostało anulowane.",
};

const DEFAULT_SMS_SETTINGS: SMSSettings = {
  enabled: false,
  api_token: "",
  from: "",
  notify_on: ["shipped", "delivered", "out_for_delivery"],
  templates: { ...DEFAULT_TEMPLATES },
};

export default function NotificationsPage() {
  const { data: emailSettings, isLoading: emailLoading } = useEmailSettings();
  const updateEmailSettings = useUpdateEmailSettings();
  const sendTestEmail = useSendTestEmail();

  const { data: smsSettings, isLoading: smsLoading } = useSMSSettings();
  const updateSMSSettings = useUpdateSMSSettings();
  const sendTestSMS = useSendTestSMS();

  const [emailForm, setEmailForm] = useState<EmailSettings>(DEFAULT_EMAIL_SETTINGS);
  const [testEmail, setTestEmail] = useState("");

  const [smsForm, setSmsForm] = useState<SMSSettings>(DEFAULT_SMS_SETTINGS);
  const [testPhone, setTestPhone] = useState("");

  useEffect(() => {
    if (emailSettings) setEmailForm(emailSettings);
  }, [emailSettings]);

  useEffect(() => {
    if (smsSettings) {
      setSmsForm({
        ...smsSettings,
        templates: { ...DEFAULT_TEMPLATES, ...smsSettings.templates },
      });
    }
  }, [smsSettings]);

  const handleEmailSave = async () => {
    try {
      await updateEmailSettings.mutateAsync(emailForm);
      toast.success("Ustawienia zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać ustawień";
      toast.error(message);
    }
  };

  const handleTestEmail = async () => {
    try {
      await sendTestEmail.mutateAsync(testEmail);
      toast.success("Testowy email wysłany");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się wysłać testowego emaila";
      toast.error(message);
    }
  };

  const handleSmsSave = async () => {
    try {
      await updateSMSSettings.mutateAsync(smsForm);
      toast.success("Ustawienia SMS zapisane");
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Nie udało się zapisać ustawień SMS";
      toast.error(message);
    }
  };

  const handleTestSMS = async () => {
    try {
      await sendTestSMS.mutateAsync(testPhone);
      toast.success("Testowy SMS wysłany");
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Nie udało się wysłać testowego SMS";
      toast.error(message);
    }
  };

  if (emailLoading || smsLoading) {
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
          <h1 className="text-2xl font-bold">Powiadomienia</h1>
          <p className="text-muted-foreground">
            Konfiguracja kanałów powiadomień do klientów
          </p>
        </div>

        <Tabs defaultValue="email">
          <TabsList>
            <TabsTrigger value="email">E-mail</TabsTrigger>
            <TabsTrigger value="sms">SMS</TabsTrigger>
          </TabsList>

          <TabsContent value="email">
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>Status</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium">Powiadomienia email</p>
                      <p className="text-sm text-muted-foreground">
                        Wysyłaj automatyczne emaile przy zmianie statusu zamówienia
                      </p>
                    </div>
                    <Switch
                      checked={emailForm.enabled}
                      onCheckedChange={(checked) =>
                        setEmailForm({ ...emailForm, enabled: checked })
                      }
                    />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Konfiguracja SMTP</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label>Host SMTP</Label>
                      <Input
                        value={emailForm.smtp_host}
                        onChange={(e) =>
                          setEmailForm({ ...emailForm, smtp_host: e.target.value })
                        }
                        placeholder="smtp.gmail.com"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Port</Label>
                      <Input
                        type="number"
                        value={emailForm.smtp_port}
                        onChange={(e) =>
                          setEmailForm({
                            ...emailForm,
                            smtp_port: parseInt(e.target.value) || 587,
                          })
                        }
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Użytkownik</Label>
                      <Input
                        value={emailForm.smtp_user}
                        onChange={(e) =>
                          setEmailForm({ ...emailForm, smtp_user: e.target.value })
                        }
                        placeholder="user@example.com"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Hasło</Label>
                      <Input
                        type="password"
                        value={emailForm.smtp_pass}
                        onChange={(e) =>
                          setEmailForm({ ...emailForm, smtp_pass: e.target.value })
                        }
                        placeholder="••••••"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Email nadawcy</Label>
                      <Input
                        value={emailForm.from_email}
                        onChange={(e) =>
                          setEmailForm({ ...emailForm, from_email: e.target.value })
                        }
                        placeholder="zamówienia@firma.pl"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Nazwa nadawcy</Label>
                      <Input
                        value={emailForm.from_name}
                        onChange={(e) =>
                          setEmailForm({ ...emailForm, from_name: e.target.value })
                        }
                        placeholder="Moja Firma"
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Wyzwalacze</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground mb-4">
                    Wybierz przy jakich zmianach statusu wysyłać email do klienta
                  </p>
                  <div className="space-y-3">
                    {EMAIL_NOTIFICATION_STATUSES.map(({ value, label }) => (
                      <div key={value} className="flex items-center gap-3">
                        <input
                          type="checkbox"
                          id={`email-notify-${value}`}
                          checked={emailForm.notify_on.includes(value)}
                          onChange={(e) => {
                            setEmailForm({
                              ...emailForm,
                              notify_on: e.target.checked
                                ? [...emailForm.notify_on, value]
                                : emailForm.notify_on.filter((s) => s !== value),
                            });
                          }}
                          className="h-4 w-4 rounded border-border"
                        />
                        <label htmlFor={`email-notify-${value}`} className="text-sm">
                          {label}
                        </label>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

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
                      disabled={!testEmail || sendTestEmail.isPending}
                    >
                      {sendTestEmail.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Send className="h-4 w-4" />
                      )}
                      Wyślij
                    </Button>
                  </div>
                </CardContent>
              </Card>

              <div className="flex justify-end">
                <Button onClick={handleEmailSave} disabled={updateEmailSettings.isPending}>
                  {updateEmailSettings.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  Zapisz ustawienia
                </Button>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="sms">
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>Status</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium">Powiadomienia SMS</p>
                      <p className="text-sm text-muted-foreground">
                        Wysyłaj automatyczne SMS-y przy zmianie statusu zamówienia
                      </p>
                    </div>
                    <Switch
                      checked={smsForm.enabled}
                      onCheckedChange={(checked) =>
                        setSmsForm({ ...smsForm, enabled: checked })
                      }
                    />
                  </div>
                </CardContent>
              </Card>

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
                        value={smsForm.api_token}
                        onChange={(e) =>
                          setSmsForm({ ...smsForm, api_token: e.target.value })
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
                        value={smsForm.from}
                        onChange={(e) => setSmsForm({ ...smsForm, from: e.target.value })}
                        placeholder="OpenOMS"
                      />
                      <p className="text-xs text-muted-foreground">
                        Nazwa zarejestrowana w panelu SMSAPI.pl
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Wyzwalacze</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground mb-4">
                    Wybierz przy jakich zmianach statusu wysyłać SMS do klienta
                  </p>
                  <div className="space-y-3">
                    {SMS_NOTIFICATION_STATUSES.map(({ value, label }) => (
                      <div key={value} className="flex items-center gap-3">
                        <input
                          type="checkbox"
                          id={`sms-notify-${value}`}
                          checked={smsForm.notify_on.includes(value)}
                          onChange={(e) => {
                            setSmsForm({
                              ...smsForm,
                              notify_on: e.target.checked
                                ? [...smsForm.notify_on, value]
                                : smsForm.notify_on.filter((s) => s !== value),
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

              <Card>
                <CardHeader>
                  <CardTitle>Szablony wiadomości</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    Dostępne zmienne:{" "}
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
                  {SMS_NOTIFICATION_STATUSES.map(({ value, label }) => (
                    <div key={value} className="space-y-2">
                      <Label>{label}</Label>
                      <Textarea
                        value={smsForm.templates[value] || ""}
                        onChange={(e) =>
                          setSmsForm({
                            ...smsForm,
                            templates: {
                              ...smsForm.templates,
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
                      disabled={!testPhone || sendTestSMS.isPending}
                    >
                      {sendTestSMS.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Send className="h-4 w-4" />
                      )}
                      Wyślij
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground mt-2">
                    Numer telefonu w formacie E.164 (np. 48123456789)
                  </p>
                </CardContent>
              </Card>

              <div className="flex justify-end">
                <Button onClick={handleSmsSave} disabled={updateSMSSettings.isPending}>
                  {updateSMSSettings.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  Zapisz ustawienia
                </Button>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </AdminGuard>
  );
}
