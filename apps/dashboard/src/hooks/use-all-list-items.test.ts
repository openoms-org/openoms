import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchAllListItems, useAllListItems } from "@/hooks/use-all-list-items";
import { useAuthStore } from "@/lib/auth";
import type { ListResponse } from "@/types/api";

const apiClientMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api-client", () => ({
  apiClient: apiClientMock,
}));

// useAllListItems now auth-guards by default (enabled: isAuthenticated); the hook
// test must be authenticated for the query to fire.
beforeEach(() => {
  useAuthStore.setState({ isAuthenticated: true });
});

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

describe("useAllListItems", () => {
  it("passes the React Query abort signal to every page request", async () => {
    apiClientMock.mockResolvedValue(listResponse([{ id: "1" }], 1, 0));

    renderHook(
      () =>
        useAllListItems<Item, { active: boolean }>(
          ["items", "all"],
          "/v1/items",
          { active: true },
        ),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(apiClientMock).toHaveBeenCalledTimes(1));

    expect(apiClientMock).toHaveBeenCalledWith(
      "/v1/items?active=true&limit=100&offset=0",
      expect.objectContaining({
        signal: expect.any(AbortSignal),
      }),
    );
  });
});

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

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
