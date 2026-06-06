import type { ProviderValidationResult } from "@/types/platform";

/**
 * Whether the current viewer may see sensitive (un-redacted) evidence. Mirrors
 * the spec rule: access to sensitive evidence requires the `providers:secrets`
 * permission (Screen 11 §537). Backend remains authoritative — this only gates
 * what the drawer offers to reveal.
 */
export function canViewSensitiveEvidence(
  permissions: string[] | undefined,
): boolean {
  return !!permissions && permissions.includes("providers:secrets");
}

const SHORT_HASH_LEN = 12;

/** A short, copy-friendly prefix of a payload hash for dense rows. */
export function shortHash(hash: string): string {
  const trimmed = hash.trim();
  if (trimmed.length <= SHORT_HASH_LEN) return trimmed;
  return `${trimmed.slice(0, SHORT_HASH_LEN)}…`;
}

/**
 * Patterns that, if they ever appeared in an observation, would indicate a
 * secret/PII leak. Observations are already redacted server-side; this is a
 * defence-in-depth mask so the drawer never renders a raw secret even if the
 * backend regressed.
 */
const SENSITIVE_PATTERNS: { re: RegExp; replacement: string }[] = [
  // bearer / authorization tokens
  { re: /(bearer\s+)[A-Za-z0-9._\-]{8,}/gi, replacement: "$1[redacted]" },
  // key=value secret-ish pairs
  {
    re: /\b(password|passwd|secret|api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|authorization)\b(\s*[:=]\s*)("?)[^\s",}]+("?)/gi,
    replacement: "$1$2$3[redacted]$4",
  },
  // long base64/hex-looking blobs (32+ chars) that aren't already a label
  { re: /\b[A-Za-z0-9+/]{32,}={0,2}\b/g, replacement: "[redacted]" },
  // email addresses (PII)
  { re: /\b[\w.+-]+@[\w-]+\.[\w.-]+\b/g, replacement: "[redacted-email]" },
];

/**
 * Produce a display-safe observation. When the viewer lacks sensitive-evidence
 * permission (or `reveal` is false), known secret/PII shapes are masked. With
 * permission AND an explicit reveal, the observation is shown as recorded
 * (already redacted upstream).
 */
export function redactObservation(
  observation: string,
  options: { canViewSensitive: boolean; reveal: boolean },
): string {
  if (options.canViewSensitive && options.reveal) {
    return observation;
  }
  let out = observation;
  for (const { re, replacement } of SENSITIVE_PATTERNS) {
    out = out.replace(re, replacement);
  }
  return out;
}

/** True when masking actually changed the observation (something was hidden). */
export function hasRedactableContent(observation: string): boolean {
  return (
    redactObservation(observation, { canViewSensitive: false, reveal: false }) !==
    observation
  );
}

/** Flatten a run's results into an evidence-timeline order (newest first). */
export function evidenceTimeline(
  results: ProviderValidationResult[] | undefined,
): ProviderValidationResult[] {
  if (!results) return [];
  return [...results].sort(
    (a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  );
}
