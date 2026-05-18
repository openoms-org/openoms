import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const registrationPages = [
  "src/app/(auth)/register/page.tsx",
  "src/app/(auth)/register/complete/page.tsx",
];

describe("registration API usage", () => {
  it("uses the shared API client instead of raw fetch in registration pages", () => {
    const pagesWithRawFetch = registrationPages
      .map((relativePath) => ({
        path: relativePath,
        source: readFileSync(join(process.cwd(), relativePath), "utf8"),
      }))
      .filter(({ source }) => /\bfetch\s*\(/.test(source))
      .map(({ path }) => path);

    expect(pagesWithRawFetch).toEqual([]);
  });
});
