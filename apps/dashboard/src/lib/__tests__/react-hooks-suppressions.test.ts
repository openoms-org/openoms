import { describe, expect, it } from "vitest";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

const dashboardSrc = join(process.cwd(), "src");
const allowedFiles = new Set(["src/hooks/use-effect-synced-state.ts"]);
const ruleName = ["react-hooks", "set-state-in-effect"].join("/");
const escapedRuleName = ruleName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const suppressionPattern = new RegExp(
  String.raw`\beslint-disable(?:-next-line|-line)?\b.*\b${escapedRuleName}\b`,
);

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const fullPath = join(dir, entry);
    if (entry === "node_modules" || entry === ".next") return [];
    if (statSync(fullPath).isDirectory()) return walk(fullPath);
    return /\.(ts|tsx)$/.test(entry) ? [fullPath] : [];
  });
}

describe("React hooks lint suppressions", () => {
  it("keeps set-state-in-effect suppressions centralized and documented", () => {
    const offenders = walk(dashboardSrc).flatMap((file) => {
      const relativePath = relative(process.cwd(), file).split(sep).join("/");
      const lines = readFileSync(file, "utf8").split("\n");

      return lines.flatMap((line, index) => {
        if (!suppressionPattern.test(line)) return [];
        if (allowedFiles.has(relativePath)) return [];
        return [`${relativePath}:${index + 1}`];
      });
    });

    expect(offenders).toEqual([]);
  });
});
