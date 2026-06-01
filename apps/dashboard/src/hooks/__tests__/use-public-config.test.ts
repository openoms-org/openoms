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

    // Wait on license_enabled (default is false) rather than registration_mode,
    // since the fail-closed default is now also "closed" — keying on it would let
    // waitFor pass on the default before the second fetch resolves.
    await waitFor(() => {
      expect(second.result.current.license_enabled).toBe(true);
    });
    expect(second.result.current.registration_mode).toBe("closed");
    expect(second.result.current.billing_enabled).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("fails closed (never 'open') when the public config cannot be loaded", async () => {
    fetchMock.mockRejectedValue(new Error("network down"));

    const { result } = renderHook(() => usePublicConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.registration_mode).not.toBe("open");
    expect(result.current.registration_mode).toBe("closed");
    expect(result.current.license_enabled).toBe(false);
    expect(result.current.billing_enabled).toBe(false);
  });
});
