import { describe, expect, it } from "vitest";

import { fetchAllListItems } from "@/hooks/use-all-list-items";
import type { ListResponse } from "@/types/api";

interface Item {
  id: string;
}

describe("fetchAllListItems", () => {
  it("loads every page using the backend page limit and preserves filters", async () => {
    const calls: Array<{ search?: string; limit: number; offset: number }> = [];

    const result = await fetchAllListItems<Item, { search?: string }>(
      { search: "abc" },
      async (params) => {
        calls.push(params);
        if (params.offset === 0) {
          return listResponse([{ id: "1" }], 2, 0);
        }
        return listResponse([{ id: "2" }], 2, 100);
      },
    );

    expect(calls).toEqual([
      { search: "abc", limit: 100, offset: 0 },
      { search: "abc", limit: 100, offset: 100 },
    ]);
    expect(result).toEqual({
      items: [{ id: "1" }, { id: "2" }],
      total: 2,
      limit: 2,
      offset: 0,
    });
  });
});

function listResponse(
  items: Item[],
  total: number,
  offset: number,
): ListResponse<Item> {
  return {
    items,
    total,
    limit: 100,
    offset,
  };
}
