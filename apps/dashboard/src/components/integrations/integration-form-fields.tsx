"use client";

import { ChevronDown, ChevronRight, Eye, EyeOff } from "lucide-react";
import { useTranslations } from "next-intl";
import type { FieldError, UseFormRegister } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { FormField } from "@/components/shared/form-wrapper";
import {
  INTEGRATION_PROVIDER_LABELS,
  type CredentialField,
} from "@/lib/constants";
import type { IntegrationFormValues } from "./integration-form.schema";


interface CredentialFieldsProps {
  selectedProvider: string;
  fields: CredentialField[];
  isEditMode: boolean;
  values: Record<string, string | boolean>;
  errors: Record<string, string>;
  visiblePasswords: Record<string, boolean>;
  onCredentialChange: (key: string, value: string | boolean) => void;
  onTogglePassword: (key: string) => void;
}

export function CredentialFields({
  selectedProvider,
  fields,
  isEditMode,
  values,
  errors,
  visiblePasswords,
  onCredentialChange,
  onTogglePassword,
}: CredentialFieldsProps) {
  const t = useTranslations("integrations");
  if (!selectedProvider || fields.length === 0) return null;

  const regularFields = fields.filter((f) => f.type !== "checkbox" && f.type !== "select");
  const selectFields = fields.filter((f) => f.type === "select");
  const checkboxFields = fields.filter((f) => f.type === "checkbox");

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-foreground">
        {isEditMode ? t("updateData") : t("credentialFields")} &mdash;{" "}
        {INTEGRATION_PROVIDER_LABELS[selectedProvider] ?? selectedProvider}
      </h3>
      {isEditMode && (
        <p className="text-sm text-muted-foreground">{t("editModeHint")}</p>
      )}

      {regularFields.map((field) => renderCredentialField({
        field,
        isEditMode,
        values,
        errors,
        visiblePasswords,
        onCredentialChange,
        onTogglePassword,
        t,
      }))}
      {selectFields.map((field) => renderCredentialField({
        field,
        isEditMode,
        values,
        errors,
        visiblePasswords,
        onCredentialChange,
        onTogglePassword,
        t,
      }))}
      {checkboxFields.length > 0 && (
        <div className="border-t pt-4">
          {checkboxFields.map((field) => renderCredentialField({
            field,
            isEditMode,
            values,
            errors,
            visiblePasswords,
            onCredentialChange,
            onTogglePassword,
            t,
          }))}
        </div>
      )}
    </div>
  );
}

interface AdvancedSettingsProps {
  showAdvanced: boolean;
  onToggle: () => void;
  register: UseFormRegister<IntegrationFormValues>;
  error?: FieldError;
}

export function AdvancedSettings({
  showAdvanced,
  onToggle,
  register,
  error,
}: AdvancedSettingsProps) {
  const t = useTranslations("integrations");

  return (
    <div className="space-y-2">
      <button
        type="button"
        className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
        onClick={onToggle}
      >
        {showAdvanced ? (
          <ChevronDown className="h-4 w-4" />
        ) : (
          <ChevronRight className="h-4 w-4" />
        )}
        {t("showAdvanced")}
      </button>
      {showAdvanced && (
        <FormField<IntegrationFormValues>
          name="settings"
          label={t("additionalSettingsJson")}
          error={error}
          errorClassName="text-sm"
          className="pt-2"
        >
          <Textarea
            id="settings"
            placeholder='{"webhook_url": "...", "sync_interval": 3600}'
            className="min-h-24 font-mono text-sm"
            aria-invalid={!!error}
            {...register("settings")}
          />
        </FormField>
      )}
    </div>
  );
}

interface RenderCredentialFieldArgs {
  field: CredentialField;
  isEditMode: boolean;
  values: Record<string, string | boolean>;
  errors: Record<string, string>;
  visiblePasswords: Record<string, boolean>;
  onCredentialChange: (key: string, value: string | boolean) => void;
  onTogglePassword: (key: string) => void;
  t: ReturnType<typeof useTranslations>;
}

function renderCredentialField({
  field,
  isEditMode,
  values,
  errors,
  visiblePasswords,
  onCredentialChange,
  onTogglePassword,
  t,
}: RenderCredentialFieldArgs) {
  if (field.type === "checkbox") {
    return (
      <div key={field.key} className="flex items-center space-x-2">
        <Checkbox
          id={`cred-${field.key}`}
          checked={values[field.key] === true}
          onCheckedChange={(checked) => onCredentialChange(field.key, checked === true)}
        />
        <Label htmlFor={`cred-${field.key}`} className="font-normal cursor-pointer">
          {field.labelKey}
        </Label>
      </div>
    );
  }

  if (field.type === "select" && field.options) {
    return (
      <FormField<IntegrationFormValues>
        key={field.key}
        label={field.labelKey}
        htmlFor={`cred-${field.key}`}
        description={field.helpTextKey}
      >
        <Select
          value={(values[field.key] as string) ?? ""}
          onValueChange={(v) => onCredentialChange(field.key, v)}
        >
          <SelectTrigger id={`cred-${field.key}`} className="w-full">
            <SelectValue placeholder={t("notSet")} />
          </SelectTrigger>
          <SelectContent>
            {field.options.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.labelKey}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormField>
    );
  }

  const isPassword = field.type === "password";
  const isVisible = visiblePasswords[field.key];
  const inputType = isPassword ? (isVisible ? "text" : "password") : field.type;
  const errorMessage = errors[field.key];

  return (
    <FormField<IntegrationFormValues>
      key={field.key}
      label={field.labelKey}
      htmlFor={`cred-${field.key}`}
      required={!isEditMode && field.required}
      description={field.helpTextKey}
      error={errorMessage ? ({ message: errorMessage } as FieldError) : undefined}
      errorClassName="text-sm"
    >
      <div className="relative">
        <Input
          id={`cred-${field.key}`}
          type={inputType}
          placeholder={isEditMode ? t("leaveEmptyToKeep") : field.placeholderKey}
          value={(values[field.key] as string) ?? ""}
          onChange={(e) => onCredentialChange(field.key, e.target.value)}
          className={isPassword ? "pr-10" : ""}
          aria-invalid={!!errorMessage}
        />
        {isPassword && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
            onClick={() => onTogglePassword(field.key)}
            tabIndex={-1}
          >
            {isVisible ? (
              <EyeOff className="h-4 w-4 text-muted-foreground" />
            ) : (
              <Eye className="h-4 w-4 text-muted-foreground" />
            )}
            <span className="sr-only">
              {isVisible ? t("hide") : t("show")} {field.labelKey.toLowerCase()}
            </span>
          </Button>
        )}
      </div>
    </FormField>
  );
}
