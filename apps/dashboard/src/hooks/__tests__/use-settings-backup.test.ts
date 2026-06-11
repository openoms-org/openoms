import {
  describe,
  it,
  expect,
  beforeAll,
  afterAll,
  afterEach,
  vi,
} from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";

// useExportSettings must download a Blob via apiFetch + downloadBlob, NOT
// apiClient (which always res.json()s and can never yield a Blob).
vi.mock("@/lib/download", () => ({ downloadBlob: vi.fn() }));
import { downloadBlob } from "@/lib/download";
import { useExportSettings, useImportSettings } from "@/hooks/use-settings";

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
afterEach(() => {
  server.resetHandlers();
  vi.mocked(downloadBlob).mockClear();
});
afterAll(() => server.close());

describe("useExportSettings", () => {
  it("GETs /v1/settings/export and triggers a blob download", async () => {
    let method = "";
    server.use(
      http.get(`${API_BASE}/settings/export`, ({ request }) => {
        method = request.method;
        return new HttpResponse(JSON.stringify({ company: {} }), {
          headers: { "Content-Type": "application/json" },
        });
      })
    );

    const { result } = renderHook(() => useExportSettings(), {
      wrapper: createWrapper(),
    });

    result.current.mutate();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("GET");
    expect(downloadBlob).toHaveBeenCalledTimes(1);
    const [blobArg, nameArg] = vi.mocked(downloadBlob).mock.calls[0];
    expect(blobArg).toBeInstanceOf(Blob);
    expect(String(nameArg)).toMatch(/\.json$/);
  });
});

describe("useImportSettings", () => {
  it("POSTs the parsed payload to /v1/settings/import", async () => {
    let method = "";
    let body: unknown = null;
    server.use(
      http.post(`${API_BASE}/settings/import`, async ({ request }) => {
        method = request.method;
        body = await request.json();
        return HttpResponse.json({ ok: true });
      })
    );

    const { result } = renderHook(() => useImportSettings(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ company: { company_name: "X" } });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(body).toEqual({ company: { company_name: "X" } });
  });
});
