import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { mockProducts } from "@/test/handlers";
import { useProducts } from "@/hooks/use-products";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("useProducts", () => {
  it("returns product data from the API", async () => {
    const { result } = renderHook(() => useProducts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBeDefined();
    expect(result.current.data!.items).toHaveLength(mockProducts.length);
    expect(result.current.data!.items[0].name).toBe("Widget A");
    expect(result.current.data!.items[1].name).toBe("Widget B");
  });

  it("handles loading state", () => {
    const { result } = renderHook(() => useProducts(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it("returns product SKUs", async () => {
    const { result } = renderHook(() => useProducts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.items[0].sku).toBe("WA-001");
    expect(result.current.data!.items[1].sku).toBe("WB-002");
  });

  it("returns product prices", async () => {
    const { result } = renderHook(() => useProducts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.items[0].price).toBe(49.99);
    expect(result.current.data!.items[1].price).toBe(89.99);
  });

  it("returns correct response shape", async () => {
    const { result } = renderHook(() => useProducts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveProperty("items");
    expect(result.current.data).toHaveProperty("total");
    expect(result.current.data).toHaveProperty("limit");
    expect(result.current.data).toHaveProperty("offset");
  });
});
