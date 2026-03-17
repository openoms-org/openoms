import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";

const SUPPORTED_LOCALES = ["en", "pl"] as const;
const DEFAULT_LOCALE = "en";

type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

function isSupportedLocale(locale: string): locale is SupportedLocale {
  return (SUPPORTED_LOCALES as readonly string[]).includes(locale);
}

function detectLocaleFromHeader(acceptLanguage: string | null): SupportedLocale {
  if (!acceptLanguage) return DEFAULT_LOCALE;
  const preferred = acceptLanguage
    .split(",")
    .map((l) => l.split(";")[0].trim().split("-")[0]);
  const match = preferred.find(isSupportedLocale);
  return match ?? DEFAULT_LOCALE;
}

function deepMerge(
  target: Record<string, unknown>,
  source: Record<string, unknown>
): Record<string, unknown> {
  const result: Record<string, unknown> = { ...target };
  for (const key of Object.keys(source)) {
    const targetVal = result[key];
    const sourceVal = source[key];
    if (
      sourceVal !== null &&
      typeof sourceVal === "object" &&
      !Array.isArray(sourceVal) &&
      targetVal !== null &&
      typeof targetVal === "object" &&
      !Array.isArray(targetVal)
    ) {
      result[key] = deepMerge(
        targetVal as Record<string, unknown>,
        sourceVal as Record<string, unknown>
      );
    } else {
      result[key] = sourceVal;
    }
  }
  return result;
}

const MODULES = [
  "automation",
  "common",
  "customers",
  "dashboard",
  "integrations",
  "invoices",
  "layout",
  "listings",
  "marketplaces",
  "misc",
  "operations",
  "orders",
  "products",
  "reconciliation",
  "reports",
  "returns",
  "settings",
  "shipments",
  "statuses",
  "suppliers",
  "warehouses",
  "workflows",
] as const;

function convertDotKeys(
  obj: Record<string, unknown>
): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    const converted =
      value !== null && typeof value === "object" && !Array.isArray(value)
        ? convertDotKeys(value as Record<string, unknown>)
        : value;

    if (key.includes(".")) {
      const parts = key.split(".");
      let current = result;
      for (let i = 0; i < parts.length - 1; i++) {
        if (!(parts[i] in current) || typeof current[parts[i]] !== "object") {
          current[parts[i]] = {};
        }
        current = current[parts[i]] as Record<string, unknown>;
      }
      const last = parts[parts.length - 1];
      if (
        typeof current[last] === "object" &&
        typeof converted === "object" &&
        converted !== null
      ) {
        current[last] = deepMerge(
          current[last] as Record<string, unknown>,
          converted as Record<string, unknown>
        );
      } else {
        current[last] = converted;
      }
    } else if (
      typeof result[key] === "object" &&
      typeof converted === "object" &&
      converted !== null &&
      !Array.isArray(converted)
    ) {
      result[key] = deepMerge(
        result[key] as Record<string, unknown>,
        converted as Record<string, unknown>
      );
    } else {
      result[key] = converted;
    }
  }
  return result;
}

async function loadMessages(
  locale: SupportedLocale
): Promise<Record<string, unknown>> {
  const chunks = await Promise.all(
    MODULES.map((mod) => import(`../../messages/${locale}/${mod}.json`))
  );
  const merged = chunks.reduce<Record<string, unknown>>(
    (acc, chunk) => deepMerge(acc, chunk.default ?? chunk),
    {}
  );
  return convertDotKeys(merged);
}

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const headerStore = await headers();

  const cookieLocale = cookieStore.get("NEXT_LOCALE")?.value;
  const locale =
    cookieLocale && isSupportedLocale(cookieLocale)
      ? cookieLocale
      : detectLocaleFromHeader(headerStore.get("accept-language"));

  return {
    locale,
    messages: await loadMessages(locale),
  };
});
