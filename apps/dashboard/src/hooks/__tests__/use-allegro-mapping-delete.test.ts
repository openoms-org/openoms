import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useDeleteAllegroParameterMapping } from "@/hooks/use-suppliers";

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

describe("useDeleteAllegroParameterMapping", () => {
  it("DELETEs /v1/suppliers/{id}/allegro-mappings/{mid}", async () => {
    let method = "";
    let path = "";
    server.use(
      http.delete(
        `${API_BASE}/suppliers/:supplierId/allegro-mappings/:mappingId`,
        ({ request, params }) => {
          method = request.method;
          path = `/v1/suppliers/${params.supplierId}/allegro-mappings/${params.mappingId}`;
          return new HttpResponse(null, { status: 204 });
        }
      )
    );

    const { result } = renderHook(
      () => useDeleteAllegroParameterMapping("sup-1"),
      { wrapper: createWrapper() }
    );

    result.current.mutate("map-9");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("DELETE");
    expect(path).toBe("/v1/suppliers/sup-1/allegro-mappings/map-9");
  });
});
