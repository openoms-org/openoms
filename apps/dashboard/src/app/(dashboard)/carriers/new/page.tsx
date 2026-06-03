"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { ArrowLeft, Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateIntegration } from "@/hooks/use-integrations";
import { ProviderPicker } from "@/components/shared/provider-picker";
import { getProviderDisplayName } from "@/lib/provider-info";
import { getErrorMessage } from "@/lib/api-client";
import {
  PROVIDER_CREDENTIAL_FIELDS,
  PROVIDER_SETTINGS_FIELDS,
} from "@/lib/constants";
import type { CredentialField } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import { FormActions, FormSection } from "@/components/shared/form-layout";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

type Step = "select" | "configure";

export default function NewCarrierPage() {
  const router = useRouter();
  const t = useTranslations("carriers");
  const tc = useTranslations("common");
  const createIntegration = useCreateIntegration();

  const [step, setStep] = useState<Step>("select");
  const [selectedProvider, setSelectedProvider] = useState<string>("");
  const [credentialValues, setCredentialValues] = useState<Record<string, string | boolean>>({});
  const [credentialErrors, setCredentialErrors] = useState<Record<string, string>>({});
  const [visiblePasswords, setVisiblePasswords] = useState<Record<string, boolean>>({});

  const fields = selectedProvider ? (PROVIDER_CREDENTIAL_FIELDS[selectedProvider] ?? []) : [];

  const handleProviderSelect = (providerKey: string) => {
    setSelectedProvider(providerKey);
    setCredentialValues({});
    setCredentialErrors({});
    setVisiblePasswords({});
    setStep("configure");
  };

  const handleBack = () => {
    setStep("select");
    setSelectedProvider("");
    setCredentialValues({});
    setCredentialErrors({});
  };

  const handleCredentialChange = useCallback((key: string, value: string | boolean) => {
    setCredentialValues((prev) => ({ ...prev, [key]: value }));
    setCredentialErrors((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  }, []);

  const togglePasswordVisibility = useCallback((key: string) => {
    setVisiblePasswords((prev) => ({ ...prev, [key]: !prev[key] }));
  }, []);

  const validateCredentials = (): boolean => {
    const newErrors: Record<string, string> = {};
    for (const field of fields) {
      if (field.required && field.type !== "checkbox") {
        const val = credentialValues[field.key];
        if (!val || (typeof val === "string" && val.trim() === "")) {
          newErrors[field.key] = `Pole "${field.labelKey}" jest wymagane`;
        }
      }
      if (field.type === "url" && credentialValues[field.key]) {
        const urlVal = credentialValues[field.key];
        if (typeof urlVal === "string" && urlVal.trim() !== "") {
          try {
            new URL(urlVal);
          } catch {
            newErrors[field.key] = "Podaj prawidłowy adres URL";
          }
        }
      }
    }
    setCredentialErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateCredentials()) return;

    const credentials: Record<string, unknown> = {};
    for (const field of fields) {
      const val = credentialValues[field.key];
      if (field.type === "checkbox") {
        credentials[field.key] = val === true;
      } else if (typeof val === "string" && val.trim() !== "") {
        credentials[field.key] = val;
      }
    }

    // Extract fields that should go to settings instead of credentials
    const settingsFields = PROVIDER_SETTINGS_FIELDS[selectedProvider] ?? [];
    const settings: Record<string, unknown> = {};
    for (const key of settingsFields) {
      if (credentials[key] !== undefined) {
        settings[key] = credentials[key];
        delete credentials[key];
      }
    }

    createIntegration.mutate(
      {
        provider: selectedProvider,
        credentials,
        ...(Object.keys(settings).length > 0 ? { settings } : {}),
      },
      {
        onSuccess: () => {
          toast.success(t("carrierAdded"));
          router.push("/carriers");
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  const renderField = (field: CredentialField) => {
    if (field.type === "checkbox") {
      return (
        <div key={field.key} className="flex items-center space-x-2">
          <Checkbox
            id={`cred-${field.key}`}
            checked={credentialValues[field.key] === true}
            onCheckedChange={(checked) =>
              handleCredentialChange(field.key, checked === true)
            }
          />
          <Label htmlFor={`cred-${field.key}`} className="font-normal cursor-pointer">
            {field.labelKey}
          </Label>
        </div>
      );
    }

    if (field.type === "select" && field.options) {
      return (
        <div key={field.key} className="space-y-2">
          <Label htmlFor={`cred-${field.key}`}>{field.labelKey}</Label>
          <Select
            value={(credentialValues[field.key] as string) ?? ""}
            onValueChange={(v) => handleCredentialChange(field.key, v)}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Nie ustawiono" />
            </SelectTrigger>
            <SelectContent>
              {field.options.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.labelKey}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {field.helpTextKey && (
            <p className="text-sm text-muted-foreground">{field.helpTextKey}</p>
          )}
        </div>
      );
    }

    const isPassword = field.type === "password";
    const isVisible = visiblePasswords[field.key];
    const inputType = isPassword ? (isVisible ? "text" : "password") : field.type;

    return (
      <div key={field.key} className="space-y-2">
        <Label htmlFor={`cred-${field.key}`}>
          {field.labelKey}
          {field.required && <span className="text-destructive ml-1">*</span>}
        </Label>
        <div className="relative">
          <Input
            id={`cred-${field.key}`}
            type={inputType}
            placeholder={field.placeholderKey}
            value={(credentialValues[field.key] as string) ?? ""}
            onChange={(e) => handleCredentialChange(field.key, e.target.value)}
            className={isPassword ? "pr-10" : ""}
            aria-invalid={!!credentialErrors[field.key]}
          />
          {isPassword && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
              onClick={() => togglePasswordVisibility(field.key)}
              tabIndex={-1}
            >
              {isVisible ? (
                <EyeOff className="h-4 w-4 text-muted-foreground" />
              ) : (
                <Eye className="h-4 w-4 text-muted-foreground" />
              )}
              <span className="sr-only">
                {isVisible ? "Ukryj" : "Pokaż"} {field.labelKey.toLowerCase()}
              </span>
            </Button>
          )}
        </div>
        {field.helpTextKey && (
          <p className="text-sm text-muted-foreground">{field.helpTextKey}</p>
        )}
        {credentialErrors[field.key] && (
          <p className="text-sm text-destructive">{credentialErrors[field.key]}</p>
        )}
      </div>
    );
  };

  const regularFields = fields.filter((f) => f.type !== "checkbox" && f.type !== "select");
  const selectFields = fields.filter((f) => f.type === "select");
  const checkboxFields = fields.filter((f) => f.type === "checkbox");

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={step === "configure" ? "#" : "/carriers"} onClick={step === "configure" ? (e) => { e.preventDefault(); handleBack(); } : undefined}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">
              {step === "select" ? "Wybierz kuriera" : `Konfiguracja ${getProviderDisplayName(selectedProvider)}`}
            </h1>
            <p className="text-muted-foreground">
              {step === "select"
                ? "Wybierz firmę kurierską, z którą chcesz się połączyć"
                : "Wprowadź dane uwierzytelniające"}
            </p>
          </div>
        </div>

        {step === "select" && (
          <ProviderPicker
            category="carrier"
            selected={selectedProvider}
            onSelect={handleProviderSelect}
          />
        )}

        {step === "configure" && (
          <FormSection
            title={`Dane uwierzytelniające — ${getProviderDisplayName(selectedProvider)}`}
            description="Wpisz tylko dane wymagane do aktywnego połączenia z kurierem."
            className="max-w-2xl"
          >
            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="space-y-4">
                {regularFields.map(renderField)}
                {selectFields.map(renderField)}

                {checkboxFields.length > 0 && (
                  <div className="border-t pt-4">
                    {checkboxFields.map(renderField)}
                  </div>
                )}
              </div>

              <FormActions
                secondary={
                  <Button type="button" variant="outline" onClick={handleBack}>
                    Wstecz
                  </Button>
                }
                primary={
                  <Button type="submit" disabled={createIntegration.isPending}>
                    {createIntegration.isPending ? tc("creating") : t("addCarrier")}
                  </Button>
                }
              />
            </form>
          </FormSection>
        )}
      </div>
    </AdminGuard>
  );
}
