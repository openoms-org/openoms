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
  useInvoicingSettings,
  useUpdateInvoicingSettings,
} from "@/hooks/use-invoices";
import { Loader2, Save } from "lucide-react";
import { INVOICING_PROVIDER_LABELS, ORDER_STATUSES } from "@/lib/constants";
import type { InvoicingSettings } from "@/types/api";

const DEFAULT_SETTINGS: InvoicingSettings = {
  provider: "",
  auto_create_on_status: [],
  default_tax_rate: 23,
  payment_days: 14,
  credentials: {},
};

export default function InvoicingSettingsPage() {
  const { data: settings, isLoading } = useInvoicingSettings();
  const updateSettings = useUpdateInvoicingSettings();

  const [form, setForm] = useState<InvoicingSettings>(DEFAULT_SETTINGS);

  useEffect(() => {
    if (settings) {
      setForm({
        ...DEFAULT_SETTINGS,
        ...settings,
        credentials: settings.credentials || {},
      });
    }
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success("Ustawienia fakturowania zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać ustawień";
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
        <h1 className="text-2xl font-bold">Fakturowanie</h1>
        <p className="text-muted-foreground">
          Konfiguracja automatycznego wystawiania faktur
        </p>
      </div>

      {/* Provider selection */}
      <Card>
        <CardHeader>
          <CardTitle>Dostawca faktur</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Dostawca</Label>
            <Select
              value={form.provider || "none"}
              onValueChange={(value) =>
                setForm({ ...form, provider: value === "none" ? "" : value })
              }
            >
              <SelectTrigger className="w-full max-w-xs">
                <SelectValue placeholder="Wybierz dostawcę" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Brak</SelectItem>
                {Object.entries(INVOICING_PROVIDER_LABELS).map(([key, label]) => (
                  <SelectItem key={key} value={key}>{label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Credentials */}
      {form.provider === "fakturownia" && (
        <Card>
          <CardHeader>
            <CardTitle>Dane dostępowe Fakturownia</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Subdomena</Label>
                <Input
                  value={form.credentials.subdomain || ""}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      credentials: { ...form.credentials, subdomain: e.target.value },
                    })
                  }
                  placeholder="moja-firma"
                />
                <p className="text-xs text-muted-foreground">
                  np. moja-firma dla moja-firma.fakturownia.pl
                </p>
              </div>
              <div className="space-y-2">
                <Label>Token API</Label>
                <Input
                  type="password"
                  value={form.credentials.api_token || ""}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      credentials: { ...form.credentials, api_token: e.target.value },
                    })
                  }
                  placeholder="Token API z Fakturowni"
                />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Invoice settings */}
      <Card>
        <CardHeader>
          <CardTitle>Ustawienia faktur</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Domyślna stawka VAT (%)</Label>
              <Input
                type="number"
                value={form.default_tax_rate}
                onChange={(e) =>
                  setForm({
                    ...form,
                    default_tax_rate: parseInt(e.target.value) || 23,
                  })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>Termin płatności (dni)</Label>
              <Input
                type="number"
                value={form.payment_days}
                onChange={(e) =>
                  setForm({
                    ...form,
                    payment_days: parseInt(e.target.value) || 14,
                  })
                }
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Auto-create triggers */}
      <Card>
        <CardHeader>
          <CardTitle>Automatyczne wystawianie</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            Wybierz przy jakich zmianach statusu zamówienia automatycznie wystawiać fakturę
          </p>
          <div className="space-y-3">
            {Object.entries(ORDER_STATUSES).map(([key, { label }]) => (
              <div key={key} className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id={`auto-invoice-${key}`}
                  checked={form.auto_create_on_status.includes(key)}
                  onChange={(e) => {
                    setForm({
                      ...form,
                      auto_create_on_status: e.target.checked
                        ? [...form.auto_create_on_status, key]
                        : form.auto_create_on_status.filter((s) => s !== key),
                    });
                  }}
                  className="h-4 w-4 rounded border-gray-300"
                />
                <label htmlFor={`auto-invoice-${key}`} className="text-sm">
                  {label}
                </label>
              </div>
            ))}
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
    </AdminGuard>
  );
}
