import { describe, expect, it } from "vitest";
import { buildSearchParams } from "@/lib/search-params";

describe("buildSearchParams", () => {
  it("serializes array values as comma-separated query parameters", () => {
    const params = buildSearchParams({
      order_ids: ["order-1", "order-2"],
      limit: 50,
    });

    expect(params.toString()).toBe("order_ids=order-1%2Corder-2&limit=50");
  });

  it("skips empty array values", () => {
    const params = buildSearchParams({
      order_ids: [],
      limit: 50,
    });

    expect(params.toString()).toBe("limit=50");
  });
});
