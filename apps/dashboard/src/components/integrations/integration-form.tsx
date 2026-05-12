"use client";

import { useCallback, useState } from "react";
import { useTranslations } from "next-intl";
import { FormWrapper } from "@/components/shared/form-wrapper";
import {
  PROVIDER_CREDENTIAL_FIELDS,
  PROVIDER_SETTINGS_FIELDS,
} from "@/lib/constants";
import {
  AdvancedSettings,
  CredentialFields,
} from "./integration-form-fields";
import { ProviderSelect } from "./integration-provider-select";
import {
  integrationSchema,
  type IntegrationFormValues,
} from "./integration-form.schema";

interface IntegrationFormProps {
  onSubmit: (data: {
    provider: string;
    credentials: Record<string, unknown>;
    settings?: Record<string, unknown>;
  }) => void;
  isLoading?: boolean;
  /** Edit mode: provider is fixed and fields pre-filled from existing settings */
  editProvider?: string;
  /** Existing settings to pre-fill settings fields (e.g. geowidget_token) */
  existingSettings?: Record<string, unknown>;
}

export function IntegrationForm({
  editProvider,
  existingSettings,
  ...props
}: IntegrationFormProps) {
  const formKey = `${editProvider ?? "new"}:${JSON.stringify(existingSettings ?? {})}`;

  return (
    <IntegrationFormContent
      key={formKey}
      editProvider={editProvider}
      existingSettings={existingSettings}
      {...props}
    />
  );
}

function IntegrationFormContent({
  onSubmit,
  isLoading = false,
  editProvider,
  existingSettings,
}: IntegrationFormProps) {
  const t = useTranslations("integrations");
  const isEditMode = !!editProvider;
  const [credentialValues, setCredentialValues] = useState<Record<string, string | boolean>>(() =>
    buildInitialValues(editProvider, existingSettings)
  );
  const [credentialErrors, setCredentialErrors] = useState<Record<string, string>>({});
  const [visiblePasswords, setVisiblePasswords] = useState<Record<string, boolean>>({});
  const [showAdvanced, setShowAdvanced] = useState(false);

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

  const validateCredentials = (provider: string): boolean => {
    const fields = PROVIDER_CREDENTIAL_FIELDS[provider] ?? [];
    const newErrors: Record<string, string> = {};
    for (const field of fields) {
      // In edit mode, required fields are optional (user may only update some)
      if (!isEditMode && field.required && field.type !== "checkbox") {
        const val = credentialValues[field.key];
        if (!val || (typeof val === "string" && val.trim() === "")) {
          newErrors[field.key] = t("fieldRequired", { field: field.labelKey });
        }
      }
      if (field.type === "url" && credentialValues[field.key]) {
        const urlVal = credentialValues[field.key];
        if (typeof urlVal === "string" && urlVal.trim() !== "") {
          try {
            new URL(urlVal);
          } catch {
            newErrors[field.key] = t("invalidUrl");
          }
        }
      }
    }
    setCredentialErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleFormSubmit = (data: IntegrationFormValues) => {
    const provider = editProvider ?? data.provider;
    if (!validateCredentials(provider)) return;

    const fields = PROVIDER_CREDENTIAL_FIELDS[provider] ?? [];
    const credentials: Record<string, unknown> = {};
    for (const field of fields) {
      const val = credentialValues[field.key];
      if (field.type === "checkbox") {
        credentials[field.key] = val === true;
      } else if (typeof val === "string" && val.trim() !== "") {
        credentials[field.key] = val;
      }
    }

    const settingsFields = PROVIDER_SETTINGS_FIELDS[provider] ?? [];
    const autoSettings: Record<string, unknown> = {};
    for (const key of settingsFields) {
      if (credentials[key] !== undefined) {
        autoSettings[key] = credentials[key];
        delete credentials[key];
      }
    }

    let finalSettings: Record<string, unknown> | undefined;
    if (data.settings && data.settings.trim() !== "") {
      finalSettings = { ...JSON.parse(data.settings), ...autoSettings };
    } else if (Object.keys(autoSettings).length > 0) {
      finalSettings = autoSettings;
    }

    if (isEditMode && existingSettings) {
      finalSettings = { ...existingSettings, ...(finalSettings ?? {}) };
    }

    onSubmit({ provider, credentials, settings: finalSettings });
  };

  return (
    <FormWrapper<IntegrationFormValues>
      schema={integrationSchema}
      defaultValues={{ provider: editProvider ?? "", settings: "" }}
      onSubmit={handleFormSubmit}
      className="space-y-6"
      actionsClassName="justify-end"
      submitLabel={isEditMode ? t("saveChanges") : t("create")}
      submittingLabel={isEditMode ? t("updating") : t("creating")}
      isSubmitting={isLoading}
      showErrorSummary={false}
    >
      {({ register, setValue, watch, formState: { errors } }) => {
        const selectedProvider = editProvider ?? watch("provider");
        const fields = selectedProvider ? (PROVIDER_CREDENTIAL_FIELDS[selectedProvider] ?? []) : [];

        return (
          <>
            {!isEditMode && (
              <ProviderSelect
                value={selectedProvider}
                error={errors.provider}
                onChange={(value) => {
                  setValue("provider", value, { shouldValidate: true });
                  setCredentialValues({});
                  setCredentialErrors({});
                  setVisiblePasswords({});
                }}
              />
            )}

            <CredentialFields
              selectedProvider={selectedProvider}
              fields={fields}
              isEditMode={isEditMode}
              values={credentialValues}
              errors={credentialErrors}
              visiblePasswords={visiblePasswords}
              onCredentialChange={handleCredentialChange}
              onTogglePassword={togglePasswordVisibility}
            />

            <AdvancedSettings
              showAdvanced={showAdvanced}
              onToggle={() => setShowAdvanced((current) => !current)}
              register={register}
              error={errors.settings}
            />
          </>
        );
      }}
    </FormWrapper>
  );
}

function buildInitialValues(
  editProvider: string | undefined,
  existingSettings: Record<string, unknown> | undefined
): Record<string, string | boolean> {
  if (!editProvider || !existingSettings) return {};

  const settingsFields = PROVIDER_SETTINGS_FIELDS[editProvider] ?? [];
  const initial: Record<string, string | boolean> = {};
  for (const key of settingsFields) {
    const val = existingSettings[key];
    if (typeof val === "string") initial[key] = val;
    if (typeof val === "boolean") initial[key] = val;
  }
  return initial;
}
