import type {
  ProviderField,
  ProviderFieldGroup,
  ProviderFieldGroupKey,
} from "@/types/platform";

/**
 * Field-key validation errors, surfaced as i18n keys so callers can translate.
 * Each code maps to a `schemaBuilder.validation.<code>` message.
 */
export type FieldValidationCode =
  | "keyRequired"
  | "keyDuplicate"
  | "labelRequired"
  | "enumValuesRequired"
  | "regexInvalid"
  | "minGreaterThanMax"
  | "minLengthGreaterThanMaxLength"
  | "secretRecommended";

export interface FieldValidationIssue {
  code: FieldValidationCode;
  /** When true this is advisory (e.g. a "should be secret" hint), not blocking. */
  warning?: boolean;
}

/**
 * Field-name fragments that strongly suggest a value is sensitive. Used to warn
 * when `secret` is off (design spec §291: "switch with warning when disabled for
 * sensitive names").
 */
const SENSITIVE_HINTS = [
  "password",
  "secret",
  "token",
  "api_key",
  "apikey",
  "client_secret",
  "private_key",
  "credential",
];

/** Slugify a label into a stable, lower-snake field key (design spec §288). */
export function slugifyKey(input: string): string {
  return input
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .replace(/_{2,}/g, "_");
}

/** True when a field key/label suggests a sensitive value. */
export function looksSensitive(field: Pick<ProviderField, "key" | "label">): boolean {
  const haystack = `${field.key} ${field.label}`.toLowerCase();
  return SENSITIVE_HINTS.some((hint) => haystack.includes(hint));
}

/** True when `pattern` compiles to a valid regular expression. */
function isValidRegex(pattern: string): boolean {
  try {
    void new RegExp(pattern);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validate a single field against the same rules the backend enforces
 * (model.ValidateFieldSchema), plus the advisory secret-flag hint. `seenKeys`
 * carries keys already used elsewhere in the schema for duplicate detection.
 */
export function validateField(
  field: ProviderField,
  seenKeys: Set<string>,
): FieldValidationIssue[] {
  const issues: FieldValidationIssue[] = [];

  if (!field.key.trim()) {
    issues.push({ code: "keyRequired" });
  } else if (seenKeys.has(field.key)) {
    issues.push({ code: "keyDuplicate" });
  }

  if (!field.label.trim()) {
    issues.push({ code: "labelRequired" });
  }

  if (field.type === "enum" && (field.validation.enum?.length ?? 0) === 0) {
    issues.push({ code: "enumValuesRequired" });
  }

  if (field.validation.regex && !isValidRegex(field.validation.regex)) {
    issues.push({ code: "regexInvalid" });
  }

  const { min, max, min_length, max_length } = field.validation;
  if (min != null && max != null && min > max) {
    issues.push({ code: "minGreaterThanMax" });
  }
  if (min_length != null && max_length != null && min_length > max_length) {
    issues.push({ code: "minLengthGreaterThanMaxLength" });
  }

  if (!field.secret && looksSensitive(field)) {
    issues.push({ code: "secretRecommended", warning: true });
  }

  return issues;
}

/** Validate a whole schema, keyed by field key. Returns only fields with issues. */
export function validateSchema(
  groups: ProviderFieldGroup[],
): Map<string, FieldValidationIssue[]> {
  const result = new Map<string, FieldValidationIssue[]>();
  const seen = new Set<string>();
  for (const group of groups) {
    for (const field of group.fields) {
      const issues = validateField(field, seen);
      const id = field.key || `${group.key}:${field.label}`;
      if (issues.length > 0) {
        result.set(id, issues);
      }
      if (field.key) {
        seen.add(field.key);
      }
    }
  }
  return result;
}

/** A single field as it appears in the generated tenant setup preview. */
export interface PreviewField {
  key: string;
  label: string;
  type: ProviderField["type"];
  required: boolean;
  secret: boolean;
  helpText: string;
  /** Empty string when the field applies to any environment. */
  environmentScope: string;
  enumValues: string[];
}

export interface PreviewGroup {
  key: ProviderFieldGroupKey;
  label: string;
  fields: PreviewField[];
}

/**
 * Build the tenant-facing setup-form preview from a schema. The preview is what
 * a customer would see: customer-visible labels and help text only — never raw
 * secret values, and never the internal `capability_enabled` / adapter bindings.
 * Fields scoped to a specific environment are filtered when `environment` is set.
 */
export function buildTenantSetupPreview(
  groups: ProviderFieldGroup[],
  environment: "all" | "production" | "sandbox" = "all",
): PreviewGroup[] {
  const preview: PreviewGroup[] = [];
  for (const group of groups) {
    const fields: PreviewField[] = [];
    for (const field of group.fields) {
      if (!field.key.trim() || !field.label.trim()) {
        continue; // incomplete drafts are excluded from the customer preview
      }
      const scope = field.environment_scope ?? "";
      if (
        environment !== "all" &&
        scope &&
        scope !== "all" &&
        scope !== environment
      ) {
        continue;
      }
      fields.push({
        key: field.key,
        label: field.label,
        type: field.type,
        required: field.required,
        secret: field.secret,
        helpText: field.help_text ?? "",
        environmentScope: scope,
        enumValues: field.validation.enum ?? [],
      });
    }
    if (fields.length > 0) {
      preview.push({ key: group.key, label: group.label, fields });
    }
  }
  return preview;
}

/** A blank field with safe defaults for a newly-added row. */
export function emptyField(): ProviderField {
  return {
    key: "",
    label: "",
    type: "string",
    required: false,
    secret: false,
    environment_scope: "all",
    help_text: "",
    validation: {},
    capability_enabled: "",
    test_connection_dependency: false,
  };
}
