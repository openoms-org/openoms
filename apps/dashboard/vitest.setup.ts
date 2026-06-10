import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// Mock next-intl — t() returns translation keys as-is for unit tests, while
// t.has() resolves key EXISTENCE against the real English catalogs so
// missing-key fallbacks (src/lib/i18n-fallback.ts) are exercised truthfully.
vi.mock("next-intl", async () => {
  const { readFileSync, readdirSync } = await import("node:fs");
  const { dirname, join } = await import("node:path");
  const { fileURLToPath } = await import("node:url");

  const messagesDir = join(
    dirname(fileURLToPath(import.meta.url)),
    "messages",
    "en",
  );

  type Tree = Record<string, unknown>;
  const deepMerge = (target: Tree, source: Tree): Tree => {
    const result: Tree = { ...target };
    for (const [key, value] of Object.entries(source)) {
      const existing = result[key];
      result[key] =
        value !== null &&
        typeof value === "object" &&
        !Array.isArray(value) &&
        existing !== null &&
        typeof existing === "object" &&
        !Array.isArray(existing)
          ? deepMerge(existing as Tree, value as Tree)
          : value;
    }
    return result;
  };

  let messages: Tree = {};
  for (const file of readdirSync(messagesDir).filter((f) => f.endsWith(".json"))) {
    messages = deepMerge(
      messages,
      JSON.parse(readFileSync(join(messagesDir, file), "utf8")) as Tree,
    );
  }

  const hasKey = (path: string): boolean => {
    let node: unknown = messages;
    for (const part of path.split(".")) {
      if (node === null || typeof node !== "object") return false;
      node = (node as Tree)[part];
    }
    return node !== undefined;
  };

  return {
    useTranslations: (namespace?: string) => {
      const t = (key: string) => key;
      t.rich = (key: string) => key;
      t.raw = (key: string) => key;
      t.has = (key: string) => hasKey(namespace ? `${namespace}.${key}` : key);
      return t;
    },
    useLocale: () => "en",
    useMessages: () => ({}),
    useNow: () => new Date(),
    useTimeZone: () => "UTC",
    useFormatter: () => ({}),
    NextIntlClientProvider: ({ children }: { children: React.ReactNode }) =>
      children,
  };
});
