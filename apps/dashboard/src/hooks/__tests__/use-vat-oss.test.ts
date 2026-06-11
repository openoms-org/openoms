import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useCalculateVAT } from "@/hooks/use-vat-oss";

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

describe("useCalculateVAT", () => {
  it("POSTs amount/country/rate_type to /v1/vat-oss/calculate", async () => {
    let method = "";
    let body: { amount?: number; country?: string; rate_type?: string } = {};
    server.use(
      http.post(`${API_BASE}/vat-oss/calculate`, async ({ request }) => {
        method = request.method;
        body = (await request.json()) as typeof body;
        return HttpResponse.json({
          net_amount: 100,
          vat_amount: 23,
          gross_amount: 123,
          vat_rate: 23,
          country: "DE",
          rate_type: "standard",
        });
      })
    );

    const { result } = renderHook(() => useCalculateVAT(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ amount: 100, country: "DE", rate_type: "standard" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(body).toEqual({ amount: 100, country: "DE", rate_type: "standard" });
    expect(result.current.data?.gross_amount).toBe(123);
  });
});
