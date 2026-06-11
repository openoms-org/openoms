import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useCreateSettlement } from "@/hooks/use-reconciliation";

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

describe("useCreateSettlement", () => {
  it("POSTs the settlement header to /v1/reconciliation/settlements", async () => {
    let method = "";
    let body: { provider?: string; total_amount?: number } = {};
    server.use(
      http.post(`${API_BASE}/reconciliation/settlements`, async ({ request }) => {
        method = request.method;
        body = (await request.json()) as typeof body;
        return HttpResponse.json({ id: "settle-1" });
      })
    );

    const { result } = renderHook(() => useCreateSettlement(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({
      provider: "payu",
      settlement_date: "2026-06-01",
      total_amount: 1000,
      fee_amount: 30,
      net_amount: 970,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(body.provider).toBe("payu");
    expect(body.total_amount).toBe(1000);
  });
});
