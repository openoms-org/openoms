import { describe, expect, it } from "vitest";
import { ORDER_SOURCES } from "@/lib/constants";
import {
  isLiveOrderChannel,
  liveMarketplaceIntegrations,
  liveOrderSources,
} from "@/lib/live-order-channels";

describe("live order channels", () => {
  it("does not treat Kaufland, Empik, or Mirakl as a live order channel", () => {
    expect(isLiveOrderChannel("kaufland")).toBe(false);
    expect(isLiveOrderChannel("empik")).toBe(false);
    expect(isLiveOrderChannel("mirakl")).toBe(false);
  });

  it("keeps Allegro and other wired marketplaces as live order channels", () => {
    expect(isLiveOrderChannel("allegro")).toBe(true);
    expect(isLiveOrderChannel("amazon")).toBe(true);
    expect(isLiveOrderChannel("olx")).toBe(true);
  });

  it("drops Kaufland and Empik from the order-source picker", () => {
    const live = liveOrderSources(ORDER_SOURCES);
    expect(live).toContain("allegro");
    expect(live).not.toContain("kaufland");
    expect(live).not.toContain("empik");
  });

  it("hides Kaufland and Empik integrations from marketplace connect lists", () => {
    const live = liveMarketplaceIntegrations([
      { provider: "allegro" },
      { provider: "kaufland" },
      { provider: "empik" },
      { provider: "mirakl" },
    ]);
    expect(live).toEqual([{ provider: "allegro" }]);
  });
});
