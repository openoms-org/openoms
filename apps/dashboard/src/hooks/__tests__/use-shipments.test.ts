import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { mockShipments } from "@/test/handlers";
import { useShipments } from "@/hooks/use-shipments";

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

describe("useShipments", () => {
  it("returns shipment data from the API", async () => {
    const { result } = renderHook(() => useShipments(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBeDefined();
    expect(result.current.data!.items).toHaveLength(mockShipments.length);
    expect(result.current.data!.items[0].provider).toBe("inpost");
    expect(result.current.data!.items[1].provider).toBe("dhl");
  });

  it("handles loading state", () => {
    const { result } = renderHook(() => useShipments(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it("returns total count", async () => {
    const { result } = renderHook(() => useShipments(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.total).toBe(mockShipments.length);
  });

  it("returns correct data structure with limit and offset", async () => {
    const { result } = renderHook(() => useShipments(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.limit).toBe(20);
    expect(result.current.data!.offset).toBe(0);
  });

  it("returns tracking numbers", async () => {
    const { result } = renderHook(() => useShipments(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.items[0].tracking_number).toBe("INP123456789");
    expect(result.current.data!.items[1].tracking_number).toBe("DHL987654321");
  });

  it("returns shipment statuses", async () => {
    const { result } = renderHook(() => useShipments(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.items[0].status).toBe("created");
    expect(result.current.data!.items[1].status).toBe("shipped");
  });
});
