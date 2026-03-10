import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// Mock next-intl — returns translation keys as-is for unit tests
vi.mock("next-intl", () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.rich = (key: string) => key;
    t.raw = (key: string) => key;
    t.has = () => true;
    return t;
  },
  useLocale: () => "en",
  useMessages: () => ({}),
  useNow: () => new Date(),
  useTimeZone: () => "UTC",
  useFormatter: () => ({}),
  NextIntlClientProvider: ({ children }: { children: React.ReactNode }) => children,
}));
