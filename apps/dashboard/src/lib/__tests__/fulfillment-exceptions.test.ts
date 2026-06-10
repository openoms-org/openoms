import { describe, expect, it } from "vitest";
import {
  EXCEPTION_BUCKETS,
  exceptionPopulationFromBuckets,
} from "@/lib/fulfillment-exceptions";

describe("exceptionPopulationFromBuckets", () => {
  it("sums exactly the blocked + stuck + provider_issue buckets", () => {
    expect(
      exceptionPopulationFromBuckets({
        ready: 9,
        processing: 4,
        stuck: 1,
        blocked: 3,
        provider_issue: 2,
        missing_data: 5,
      }),
    ).toBe(6);
  });

  it("treats absent buckets as zero", () => {
    expect(exceptionPopulationFromBuckets({ blocked: 2 })).toBe(2);
    expect(exceptionPopulationFromBuckets({})).toBe(0);
  });

  it("returns undefined when the summary is unavailable", () => {
    expect(exceptionPopulationFromBuckets(undefined)).toBeUndefined();
  });

  it("mirrors the backend exception population definition", () => {
    // Backend: OperationsExceptions scans blocked + waiting_external +
    // unhealthy in_progress processes — the summary's blocked, provider_issue
    // and stuck buckets respectively.
    expect([...EXCEPTION_BUCKETS].sort()).toEqual([
      "blocked",
      "provider_issue",
      "stuck",
    ]);
  });
});
