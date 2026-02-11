import { describe, it, expect } from "vitest";
import { cn, formatDate, formatCurrency, shortId } from "@/lib/utils";

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("handles conditional classes", () => {
    expect(cn("base", false && "hidden", "visible")).toBe("base visible");
  });

  it("merges tailwind classes correctly (last wins)", () => {
    const result = cn("p-4", "p-2");
    expect(result).toBe("p-2");
  });

  it("handles empty input", () => {
    expect(cn()).toBe("");
  });

  it("handles undefined and null", () => {
    expect(cn("foo", undefined, null, "bar")).toBe("foo bar");
  });
});

describe("formatDate", () => {
  it("formats a date string correctly", () => {
    // format is dd.MM.yyyy HH:mm
    const result = formatDate("2025-01-15T10:30:00Z");
    // The exact output depends on timezone, but should contain the date parts
    expect(result).toMatch(/\d{2}\.\d{2}\.\d{4} \d{2}:\d{2}/);
  });

  it("formats a Date object", () => {
    const date = new Date(2025, 0, 15, 10, 30);
    const result = formatDate(date);
    expect(result).toMatch(/15\.01\.2025 10:30/);
  });

  it("handles ISO date strings", () => {
    const result = formatDate("2025-06-01T14:00:00.000Z");
    expect(result).toMatch(/\d{2}\.\d{2}\.\d{4}/);
  });
});

describe("formatCurrency", () => {
  it("formats a positive amount", () => {
    const result = formatCurrency(199.99);
    // Polish locale should use comma as decimal separator
    expect(result).toContain("199");
    expect(result).toContain("99");
  });

  it("formats zero", () => {
    const result = formatCurrency(0);
    expect(result).toContain("0");
  });

  it("handles null amount", () => {
    expect(formatCurrency(null)).toBe("0,00 zł");
  });

  it("handles undefined amount", () => {
    expect(formatCurrency(undefined)).toBe("0,00 zł");
  });

  it("handles NaN", () => {
    expect(formatCurrency(NaN)).toBe("0,00 zł");
  });

  it("formats with custom currency", () => {
    const result = formatCurrency(100, "EUR");
    expect(result).toContain("100");
  });

  it("formats large amounts", () => {
    const result = formatCurrency(10000.50);
    expect(result).toContain("10");
    expect(result).toContain("50");
  });
});

describe("shortId", () => {
  it("returns first 8 characters of a UUID", () => {
    const uuid = "550e8400-e29b-41d4-a716-446655440000";
    expect(shortId(uuid)).toBe("550e8400");
  });

  it("returns dash for empty string", () => {
    expect(shortId("")).toBe("\u2014");
  });

  it("handles short strings (less than 8 chars)", () => {
    expect(shortId("abc")).toBe("abc");
  });

  it("returns exactly 8 characters for longer strings", () => {
    expect(shortId("abcdefghijklmnop")).toBe("abcdefgh");
  });
});
