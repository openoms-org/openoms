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

const DEFAULT_SETTINGS: KSeFSettings = {
  enabled: false,
  environment: "test",
  nip: "",
  token: "",
  company_name: "",
  company_street: "",
  company_city: "",
  company_postal: "",
  company_country: "PL",
};

export default function KSeFSettingsPage() {
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
      setForm({
        ...DEFAULT_SETTINGS,
        ...settings,
      });
    }
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success("Ustawienia KSeF zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać ustawień";
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
        err instanceof Error ? err.message : "Nie udało się przetestować połączenia";
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
          <h1 className="text-2xl font-bold">KSeF - Krajowy System e-Faktur</h1>
          <p className="text-muted-foreground">
            Konfiguracja integracji z Krajowym Systemem e-Faktur
          </p>
        </div>

        <DevelopmentBanner />

        {/* Enable/disable */}
        <Card>
          <CardHeader>
            <CardTitle>Status integracji</CardTitle>
            <CardDescription>
              Włącz lub wyłącz wysyłanie faktur do KSeF
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
                {form.enabled ? "KSeF włączony" : "KSeF wyłączony"}
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Environment */}
        <Card>
          <CardHeader>
            <CardTitle>Środowisko</CardTitle>
            <CardDescription>
              Wybierz środowisko KSeF (testowe do testów, produkcyjne do wysyłki prawdziwych faktur)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-w-sm">
              <Label>Środowisko</Label>
              <Select
                value={form.environment || "test"}
                onValueChange={(value) =>
                  setForm({ ...form, environment: value })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Wybierz środowisko" />
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
            <CardTitle>Dane autoryzacyjne</CardTitle>
            <CardDescription>
              NIP firmy i token autoryzacyjny z portalu KSeF
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
                  10-cyfrowy numer identyfikacji podatkowej
                </p>
              </div>
              <div className="space-y-2">
                <Label>Token autoryzacyjny</Label>
                <Input
                  type="password"
                  value={form.token}
                  onChange={(e) => setForm({ ...form, token: e.target.value })}
                  placeholder="Token z portalu KSeF"
                />
                <p className="text-xs text-muted-foreground">
                  Wygeneruj token w portalu{" "}
                  {form.environment === "production"
                    ? "ksef.mf.gov.pl"
                    : "ksef-test.mf.gov.pl"}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Company details */}
        <Card>
          <CardHeader>
            <CardTitle>Dane firmy (sprzedawca)</CardTitle>
            <CardDescription>
              Dane firmy używane w fakturach strukturalnych KSeF
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Nazwa firmy</Label>
              <Input
                value={form.company_name}
                onChange={(e) =>
                  setForm({ ...form, company_name: e.target.value })
                }
                placeholder="Nazwa firmy sp. z o.o."
              />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Ulica</Label>
                <Input
                  value={form.company_street}
                  onChange={(e) =>
                    setForm({ ...form, company_street: e.target.value })
                  }
                  placeholder="ul. Przykładowa 1"
                />
              </div>
              <div className="space-y-2">
                <Label>Miasto</Label>
                <Input
                  value={form.company_city}
                  onChange={(e) =>
                    setForm({ ...form, company_city: e.target.value })
                  }
                  placeholder="Warszawa"
                />
              </div>
              <div className="space-y-2">
                <Label>Kod pocztowy</Label>
                <Input
                  value={form.company_postal}
                  onChange={(e) =>
                    setForm({ ...form, company_postal: e.target.value })
                  }
                  placeholder="00-001"
                />
              </div>
              <div className="space-y-2">
                <Label>Kraj</Label>
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
            <CardTitle>Test połączenia</CardTitle>
            <CardDescription>
              Sprawdź czy połączenie z KSeF działa poprawnie
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
              Testuj połączenie
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
            Zapisz ustawienia
          </Button>
        </div>
      </div>
    </AdminGuard>
  );
}
