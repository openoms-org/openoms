"use client";

import { useTranslations } from "next-intl";
import type { FieldError } from "react-hook-form";
import { FormField } from "@/components/shared/form-wrapper";
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
  PROVIDERS_WITH_DEDICATED_PAGES,
} from "@/lib/constants";
import { isInDevelopment } from "@/lib/integration-status";
import { getVisibleProviderKeys } from "@/lib/readiness";
import type { IntegrationFormValues } from "./integration-form.schema";

interface ProviderSelectProps {
  value: string;
  error?: FieldError;
  onChange: (value: string) => void;
}

export function ProviderSelect({ value, error, onChange }: ProviderSelectProps) {
  const t = useTranslations("integrations");
  const categoryEntries = Object.entries(PROVIDER_CATEGORIES).map(([key, cat]) => [
    key,
    {
      ...cat,
      providers: getVisibleProviderKeys(
        cat.providers.filter((provider) => !(provider in PROVIDERS_WITH_DEDICATED_PAGES)),
      ),
    },
  ] as [string, { labelKey: string; providers: string[] }]).filter(([, cat]) => cat.providers.length > 0);

  return (
    <FormField<IntegrationFormValues>
      name="provider"
      label={t("provider")}
      error={error}
      errorClassName="text-sm"
    >
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className="w-full" aria-invalid={!!error}>
          <SelectValue placeholder={t("selectProviderPlaceholder")} />
        </SelectTrigger>
        <SelectContent>
          {categoryEntries.map(([catKey, category], catIndex) => (
            <SelectGroup key={catKey}>
              {catIndex > 0 && <SelectSeparator />}
              <SelectLabel>{t(category.labelKey)}</SelectLabel>
              {category.providers.map((provider) => (
                <SelectItem key={provider} value={provider}>
                  {INTEGRATION_PROVIDER_LABELS[provider] ??
                    provider.charAt(0).toUpperCase() + provider.slice(1)}
                  {isInDevelopment(provider) ? ` (${t("inDevelopment")})` : ""}
                </SelectItem>
              ))}
            </SelectGroup>
          ))}
        </SelectContent>
      </Select>
    </FormField>
  );
}
