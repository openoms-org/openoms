import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useLinkSupplierProduct } from "@/hooks/use-suppliers";

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

describe("useLinkSupplierProduct", () => {
  it("POSTs product_id to /v1/suppliers/{id}/products/{spid}/link", async () => {
    let method = "";
    let path = "";
    let body: { product_id?: string } = {};
    server.use(
      http.post(
        `${API_BASE}/suppliers/:supplierId/products/:spId/link`,
        async ({ request, params }) => {
          method = request.method;
          path = `/v1/suppliers/${params.supplierId}/products/${params.spId}/link`;
          body = (await request.json()) as { product_id?: string };
          return HttpResponse.json({ id: params.spId, product_id: body.product_id });
        }
      )
    );

    const { result } = renderHook(() => useLinkSupplierProduct("sup-1"), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ supplierProductId: "sp-9", productId: "prod-7" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(path).toBe("/v1/suppliers/sup-1/products/sp-9/link");
    expect(body.product_id).toBe("prod-7");
  });
});
