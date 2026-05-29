// OPE-463 — sync the canonical feature-readiness registry into the dashboard.
//
// The canonical source of truth lives in the api-server package
// (apps/api-server/internal/readiness/readiness.json) and is embedded into the
// Go binary. The dashboard cannot relative-import that file at Docker build time
// because the dashboard image build context is apps/dashboard (the api-server
// directory is not present). To keep a single source of truth, this script copies
// the canonical JSON into a committed file that the dashboard imports:
//
//   apps/dashboard/src/lib/readiness.generated.json
//
// Run automatically via the `pretest`/`prebuild` npm hooks. CI re-runs it and
// fails on any drift (git diff --exit-code), so the committed copy can never go
// stale relative to the canonical source.
//
// When the canonical source is absent (e.g. inside the dashboard Docker build
// context, which only contains apps/dashboard), this script is a no-op and the
// already-committed generated file is used as-is.

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const SOURCE = resolve(
  here,
  "../../api-server/internal/readiness/readiness.json",
);
const TARGET = resolve(here, "../src/lib/readiness.generated.json");

if (!existsSync(SOURCE)) {
  console.warn(
    `[sync-readiness] canonical source not found at ${SOURCE}; ` +
      "using the committed readiness.generated.json as-is.",
  );
  process.exit(0);
}

// Validate the source parses, then re-serialize deterministically so the
// committed copy is stable regardless of source formatting.
const raw = readFileSync(SOURCE, "utf8");
let parsed;
try {
  parsed = JSON.parse(raw);
} catch (err) {
  console.error(`[sync-readiness] invalid JSON at ${SOURCE}: ${err.message}`);
  process.exit(1);
}

const serialized = `${JSON.stringify(parsed, null, 2)}\n`;

mkdirSync(dirname(TARGET), { recursive: true });

const current = existsSync(TARGET) ? readFileSync(TARGET, "utf8") : null;
if (current === serialized) {
  console.log("[sync-readiness] readiness.generated.json is up to date.");
  process.exit(0);
}

writeFileSync(TARGET, serialized);
console.log(`[sync-readiness] wrote ${TARGET}`);
