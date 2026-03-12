import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";
import { readdir, readFile } from "fs/promises";
import { join } from "path";

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

async function loadMessages(locale: SupportedLocale): Promise<Record<string, unknown>> {
  const messagesDir = join(process.cwd(), "messages", locale);
  const files = (await readdir(messagesDir))
    .filter((f) => f.endsWith(".json"))
    .sort();
  const chunks = await Promise.all(
    files.map(async (f) => {
      const content = await readFile(join(messagesDir, f), "utf-8");
      return JSON.parse(content) as Record<string, unknown>;
    })
  );
  return chunks.reduce<Record<string, unknown>>(
    (acc, chunk) => deepMerge(acc, chunk),
    {}
  );
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
