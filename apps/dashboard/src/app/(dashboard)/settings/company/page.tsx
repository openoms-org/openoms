"use client";

import { useState, useEffect, useRef } from "react";
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
  useCompanySettings,
  useUpdateCompanySettings,
} from "@/hooks/use-settings";
import { uploadFile } from "@/lib/api-client";
import { Loader2, Save, Upload, Building2 } from "lucide-react";
import type { CompanySettings } from "@/types/api";

const DEFAULT_SETTINGS: CompanySettings = {
  company_name: "",
  logo_url: "",
  address: "",
  city: "",
  post_code: "",
  nip: "",
  phone: "",
  email: "",
  website: "",
};

export default function CompanySettingsPage() {
  const { data: settings, isLoading } = useCompanySettings();
  const updateSettings = useUpdateCompanySettings();

  const [form, setForm] = useState<CompanySettings>(DEFAULT_SETTINGS);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (settings) setForm(settings);
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success("Dane firmy zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać danych firmy";
      toast.error(message);
    }
  };

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    try {
      const result = await uploadFile(file);
      setForm({ ...form, logo_url: result.url });
      toast.success("Logo wgrane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się wgrać logo";
      toast.error(message);
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
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
        <h1 className="text-2xl font-bold">Dane firmy</h1>
        <p className="text-muted-foreground">
          Informacje o firmie widoczne na dokumentach
        </p>
      </div>

      {/* Logo card */}
      <Card>
        <CardHeader>
          <CardTitle>Logo firmy</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-6">
            <div className="flex h-24 w-24 items-center justify-center rounded-lg border bg-muted">
              {form.logo_url ? (
                <img
                  src={form.logo_url}
                  alt="Logo firmy"
                  className="h-full w-full rounded-lg object-contain"
                />
              ) : (
                <Building2 className="h-10 w-10 text-muted-foreground" />
              )}
            </div>
            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={handleLogoUpload}
              />
              <Button
                variant="outline"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
              >
                {uploading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Upload className="h-4 w-4" />
                )}
                Wgraj logo
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Company info card */}
      <Card>
        <CardHeader>
          <CardTitle>Informacje o firmie</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Nazwa firmy</Label>
              <Input
                value={form.company_name}
                onChange={(e) =>
                  setForm({ ...form, company_name: e.target.value })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>NIP</Label>
              <Input
                value={form.nip}
                onChange={(e) =>
                  setForm({ ...form, nip: e.target.value })
                }
                placeholder="1234567890"
              />
            </div>
            <div className="space-y-2">
              <Label>Adres</Label>
              <Input
                value={form.address}
                onChange={(e) =>
                  setForm({ ...form, address: e.target.value })
                }
                placeholder="ul. Przykładowa 1"
              />
            </div>
            <div className="space-y-2">
              <Label>Miasto</Label>
              <Input
                value={form.city}
                onChange={(e) =>
                  setForm({ ...form, city: e.target.value })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>Kod pocztowy</Label>
              <Input
                value={form.post_code}
                onChange={(e) =>
                  setForm({ ...form, post_code: e.target.value })
                }
                placeholder="00-000"
              />
            </div>
            <div className="space-y-2">
              <Label>Telefon</Label>
              <Input
                value={form.phone}
                onChange={(e) =>
                  setForm({ ...form, phone: e.target.value })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>Email</Label>
              <Input
                type="email"
                value={form.email}
                onChange={(e) =>
                  setForm({ ...form, email: e.target.value })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>Strona WWW</Label>
              <Input
                type="url"
                value={form.website}
                onChange={(e) =>
                  setForm({ ...form, website: e.target.value })
                }
                placeholder="https://..."
              />
            </div>
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
          Zapisz dane firmy
        </Button>
      </div>
    </div>
    </AdminGuard>
  );
}
