"use client";

import { useState, useCallback } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeOff, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  INTEGRATION_PROVIDER_LABELS,
  PROVIDER_CATEGORIES,
  PROVIDER_CREDENTIAL_FIELDS,
  PROVIDER_SETTINGS_FIELDS,
} from "@/lib/constants";
import type { CredentialField } from "@/lib/constants";

const integrationSchema = z.object({
  provider: z.string().min(1, "Dostawca jest wymagany"),
  settings: z.string().optional().refine(
    (val) => {
      if (!val || val.trim() === "") return true;
      try {
        JSON.parse(val);
        return true;
      } catch {
        return false;
      }
    },
    { message: "Nieprawidłowy format JSON" }
  ),
});

type IntegrationFormValues = z.infer<typeof integrationSchema>;

interface IntegrationFormProps {
  onSubmit: (data: {
    provider: string;
    credentials: Record<string, unknown>;
    settings?: Record<string, unknown>;
  }) => void;
  isLoading?: boolean;
}

export function IntegrationForm({ onSubmit, isLoading = false }: IntegrationFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<IntegrationFormValues>({
    resolver: zodResolver(integrationSchema),
    defaultValues: {
      provider: "",
      settings: "",
    },
  });

  const selectedProvider = watch("provider");
  const fields = selectedProvider ? (PROVIDER_CREDENTIAL_FIELDS[selectedProvider] ?? []) : [];

  const [credentialValues, setCredentialValues] = useState<Record<string, string | boolean>>({});
  const [credentialErrors, setCredentialErrors] = useState<Record<string, string>>({});
  const [visiblePasswords, setVisiblePasswords] = useState<Record<string, boolean>>({});
  const [showAdvanced, setShowAdvanced] = useState(false);

  const handleProviderChange = useCallback((value: string) => {
    setValue("provider", value, { shouldValidate: true });
    setCredentialValues({});
    setCredentialErrors({});
    setVisiblePasswords({});
  }, [setValue]);

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
          newErrors[field.key] = `Pole "${field.label}" jest wymagane`;
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

  const handleFormSubmit = (data: IntegrationFormValues) => {
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
    const settingsFields = PROVIDER_SETTINGS_FIELDS[data.provider] ?? [];
    const autoSettings: Record<string, unknown> = {};
    for (const key of settingsFields) {
      if (credentials[key] !== undefined) {
        autoSettings[key] = credentials[key];
        delete credentials[key];
      }
    }

    // Merge with manual JSON settings if provided
    let finalSettings: Record<string, unknown> | undefined;
    if (data.settings && data.settings.trim() !== "") {
      finalSettings = { ...JSON.parse(data.settings), ...autoSettings };
    } else if (Object.keys(autoSettings).length > 0) {
      finalSettings = autoSettings;
    }

    const result: {
      provider: string;
      credentials: Record<string, unknown>;
      settings?: Record<string, unknown>;
    } = {
      provider: data.provider,
      credentials,
    };

    result.settings = finalSettings;

    onSubmit(result);
  };

  const regularFields = fields.filter((f) => f.type !== "checkbox");
  const checkboxFields = fields.filter((f) => f.type === "checkbox");
  const categoryEntries = Object.entries(PROVIDER_CATEGORIES);

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
            {field.label}
          </Label>
        </div>
      );
    }

    const isPassword = field.type === "password";
    const isVisible = visiblePasswords[field.key];
    const inputType = isPassword ? (isVisible ? "text" : "password") : field.type;

    return (
      <div key={field.key} className="space-y-2">
        <Label htmlFor={`cred-${field.key}`}>
          {field.label}
          {field.required && <span className="text-destructive ml-1">*</span>}
        </Label>
        <div className="relative">
          <Input
            id={`cred-${field.key}`}
            type={inputType}
            placeholder={field.placeholder}
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
                {isVisible ? "Ukryj" : "Pokaż"} {field.label.toLowerCase()}
              </span>
            </Button>
          )}
        </div>
        {field.helpText && (
          <p className="text-sm text-muted-foreground">{field.helpText}</p>
        )}
        {credentialErrors[field.key] && (
          <p className="text-sm text-destructive">{credentialErrors[field.key]}</p>
        )}
      </div>
    );
  };

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      {/* Provider selection grouped by category */}
      <div className="space-y-2">
        <Label htmlFor="provider">Dostawca</Label>
        <Select
          value={selectedProvider}
          onValueChange={handleProviderChange}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Wybierz dostawcę" />
          </SelectTrigger>
          <SelectContent>
            {categoryEntries.map(([catKey, category], catIndex) => (
              <SelectGroup key={catKey}>
                {catIndex > 0 && <SelectSeparator />}
                <SelectLabel>{category.label}</SelectLabel>
                {category.providers.map((provider) => (
                  <SelectItem key={provider} value={provider}>
                    {INTEGRATION_PROVIDER_LABELS[provider] ??
                      provider.charAt(0).toUpperCase() + provider.slice(1)}
                  </SelectItem>
                ))}
              </SelectGroup>
            ))}
          </SelectContent>
        </Select>
        {errors.provider && (
          <p className="text-sm text-destructive">{errors.provider.message}</p>
        )}
      </div>

      {/* Dynamic credential fields */}
      {selectedProvider && fields.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-sm font-medium text-foreground">
            Dane uwierzytelniające &mdash;{" "}
            {INTEGRATION_PROVIDER_LABELS[selectedProvider] ?? selectedProvider}
          </h3>

          {/* Regular (text/password/url) fields */}
          {regularFields.map(renderField)}

          {/* Checkbox fields separated visually */}
          {checkboxFields.length > 0 && (
            <>
              <div className="border-t pt-4">
                {checkboxFields.map(renderField)}
              </div>
            </>
          )}
        </div>
      )}

      {/* Collapsible advanced settings */}
      <div className="space-y-2">
        <button
          type="button"
          className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          {showAdvanced ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          Pokaż zaawansowane
        </button>
        {showAdvanced && (
          <div className="space-y-2 pt-2">
            <Label htmlFor="settings">Ustawienia dodatkowe (JSON, opcjonalne)</Label>
            <Textarea
              id="settings"
              placeholder='{"webhook_url": "...", "sync_interval": 3600}'
              className="min-h-24 font-mono text-sm"
              {...register("settings")}
            />
            {errors.settings && (
              <p className="text-sm text-destructive">{errors.settings.message}</p>
            )}
          </div>
        )}
      </div>

      <div className="flex justify-end gap-3">
        <Button type="submit" disabled={isLoading}>
          {isLoading ? "Tworzenie..." : "Utwórz integrację"}
        </Button>
      </div>
    </form>
  );
}
