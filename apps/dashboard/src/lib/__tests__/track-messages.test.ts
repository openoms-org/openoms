import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const messagesRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../../messages");

function leafKeys(value: unknown, prefix = ""): string[] {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return prefix ? [prefix] : [];
  }
  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) =>
    leafKeys(child, prefix ? `${prefix}.${key}` : key),
  );
}

describe("track messages", () => {
  it("keeps the same keys in pl and en", () => {
    const pl = JSON.parse(readFileSync(resolve(messagesRoot, "pl/track.json"), "utf8"));
    const en = JSON.parse(readFileSync(resolve(messagesRoot, "en/track.json"), "utf8"));
    expect(leafKeys(pl).sort()).toEqual(leafKeys(en).sort());
  });
});
