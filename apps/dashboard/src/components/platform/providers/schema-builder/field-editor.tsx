"use client";

import { useTranslations } from "next-intl";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ValidationErrors } from "@/components/platform/providers/shared/validation-errors";
import {
  type FieldValidationIssue,
  looksSensitive,
  slugifyKey,
} from "@/components/platform/providers/schema-builder/schema-helpers";
import {
  PROVIDER_FIELD_ENV_SCOPES,
  PROVIDER_FIELD_TYPES,
  type ProviderField,
  type ProviderFieldEnvScope,
  type ProviderFieldType,
} from "@/types/platform";

interface FieldEditorProps {
  field: ProviderField;
  issues: FieldValidationIssue[];
  readOnly: boolean;
  onChange: (field: ProviderField) => void;
}

function numberOrUndefined(value: string): number | undefined {
  if (value.trim() === "") return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

/**
 * Right-pane editor for a single field's properties (design spec Screen 5).
 * Edits are blocked when `readOnly` (published version). Toggling Secret off on
 * a sensitive-looking field surfaces a validation warning (handled by `issues`).
 */
export function FieldEditor({ field, issues, readOnly, onChange }: FieldEditorProps) {
  const t = useTranslations("platform");

  const update = (patch: Partial<ProviderField>) => {
    onChange({ ...field, ...patch });
  };

  const updateValidation = (patch: Partial<ProviderField["validation"]>) => {
    onChange({ ...field, validation: { ...field.validation, ...patch } });
  };

  return (
    <div className="space-y-4">
      <div className="grid gap-2">
        <Label htmlFor="field-label">{t("schemaBuilder.field.label")}</Label>
        <Input
          id="field-label"
          value={field.label}
          disabled={readOnly}
          onChange={(e) => {
            const label = e.target.value;
            // Auto-derive key from label while the key is still untouched/blank.
            const shouldSyncKey =
              !field.key || field.key === slugifyKey(field.label);
            update(
              shouldSyncKey
                ? { label, key: slugifyKey(label) }
                : { label },
            );
          }}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="field-key">{t("schemaBuilder.field.key")}</Label>
        <Input
          id="field-key"
          value={field.key}
          disabled={readOnly}
          className="font-mono"
          onChange={(e) => update({ key: slugifyKey(e.target.value) })}
        />
        <p className="text-xs text-muted-foreground">
          {t("schemaBuilder.field.keyHint")}
        </p>
      </div>

      <div className="grid gap-2">
        <Label htmlFor="field-type">{t("schemaBuilder.field.type")}</Label>
        <Select
          value={field.type}
          disabled={readOnly}
          onValueChange={(value) => update({ type: value as ProviderFieldType })}
        >
          <SelectTrigger id="field-type">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PROVIDER_FIELD_TYPES.map((type) => (
              <SelectItem key={type} value={type}>
                {t(`schemaBuilder.fieldType.${type}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <Label htmlFor="field-env">{t("schemaBuilder.field.environment")}</Label>
        <Select
          value={field.environment_scope || "all"}
          disabled={readOnly}
          onValueChange={(value) =>
            update({ environment_scope: value as ProviderFieldEnvScope })
          }
        >
          <SelectTrigger id="field-env">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PROVIDER_FIELD_ENV_SCOPES.map((scope) => (
              <SelectItem key={scope} value={scope}>
                {t(`schemaBuilder.envScope.${scope}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center justify-between rounded-md border p-3">
        <Label htmlFor="field-required" className="cursor-pointer">
          {t("schemaBuilder.field.required")}
        </Label>
        <Switch
          id="field-required"
          checked={field.required}
          disabled={readOnly}
          onCheckedChange={(checked) => update({ required: checked })}
        />
      </div>

      <div className="rounded-md border p-3">
        <div className="flex items-center justify-between">
          <Label htmlFor="field-secret" className="cursor-pointer">
            {t("schemaBuilder.field.secret")}
          </Label>
          <Switch
            id="field-secret"
            checked={field.secret}
            disabled={readOnly}
            onCheckedChange={(checked) => update({ secret: checked })}
          />
        </div>
        {!field.secret && looksSensitive(field) && (
          <p className="mt-2 text-xs text-warning">
            {t("schemaBuilder.field.secretWarning")}
          </p>
        )}
      </div>

      {field.type === "enum" && (
        <div className="grid gap-2">
          <Label htmlFor="field-enum">{t("schemaBuilder.field.enumValues")}</Label>
          <Input
            id="field-enum"
            value={(field.validation.enum ?? []).join(", ")}
            disabled={readOnly}
            placeholder={t("schemaBuilder.field.enumPlaceholder")}
            onChange={(e) =>
              updateValidation({
                enum: e.target.value
                  .split(",")
                  .map((v) => v.trim())
                  .filter(Boolean),
              })
            }
          />
        </div>
      )}

      {(field.type === "number" || field.type === "string" || field.type === "textarea" || field.type === "url" || field.type === "password") && (
        <div className="grid grid-cols-2 gap-3">
          {field.type === "number" ? (
            <>
              <div className="grid gap-2">
                <Label htmlFor="field-min">{t("schemaBuilder.field.min")}</Label>
                <Input
                  id="field-min"
                  type="number"
                  value={field.validation.min ?? ""}
                  disabled={readOnly}
                  onChange={(e) =>
                    updateValidation({ min: numberOrUndefined(e.target.value) })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="field-max">{t("schemaBuilder.field.max")}</Label>
                <Input
                  id="field-max"
                  type="number"
                  value={field.validation.max ?? ""}
                  disabled={readOnly}
                  onChange={(e) =>
                    updateValidation({ max: numberOrUndefined(e.target.value) })
                  }
                />
              </div>
            </>
          ) : (
            <>
              <div className="grid gap-2">
                <Label htmlFor="field-minlen">
                  {t("schemaBuilder.field.minLength")}
                </Label>
                <Input
                  id="field-minlen"
                  type="number"
                  value={field.validation.min_length ?? ""}
                  disabled={readOnly}
                  onChange={(e) =>
                    updateValidation({
                      min_length: numberOrUndefined(e.target.value),
                    })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="field-maxlen">
                  {t("schemaBuilder.field.maxLength")}
                </Label>
                <Input
                  id="field-maxlen"
                  type="number"
                  value={field.validation.max_length ?? ""}
                  disabled={readOnly}
                  onChange={(e) =>
                    updateValidation({
                      max_length: numberOrUndefined(e.target.value),
                    })
                  }
                />
              </div>
            </>
          )}
        </div>
      )}

      <div className="grid gap-2">
        <Label htmlFor="field-regex">{t("schemaBuilder.field.regex")}</Label>
        <Input
          id="field-regex"
          value={field.validation.regex ?? ""}
          disabled={readOnly}
          className="font-mono"
          onChange={(e) => updateValidation({ regex: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="field-help">{t("schemaBuilder.field.helpText")}</Label>
        <Textarea
          id="field-help"
          value={field.help_text ?? ""}
          disabled={readOnly}
          rows={2}
          onChange={(e) => update({ help_text: e.target.value })}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="field-cap">{t("schemaBuilder.field.capabilityEnabled")}</Label>
        <Input
          id="field-cap"
          value={field.capability_enabled ?? ""}
          disabled={readOnly}
          className="font-mono"
          placeholder={t("schemaBuilder.field.capabilityPlaceholder")}
          onChange={(e) => update({ capability_enabled: e.target.value })}
        />
      </div>

      <div className="flex items-center justify-between rounded-md border p-3">
        <Label htmlFor="field-testdep" className="cursor-pointer">
          {t("schemaBuilder.field.testConnectionDependency")}
        </Label>
        <Switch
          id="field-testdep"
          checked={field.test_connection_dependency ?? false}
          disabled={readOnly}
          onCheckedChange={(checked) =>
            update({ test_connection_dependency: checked })
          }
        />
      </div>

      <ValidationErrors issues={issues} />
    </div>
  );
}
