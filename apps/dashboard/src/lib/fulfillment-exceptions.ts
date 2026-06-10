// Exception-population derivation for the fulfillment control tower (OPE-529).
//
// The home-page exceptions feed renders a CAPPED preview (limit 3), so its
// item count understates the real exception volume. The true population is
// derivable from the operations summary the panel already fetches: the
// backend's OperationsExceptions (apps/api-server/internal/service/
// fulfillment_read_service.go) scans processes that are blocked,
// waiting_external, or unhealthy in_progress — i.e. exactly the summary's
// blocked + provider_issue + stuck buckets. The per-blocker refinement into
// missing_data/provider_issue only RELABELS members of that same population.
// Keep in sync with fulfillment_parity_service.go (process_backed_exceptions).

import type { OperatorBucket } from "@/types/fulfillment";

/** Summary buckets that make up the operator exception population. */
export const EXCEPTION_BUCKETS = [
  "blocked",
  "stuck",
  "provider_issue",
] as const satisfies readonly OperatorBucket[];

/**
 * Derives the true exception population from the operations-summary bucket
 * counts; undefined when the summary is unavailable (loading or errored).
 */
export function exceptionPopulationFromBuckets(
  buckets: Partial<Record<OperatorBucket, number>> | undefined,
): number | undefined {
  if (!buckets) {
    return undefined;
  }
  return EXCEPTION_BUCKETS.reduce(
    (sum, bucket) => sum + (buckets[bucket] ?? 0),
    0,
  );
}
