import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getFulfillmentDashboardMode,
  isProcessBackedDashboard,
} from "@/lib/fulfillment-dashboard";

// OPE-423c cutover flag. Default/unset MUST resolve to "heuristic" (current
// production behavior); only the exact "process-backed" opts into the new view.
describe("getFulfillmentDashboardMode", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("defaults to heuristic when the flag is unset", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD", undefined as never);
    expect(getFulfillmentDashboardMode()).toBe("heuristic");
    expect(isProcessBackedDashboard()).toBe(false);
  });

  it("returns process-backed only for the exact opt-in value", () => {
    vi.stubEnv(
      "NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD",
      "process-backed",
    );
    expect(getFulfillmentDashboardMode()).toBe("process-backed");
    expect(isProcessBackedDashboard()).toBe(true);
  });

  it("treats an explicit heuristic value as heuristic", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD", "heuristic");
    expect(getFulfillmentDashboardMode()).toBe("heuristic");
    expect(isProcessBackedDashboard()).toBe(false);
  });

  it("falls back to heuristic for any unknown value", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD", "something-else");
    expect(getFulfillmentDashboardMode()).toBe("heuristic");
    expect(isProcessBackedDashboard()).toBe(false);
  });

  it("falls back to heuristic for an empty string", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD", "");
    expect(getFulfillmentDashboardMode()).toBe("heuristic");
  });
});
