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
  useExportSettings,
  useImportSettings,
} from "@/hooks/use-settings";
import { uploadFile, getErrorMessage } from "@/lib/api-client";
import { Loader2, Save, Upload, Download, Building2 } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import type { CompanySettings } from "@/types/api";
import { useTranslations } from "next-intl";

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
  const t = useTranslations("settings");
  const { data: settings, isLoading } = useCompanySettings();
  const updateSettings = useUpdateCompanySettings();
  const exportSettings = useExportSettings();
  const importSettings = useImportSettings();

  const [form, setForm] = useState<CompanySettings>(DEFAULT_SETTINGS);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const importFileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (settings) setForm(settings);
  }, [settings]);

  const validateField = (field: string, value: string) => {
    let error = "";
    if (field === "company_name" && !value.trim()) {
      error = t("company.companyNameRequired");
    }
    if (field === "email" && value.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
      error = t("company.emailInvalid");
    }
    setFieldErrors((prev) => {
      const next = { ...prev };
      if (error) {
        next[field] = error;
      } else {
        delete next[field];
      }
      return next;
    });
  };

  const handleSave = async () => {
    // Validate required fields before saving
    const errors: Record<string, string> = {};
    if (!form.company_name.trim()) {
      errors.company_name = t("company.companyNameRequired");
    }
    if (form.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
      errors.email = t("company.emailInvalid");
    }
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      toast.error(t("company.fixFormErrors"));
      return;
    }
    setFieldErrors({});
    try {
      await updateSettings.mutateAsync(form);
      toast.success(t("company.dataSaved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("company.saveFailed");
      toast.error(message);
    }
  };

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (file.size > 5 * 1024 * 1024) {
      toast.error(t("form.fileTooLarge"));
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
      return;
    }

    setUploading(true);
    try {
      const result = await uploadFile(file);
      setForm({ ...form, logo_url: result.url });
      toast.success(t("company.logoUploaded"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("company.logoUploadFailed");
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
    <div className="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("company.title")}</h1>
        <p className="text-muted-foreground">
          {t("company.subtitle")}
        </p>
      </div>

      {/* Logo card */}
      <Card>
        <CardHeader>
          <CardTitle>{t("company.logo")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-6">
            <div className="flex h-24 w-24 items-center justify-center rounded-lg border bg-muted">
              {form.logo_url ? (
                <img
                  src={form.logo_url}
                  alt={t("company.logo")}
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
                accept="image/jpeg,image/png,image/webp"
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
                {t("company.uploadLogo")}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Company info card */}
      <Card>
        <CardHeader>
          <CardTitle>{t("company.companyInfo")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("company.companyName")} <span className="text-destructive">*</span></Label>
              <Input
                value={form.company_name}
                aria-invalid={!!fieldErrors.company_name}
                onChange={(e) =>
                  setForm({ ...form, company_name: e.target.value })
                }
                onBlur={(e) => validateField("company_name", e.target.value)}
              />
              {fieldErrors.company_name && (
                <p className="text-destructive text-xs mt-1">{fieldErrors.company_name}</p>
              )}
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
              <Label>{t("company.address")}</Label>
              <Input
                value={form.address}
                onChange={(e) =>
                  setForm({ ...form, address: e.target.value })
                }
                placeholder={t("form.streetPlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("company.city")}</Label>
              <Input
                value={form.city}
                onChange={(e) =>
                  setForm({ ...form, city: e.target.value })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>{t("company.postCode")}</Label>
              <Input
                value={form.post_code}
                onChange={(e) =>
                  setForm({ ...form, post_code: e.target.value })
                }
                placeholder="00-000"
              />
            </div>
            <div className="space-y-2">
              <Label>{t("company.phone")}</Label>
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
                aria-invalid={!!fieldErrors.email}
                onChange={(e) =>
                  setForm({ ...form, email: e.target.value })
                }
                onBlur={(e) => validateField("email", e.target.value)}
              />
              {fieldErrors.email && (
                <p className="text-destructive text-xs mt-1">{fieldErrors.email}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label>{t("company.website")}</Label>
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
          {t("company.saveCompanyData")}
        </Button>
      </div>

      <Separator />

      {/* Export / Import settings */}
      <Card>
        <CardHeader>
          <CardTitle>{t("company.exportImportSettings")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {t("company.exportDescription")}
          </p>
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              onClick={() =>
                exportSettings.mutate(undefined, {
                  onSuccess: () => toast.success(t("company.settingsExported")),
                  onError: (err) => toast.error(getErrorMessage(err)),
                })
              }
              disabled={exportSettings.isPending}
            >
              {exportSettings.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Download className="mr-2 h-4 w-4" />
              )}
              {t("company.exportSettings")}
            </Button>
            <input
              ref={importFileInputRef}
              type="file"
              accept=".json,application/json"
              className="hidden"
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (!file) return;
                let data: Record<string, unknown>;
                try {
                  data = JSON.parse(await file.text());
                } catch {
                  toast.error(t("company.settingsImportInvalid"));
                  if (importFileInputRef.current) importFileInputRef.current.value = "";
                  return;
                }
                importSettings.mutate(data, {
                  onSuccess: () => toast.success(t("company.settingsImported")),
                  onError: (err) => toast.error(getErrorMessage(err)),
                  onSettled: () => {
                    if (importFileInputRef.current) {
                      importFileInputRef.current.value = "";
                    }
                  },
                });
              }}
            />
            <Button
              variant="outline"
              onClick={() => importFileInputRef.current?.click()}
              disabled={importSettings.isPending}
            >
              {importSettings.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Upload className="mr-2 h-4 w-4" />
              )}
              {t("company.importSettings")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
    </AdminGuard>
  );
}
