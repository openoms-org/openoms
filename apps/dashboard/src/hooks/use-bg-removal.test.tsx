import { createElement, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { apiFetch } from "@/lib/api-client";
import { useRemoveBackground } from "./use-bg-removal";

vi.mock("@/lib/api-client", () => ({
  API_URL: "https://api.test",
  apiClient: vi.fn(),
  apiFetch: vi.fn(),
}));

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

beforeEach(() => {
  vi.mocked(apiFetch).mockReset();
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ url: "https://cdn.test/raw-fetch.png" }), { status: 200 }),
  );
});

describe("useRemoveBackground", () => {
  it("uploads through apiFetch so auth refresh and CSRF handling are preserved", async () => {
    vi.mocked(apiFetch).mockResolvedValue(
      new Response(JSON.stringify({ url: "https://cdn.test/bg-removed.png" }), { status: 200 }),
    );

    const file = new File(["image"], "image.png", { type: "image/png" });
    const { result } = renderHook(() => useRemoveBackground(), { wrapper: createWrapper() });

    const data = await act(async () => result.current.mutateAsync(file));

    expect(apiFetch).toHaveBeenCalledTimes(1);
    expect(apiFetch).toHaveBeenCalledWith(
      "/v1/images/remove-background",
      expect.objectContaining({
        method: "POST",
        body: expect.any(FormData),
      }),
    );
    const [, init] = vi.mocked(apiFetch).mock.calls[0];
    expect((init?.body as FormData).get("file")).toBe(file);
    expect(globalThis.fetch).not.toHaveBeenCalled();
    expect(data).toEqual({ url: "https://cdn.test/bg-removed.png" });
  });
});
