import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useCreateDropshipOrder } from "@/hooks/use-dropship-orders";

const API_BASE = "*/v1";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("useCreateDropshipOrder", () => {
  it("POSTs order_id/supplier_id/items to /v1/dropship-orders", async () => {
    let method = "";
    let body: { order_id?: string; supplier_id?: string; items?: unknown[] } = {};
    server.use(
      http.post(`${API_BASE}/dropship-orders`, async ({ request }) => {
        method = request.method;
        body = (await request.json()) as typeof body;
        return HttpResponse.json({ id: "ds-1" });
      })
    );

    const { result } = renderHook(() => useCreateDropshipOrder(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({
      order_id: "ord-1",
      supplier_id: "sup-1",
      currency: "PLN",
      items: [
        { sku: "A", product_name: "Item A", quantity: 2, unit_cost: 5 },
      ],
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(body.order_id).toBe("ord-1");
    expect(body.supplier_id).toBe("sup-1");
    expect(body.items).toHaveLength(1);
  });
});
