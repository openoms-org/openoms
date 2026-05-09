import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useIntegrations } from "@/hooks/use-integrations";

const API_BASE = "*/v1";

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

describe("useIntegrations", () => {
  it("returns integration data", async () => {
    const { result } = renderHook(() => useIntegrations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBeDefined();
    expect(Array.isArray(result.current.data)).toBe(true);
  });

  it("handles loading state", () => {
    const { result } = renderHook(() => useIntegrations(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it("returns empty array by default", async () => {
    const { result } = renderHook(() => useIntegrations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(0);
  });

  it("returns integrations when they exist", async () => {
    server.use(
      http.get(`${API_BASE}/integrations`, () => {
        return HttpResponse.json([
          {
            id: "int-001",
            tenant_id: "t-1",
            provider: "allegro",
            status: "active",
            has_credentials: true,
            created_at: "2025-01-10T10:00:00Z",
            updated_at: "2025-01-10T10:00:00Z",
          },
          {
            id: "int-002",
            tenant_id: "t-1",
            provider: "amazon",
            status: "inactive",
            has_credentials: false,
            created_at: "2025-01-11T10:00:00Z",
            updated_at: "2025-01-11T10:00:00Z",
          },
        ]);
      })
    );

    const { result } = renderHook(() => useIntegrations(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(2);
    expect(result.current.data![0].provider).toBe("allegro");
    expect(result.current.data![0].status).toBe("active");
    expect(result.current.data![1].provider).toBe("amazon");
    expect(result.current.data![1].status).toBe("inactive");
  });
});
