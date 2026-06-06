import type {
  ProbeSafetyClass,
  ProbeType,
  ProviderValidationProbe,
  ProviderValidationResult,
  ResultStatus,
  RunVerdict,
} from "@/types/platform";

type BadgeTone = "success" | "warning" | "info" | "secondary" | "destructive" | "outline";

/**
 * Probe types that always mutate a provider's state when run for real (Screen 8
 * "production_write" / "sandbox_write" level). `sandbox_order_create` is a write
 * that targets the sandbox/test environment. All other declared probe types are
 * read-only. A probe additionally flagged `destructive` overrides this to the
 * destructive class regardless of type.
 */
const WRITE_PROBE_TYPES: ReadonlySet<string> = new Set<ProbeType>([
  "sandbox_order_create",
]);

/**
 * Derive a probe's safety class (Screen 8). NOT a persisted backend column:
 * derived from probe type + the `destructive` flag. The authoritative backend
 * control is the per-run `allow_destructive` flag, which a destructive probe
 * requires before it will run.
 *
 * - destructive flag                  -> destructive
 * - write probe in production env     -> production_write
 * - write probe (sandbox env)         -> sandbox_write
 * - everything else                   -> read_only
 */
export function probeSafetyClass(
  probe: Pick<ProviderValidationProbe, "probe_type" | "destructive">,
  environment: "sandbox" | "production",
): ProbeSafetyClass {
  if (probe.destructive) {
    return "destructive";
  }
  if (WRITE_PROBE_TYPES.has(probe.probe_type)) {
    return environment === "production" ? "production_write" : "sandbox_write";
  }
  return "read_only";
}

/** Tone for a probe safety class — never color-only (carries a label too). */
const SAFETY_TONES: Record<ProbeSafetyClass, BadgeTone> = {
  read_only: "secondary",
  sandbox_write: "info",
  production_write: "warning",
  destructive: "destructive",
};

export function safetyClassTone(safety: ProbeSafetyClass): BadgeTone {
  return SAFETY_TONES[safety] ?? "outline";
}

/** Tone for a completed run verdict. */
const VERDICT_TONES: Record<RunVerdict, BadgeTone> = {
  pending: "info",
  passed: "success",
  failed: "destructive",
  error: "destructive",
};

export function verdictTone(verdict: RunVerdict): BadgeTone {
  return VERDICT_TONES[verdict] ?? "outline";
}

/** Tone for a per-probe result status. */
const RESULT_TONES: Record<ResultStatus, BadgeTone> = {
  passed: "success",
  failed: "destructive",
  skipped: "secondary",
  error: "destructive",
};

export function resultStatusTone(status: ResultStatus): BadgeTone {
  return RESULT_TONES[status] ?? "outline";
}

/**
 * Whether a run is still open for recording results. Mirrors the backend rule:
 * results can only be recorded while the verdict is `pending`.
 */
export function isRunOpen(run: { verdict: RunVerdict }): boolean {
  return run.verdict === "pending";
}

/**
 * Group probes by safety class for the lab's left panel, ordered from safest to
 * most dangerous so read-only probes lead and destructive probes are visually
 * separated (Screen 8 "never part of default suite").
 */
const SAFETY_ORDER: ProbeSafetyClass[] = [
  "read_only",
  "sandbox_write",
  "production_write",
  "destructive",
];

export function groupProbesBySafety(
  probes: ProviderValidationProbe[],
  environment: "sandbox" | "production",
): { safety: ProbeSafetyClass; probes: ProviderValidationProbe[] }[] {
  const byClass = new Map<ProbeSafetyClass, ProviderValidationProbe[]>();
  for (const probe of probes) {
    const safety = probeSafetyClass(probe, environment);
    const bucket = byClass.get(safety);
    if (bucket) {
      bucket.push(probe);
    } else {
      byClass.set(safety, [probe]);
    }
  }
  return SAFETY_ORDER.filter((s) => byClass.has(s)).map((safety) => ({
    safety,
    probes: byClass.get(safety) ?? [],
  }));
}

/**
 * True when running the given probes (in this environment) would touch a
 * destructive probe — used to require the explicit destructive confirmation and
 * to set `allow_destructive` on the run.
 */
export function hasDestructiveProbe(
  probes: Pick<ProviderValidationProbe, "probe_type" | "destructive">[],
): boolean {
  return probes.some((p) => p.destructive);
}

/**
 * Compute a run verdict from results, mirroring model.VerdictFromResults: any
 * errored probe -> error, any failed probe -> failed, no results -> error,
 * otherwise passed. Used to preview the verdict before a run is completed.
 */
export function verdictFromResults(
  results: ProviderValidationResult[],
): RunVerdict {
  if (results.length === 0) {
    return "error";
  }
  let verdict: RunVerdict = "passed";
  for (const r of results) {
    if (r.status === "error") {
      return "error";
    }
    if (r.status === "failed") {
      verdict = "failed";
    }
  }
  return verdict;
}
