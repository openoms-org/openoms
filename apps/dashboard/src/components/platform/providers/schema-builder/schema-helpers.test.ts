import { describe, expect, it } from "vitest";
import {
  buildTenantSetupPreview,
  emptyField,
  looksSensitive,
  slugifyKey,
  validateField,
  validateSchema,
} from "@/components/platform/providers/schema-builder/schema-helpers";
import type { ProviderField, ProviderFieldGroup } from "@/types/platform";

function field(overrides: Partial<ProviderField>): ProviderField {
  return { ...emptyField(), ...overrides };
}

describe("slugifyKey", () => {
  it("lowercases and snake-cases a label", () => {
    expect(slugifyKey("API Key")).toBe("api_key");
  });

  it("strips diacritics and collapses separators", () => {
    expect(slugifyKey("  Bazá  URL!! ")).toBe("baza_url");
  });
});

describe("looksSensitive", () => {
  it("flags sensitive-looking keys", () => {
    expect(looksSensitive({ key: "client_secret", label: "Secret" })).toBe(true);
    expect(looksSensitive({ key: "api_token", label: "Token" })).toBe(true);
  });

  it("does not flag ordinary fields", () => {
    expect(looksSensitive({ key: "base_url", label: "Base URL" })).toBe(false);
  });
});

describe("validateField", () => {
  it("requires a key and a label", () => {
    const issues = validateField(field({ key: "", label: "" }), new Set());
    const codes = issues.map((i) => i.code);
    expect(codes).toContain("keyRequired");
    expect(codes).toContain("labelRequired");
  });

  it("detects duplicate keys", () => {
    const seen = new Set(["api_key"]);
    const issues = validateField(
      field({ key: "api_key", label: "API key" }),
      seen,
    );
    expect(issues.map((i) => i.code)).toContain("keyDuplicate");
  });

  it("requires enum values for enum fields", () => {
    const issues = validateField(
      field({ key: "mode", label: "Mode", type: "enum", validation: {} }),
      new Set(),
    );
    expect(issues.map((i) => i.code)).toContain("enumValuesRequired");
  });

  it("rejects an invalid regex", () => {
    const issues = validateField(
      field({ key: "x", label: "X", validation: { regex: "([" } }),
      new Set(),
    );
    expect(issues.map((i) => i.code)).toContain("regexInvalid");
  });

  it("rejects min > max and minLength > maxLength", () => {
    const numeric = validateField(
      field({ key: "n", label: "N", type: "number", validation: { min: 10, max: 1 } }),
      new Set(),
    );
    expect(numeric.map((i) => i.code)).toContain("minGreaterThanMax");

    const text = validateField(
      field({ key: "s", label: "S", validation: { min_length: 9, max_length: 2 } }),
      new Set(),
    );
    expect(text.map((i) => i.code)).toContain("minLengthGreaterThanMaxLength");
  });

  it("warns (advisory) when a sensitive field is not secret", () => {
    const issues = validateField(
      field({ key: "api_secret", label: "API secret", secret: false }),
      new Set(),
    );
    const secretIssue = issues.find((i) => i.code === "secretRecommended");
    expect(secretIssue).toBeDefined();
    expect(secretIssue?.warning).toBe(true);
  });

  it("does not warn when the sensitive field is already secret", () => {
    const issues = validateField(
      field({ key: "api_secret", label: "API secret", secret: true }),
      new Set(),
    );
    expect(issues.map((i) => i.code)).not.toContain("secretRecommended");
  });
});

describe("validateSchema", () => {
  it("flags only fields that have issues, keyed by field key", () => {
    const groups: ProviderFieldGroup[] = [
      {
        key: "secret_credentials",
        label: "Secret",
        fields: [
          field({ key: "api_key", label: "API key", secret: true }),
          field({ key: "api_key", label: "Duplicate", secret: true }),
        ],
      },
    ];
    const result = validateSchema(groups);
    expect(result.has("api_key")).toBe(true);
    expect(result.get("api_key")?.map((i) => i.code)).toContain("keyDuplicate");
  });
});

describe("buildTenantSetupPreview", () => {
  const groups: ProviderFieldGroup[] = [
    {
      key: "secret_credentials",
      label: "Secret credentials",
      fields: [
        field({
          key: "api_key",
          label: "API key",
          type: "password",
          secret: true,
          required: true,
          help_text: "Find it in your account.",
        }),
        // Incomplete draft (no key) — excluded from the customer preview.
        field({ key: "", label: "", secret: false }),
      ],
    },
    {
      key: "environment",
      label: "Environment",
      fields: [
        field({
          key: "prod_url",
          label: "Production URL",
          type: "url",
          environment_scope: "production",
        }),
      ],
    },
  ];

  it("includes complete fields and excludes incomplete drafts", () => {
    const preview = buildTenantSetupPreview(groups);
    const secretGroup = preview.find((g) => g.key === "secret_credentials");
    expect(secretGroup?.fields).toHaveLength(1);
    expect(secretGroup?.fields[0].key).toBe("api_key");
    expect(secretGroup?.fields[0].secret).toBe(true);
    expect(secretGroup?.fields[0].helpText).toBe("Find it in your account.");
  });

  it("filters out production-only fields when previewing sandbox", () => {
    const preview = buildTenantSetupPreview(groups, "sandbox");
    expect(preview.find((g) => g.key === "environment")).toBeUndefined();
  });

  it("keeps production-only fields when previewing production", () => {
    const preview = buildTenantSetupPreview(groups, "production");
    expect(preview.find((g) => g.key === "environment")?.fields[0].key).toBe(
      "prod_url",
    );
  });
});
