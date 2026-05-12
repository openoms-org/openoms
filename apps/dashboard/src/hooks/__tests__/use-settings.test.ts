import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import {
  mockCompanySettings,
  mockInventorySettings,
} from "@/test/handlers";
import {
  useCompanySettings,
  useInventorySettings,
} from "@/hooks/use-settings";

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

describe("useCompanySettings", () => {
  it("returns company settings from the API", async () => {
    const { result } = renderHook(() => useCompanySettings(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBeDefined();
    expect(result.current.data!.company_name).toBe(
      mockCompanySettings.company_name
    );
    expect(result.current.data!.nip).toBe(mockCompanySettings.nip);
    expect(result.current.data!.city).toBe(mockCompanySettings.city);
    expect(result.current.data!.post_code).toBe(mockCompanySettings.post_code);
  });

  it("handles loading state", () => {
    const { result } = renderHook(() => useCompanySettings(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it("returns contact information", async () => {
    const { result } = renderHook(() => useCompanySettings(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data!.phone).toBe(mockCompanySettings.phone);
    expect(result.current.data!.email).toBe(mockCompanySettings.email);
    expect(result.current.data!.website).toBe(mockCompanySettings.website);
  });
});

describe("useInventorySettings", () => {
  it("returns inventory settings from the API", async () => {
    const { result } = renderHook(() => useInventorySettings(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBeDefined();
    expect(result.current.data!.strict_mode).toBe(
      mockInventorySettings.strict_mode
    );
  });

  it("handles loading state", () => {
    const { result } = renderHook(() => useInventorySettings(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });
});
