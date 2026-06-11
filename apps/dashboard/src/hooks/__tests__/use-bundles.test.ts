import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useUpdateBundleComponent } from "@/hooks/use-bundles";

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

describe("useUpdateBundleComponent", () => {
  it("PUTs quantity to /v1/products/{id}/bundle/{componentId}", async () => {
    let method = "";
    let path = "";
    let body: unknown = null;
    server.use(
      http.put(
        `${API_BASE}/products/:productId/bundle/:componentId`,
        async ({ request, params }) => {
          method = request.method;
          path = `/v1/products/${params.productId}/bundle/${params.componentId}`;
          body = await request.json();
          return HttpResponse.json({ id: params.componentId, quantity: 5 });
        }
      )
    );

    const { result } = renderHook(() => useUpdateBundleComponent("prod-001"), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ componentId: "comp-9", data: { quantity: 5 } });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("PUT");
    expect(path).toBe("/v1/products/prod-001/bundle/comp-9");
    expect(body).toEqual({ quantity: 5 });
  });
});
