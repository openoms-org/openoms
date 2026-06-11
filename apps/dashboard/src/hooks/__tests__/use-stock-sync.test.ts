import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import {
  usePushProductStock,
  useReconcileProductStock,
} from "@/hooks/use-stock-sync";

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

describe("usePushProductStock", () => {
  it("POSTs to /v1/stock-sync/push/{productId}", async () => {
    let method = "";
    let path = "";
    server.use(
      http.post(`${API_BASE}/stock-sync/push/:productId`, ({ request, params }) => {
        method = request.method;
        path = `/v1/stock-sync/push/${params.productId}`;
        return HttpResponse.json({ message: "ok" });
      })
    );

    const { result } = renderHook(() => usePushProductStock(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("prod-001");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(path).toBe("/v1/stock-sync/push/prod-001");
  });
});

describe("useReconcileProductStock", () => {
  it("POSTs to /v1/stock-sync/reconcile/{productId}", async () => {
    let method = "";
    let path = "";
    server.use(
      http.post(
        `${API_BASE}/stock-sync/reconcile/:productId`,
        ({ request, params }) => {
          method = request.method;
          path = `/v1/stock-sync/reconcile/${params.productId}`;
          return HttpResponse.json({ id: "evt-1" });
        }
      )
    );

    const { result } = renderHook(() => useReconcileProductStock(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("prod-002");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(path).toBe("/v1/stock-sync/reconcile/prod-002");
  });
});
