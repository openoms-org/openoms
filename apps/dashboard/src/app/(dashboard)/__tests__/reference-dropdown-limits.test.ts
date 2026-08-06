import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";

const referenceListLimitPattern =
  /use(?:Products|Suppliers|Warehouses|PriceLists)\s*\(\s*\{[^}]*limit:\s*100/;

describe("reference dropdown list limits", () => {
  it("does not use one-page limit:100 queries for reference dropdown options", () => {
    const root = process.cwd();
    const files = [
      ...tsxFiles(join(root, "src/app/(dashboard)")),
      ...tsxFiles(join(root, "src/components")),
    ];

    const offenders = files
      .map((file) => {
        return {
          file: relative(root, file),
          source: readFileSync(file, "utf8"),
        };
      })
      .filter(({ source }) => referenceListLimitPattern.test(source))
      .map(({ file }) => file);

    expect(offenders).toEqual([]);
  });
});

function tsxFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) {
      return tsxFiles(path);
    }
    return path.endsWith(".tsx") ? [path] : [];
  });
}
