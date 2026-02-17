"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  useAccountingSettings,
  useUpdateAccountingSettings,
  useTestAccountingConnection,
} from "@/hooks/use-invoices";
import {
  Loader2,
  Save,
  TestTube,
  CheckCircle,
  XCircle,
  ExternalLink,
} from "lucide-react";
import { INVOICING_PROVIDER_LABELS } from "@/lib/constants";

interface ProviderInfo {
  name: string;
  description: string;
  url: string;
  credentialFields: {
    key: string;
    label: string;
    placeholder: string;
    type?: string;
  }[];
}

const PROVIDER_INFO: Record<string, ProviderInfo> = {
  fakturownia: {
    name: "Fakturownia",
    description:
      "Popularny polski system fakturowania online. Obsługuje faktury VAT, proforma, korekty. Integracja przez API REST.",
    url: "https://fakturownia.pl",
    credentialFields: [
      {
        key: "subdomain",
        label: "Subdomena",
        placeholder: "moja-firma",
      },
      {
        key: "api_token",
        label: "Token API",
        placeholder: "Token API z Fakturowni",
        type: "password",
      },
    ],
  },
  wfirma: {
    name: "wFirma",
    description:
      "System ERP i fakturowania dla firm. Obsługuje pełną księgowość, faktury VAT, JPK. API REST z autoryzacją kluczem API.",
    url: "https://wfirma.pl",
    credentialFields: [
      {
        key: "api_key",
        label: "Klucz API",
        placeholder: "Klucz API z wFirma",
        type: "password",
      },
    ],
  },
  infakt: {
    name: "inFakt",
    description:
      "Prosty system fakturowania dla freelancerów i małych firm. Automatyczna wysyłka faktur, rozliczenia. API REST v3.",
    url: "https://www.infakt.pl",
    credentialFields: [
      {
        key: "api_key",
        label: "Klucz API",
        placeholder: "Klucz API z inFakt",
        type: "password",
      },
    ],
  },
};

export default function AccountingSettingsPage() {
  const { data: config, isLoading } = useAccountingSettings();
  const updateSettings = useUpdateAccountingSettings();
  const testConnection = useTestAccountingConnection();

  const [provider, setProvider] = useState("");
  const [credentials, setCredentials] = useState<Record<string, string>>({});
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  useEffect(() => {
    if (config) {
      setProvider(config.provider || "");
      setCredentials(config.credentials || {});
    }
  }, [config]);

  const handleProviderChange = (value: string) => {
    const newProvider = value === "none" ? "" : value;
    setProvider(newProvider);
    // Reset credentials when switching providers
    if (newProvider !== provider) {
      setCredentials({});
      setTestResult(null);
    }
  };

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync({ provider, credentials });
      toast.success("Ustawienia księgowości zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać ustawień";
      toast.error(message);
    }
  };

  const handleTestConnection = async () => {
    setTestResult(null);
    // Save first, then test
    try {
      await updateSettings.mutateAsync({ provider, credentials });
    } catch {
      toast.error("Zapisz ustawienia przed testem");
      return;
    }

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
        err instanceof Error
          ? err.message
          : "Nie udało się przetestować połączenia";
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

  const currentInfo = provider ? PROVIDER_INFO[provider] : null;

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Integracje księgowe</h1>
          <p className="text-muted-foreground">
            Połącz OpenOMS z systemem fakturowania: Fakturownia, wFirma lub
            inFakt
          </p>
        </div>

        {/* Provider selection */}
        <Card>
          <CardHeader>
            <CardTitle>Dostawca usług księgowych</CardTitle>
            <CardDescription>
              Wybierz system księgowy, z którym chcesz zintegrować OpenOMS
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Dostawca</Label>
              <Select
                value={provider || "none"}
                onValueChange={handleProviderChange}
              >
                <SelectTrigger className="w-full max-w-xs">
                  <SelectValue placeholder="Wybierz dostawcę" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Brak</SelectItem>
                  {Object.entries(INVOICING_PROVIDER_LABELS).map(
                    ([key, label]) => (
                      <SelectItem key={key} value={key}>
                        {label}
                      </SelectItem>
                    )
                  )}
                </SelectContent>
              </Select>
            </div>

            {/* Connection status */}
            {config && config.provider && config.connected && (
              <div className="flex items-center gap-2 rounded-md border border-success/30 bg-success/15 p-3">
                <CheckCircle className="h-5 w-5 text-success" />
                <span className="text-sm">
                  Połączono z {INVOICING_PROVIDER_LABELS[config.provider] || config.provider}
                </span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Provider info + credentials */}
        {currentInfo && (
          <Card>
            <CardHeader>
              <CardTitle>
                {currentInfo.name}
              </CardTitle>
              <CardDescription>{currentInfo.description}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <a
                href={currentInfo.url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-sm text-primary underline"
              >
                {currentInfo.url}
                <ExternalLink className="h-3 w-3" />
              </a>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                {currentInfo.credentialFields.map((field) => (
                  <div key={field.key} className="space-y-2">
                    <Label>{field.label}</Label>
                    <Input
                      type={field.type || "text"}
                      value={credentials[field.key] || ""}
                      onChange={(e) =>
                        setCredentials({
                          ...credentials,
                          [field.key]: e.target.value,
                        })
                      }
                      placeholder={field.placeholder}
                    />
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Test connection */}
        {provider && (
          <Card>
            <CardHeader>
              <CardTitle>Test połączenia</CardTitle>
              <CardDescription>
                Sprawdź czy dane dostępowe są poprawne
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Button
                variant="outline"
                onClick={handleTestConnection}
                disabled={testConnection.isPending || updateSettings.isPending}
              >
                {testConnection.isPending || updateSettings.isPending ? (
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
        )}

        {/* Save */}
        <div className="flex justify-end">
          <Button
            onClick={handleSave}
            disabled={updateSettings.isPending}
          >
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
