import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useUpdatePurchaseOrder } from "@/hooks/use-purchase-orders";
import { useUpdateRecurringOrder } from "@/hooks/use-recurring-orders";

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

describe("useUpdatePurchaseOrder", () => {
  it("PUTs to /v1/purchase-orders/{id}", async () => {
    let method = "";
    let path = "";
    server.use(
      http.put(`${API_BASE}/purchase-orders/:id`, ({ request, params }) => {
        method = request.method;
        path = `/v1/purchase-orders/${params.id}`;
        return HttpResponse.json({ id: params.id });
      })
    );

    const { result } = renderHook(() => useUpdatePurchaseOrder("po-1"), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ supplier_name: "Acme", items: [] });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("PUT");
    expect(path).toBe("/v1/purchase-orders/po-1");
  });
});

describe("useUpdateRecurringOrder", () => {
  it("PUTs to /v1/recurring-orders/{id}", async () => {
    let method = "";
    let path = "";
    server.use(
      http.put(`${API_BASE}/recurring-orders/:id`, ({ request, params }) => {
        method = request.method;
        path = `/v1/recurring-orders/${params.id}`;
        return HttpResponse.json({ id: params.id });
      })
    );

    const { result } = renderHook(() => useUpdateRecurringOrder("ro-1"), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ frequency: "weekly" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("PUT");
    expect(path).toBe("/v1/recurring-orders/ro-1");
  });
});
