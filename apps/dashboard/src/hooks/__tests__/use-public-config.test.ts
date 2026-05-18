import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { usePublicConfig } from "@/hooks/use-public-config";

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });
}

function createWrapper(queryClient = createTestQueryClient()) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function jsonResponse(data: unknown) {
  return Promise.resolve(
    new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    })
  );
}

const fetchMock = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
  fetchMock.mockReset();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("usePublicConfig", () => {
  it("uses the QueryClient cache instead of module-level stale config", async () => {
    fetchMock
      .mockReturnValueOnce(
        jsonResponse({
          registration_mode: "invite",
          license_enabled: false,
          billing_enabled: true,
          stripe_public_key: "pk_first",
        })
      )
      .mockReturnValueOnce(
        jsonResponse({
          registration_mode: "closed",
          license_enabled: true,
          billing_enabled: false,
        })
      );

    const first = renderHook(() => usePublicConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(first.result.current.registration_mode).toBe("invite");
    });
    first.unmount();

    const second = renderHook(() => usePublicConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(second.result.current.registration_mode).toBe("closed");
    });
    expect(second.result.current.license_enabled).toBe(true);
    expect(second.result.current.billing_enabled).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
