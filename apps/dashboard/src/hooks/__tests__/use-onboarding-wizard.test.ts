import { describe, it, expect, beforeAll, beforeEach, afterAll, afterEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useAuthStore } from "@/lib/auth";

const API_BASE = "*/v1";

// ---------------------------------------------------------------------------
// Hook imports — these will fail until implementation exists
// ---------------------------------------------------------------------------
import {
  useOnboardingStatus,
  useUpdateOnboardingStep,
  useCompleteOnboarding,
} from "@/hooks/use-onboarding-wizard";

// ---------------------------------------------------------------------------
// Test wrapper
// ---------------------------------------------------------------------------
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
beforeEach(() => {
  // Set authenticated state so useOnboardingStatus query is enabled
  useAuthStore.getState().setAuth("test-token", {} as never, {} as never);
});
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// ---------------------------------------------------------------------------
// useOnboardingStatus
// ---------------------------------------------------------------------------
describe("useOnboardingStatus", () => {
  it("fetches onboarding status from /v1/onboarding/status", async () => {
    server.use(
      http.get(`${API_BASE}/onboarding/status`, () => {
        return HttpResponse.json({
          completed: false,
          current_step: 1,
          completed_steps: [],
          skipped_steps: [],
          completed_at: null,
          dismissed: false,
        });
      })
    );

    const { result } = renderHook(() => useOnboardingStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.data).toBeDefined());

    expect(result.current.data?.completed).toBe(false);
    expect(result.current.data?.current_step).toBe(1);
    expect(result.current.data?.completed_steps).toEqual([]);
    expect(result.current.data?.skipped_steps).toEqual([]);
  });

  it("returns completed=true for finished onboarding", async () => {
    server.use(
      http.get(`${API_BASE}/onboarding/status`, () => {
        return HttpResponse.json({
          completed: true,
          current_step: 4,
          completed_steps: [1, 2, 3, 4],
          skipped_steps: [],
          completed_at: "2025-03-01T12:00:00Z",
          dismissed: false,
        });
      })
    );

    const { result } = renderHook(() => useOnboardingStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.data?.completed).toBe(true));
    expect(result.current.data?.completed_steps).toEqual([1, 2, 3, 4]);
    expect(result.current.data?.completed_at).toBe("2025-03-01T12:00:00Z");
  });

  it("returns dismissed=true for existing tenants", async () => {
    server.use(
      http.get(`${API_BASE}/onboarding/status`, () => {
        return HttpResponse.json({
          completed: false,
          current_step: 1,
          completed_steps: [],
          skipped_steps: [],
          completed_at: null,
          dismissed: true,
        });
      })
    );

    const { result } = renderHook(() => useOnboardingStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.data?.dismissed).toBe(true));
  });

  it("handles partially completed onboarding", async () => {
    server.use(
      http.get(`${API_BASE}/onboarding/status`, () => {
        return HttpResponse.json({
          completed: false,
          current_step: 3,
          completed_steps: [1],
          skipped_steps: [2],
          completed_at: null,
          dismissed: false,
        });
      })
    );

    const { result } = renderHook(() => useOnboardingStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.current_step).toBe(3);
    expect(result.current.data?.completed_steps).toEqual([1]);
    expect(result.current.data?.skipped_steps).toEqual([2]);
  });
});

// ---------------------------------------------------------------------------
// useUpdateOnboardingStep
// ---------------------------------------------------------------------------
describe("useUpdateOnboardingStep", () => {
  it("sends PUT request with step number and action", async () => {
    let capturedUrl = "";
    let capturedBody: Record<string, unknown> = {};

    server.use(
      http.put(`${API_BASE}/onboarding/step/:step`, async ({ request, params }) => {
        capturedUrl = `/v1/onboarding/step/${params.step}`;
        capturedBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ message: "step updated" });
      })
    );

    const { result } = renderHook(() => useUpdateOnboardingStep(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate({ step: 1, action: "completed" });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(capturedUrl).toBe("/v1/onboarding/step/1");
    expect(capturedBody).toEqual({ action: "completed" });
  });

  it("supports skipped action for steps 2-4", async () => {
    let capturedBody: Record<string, unknown> = {};

    server.use(
      http.put(`${API_BASE}/onboarding/step/:step`, async ({ request }) => {
        capturedBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ message: "step updated" });
      })
    );

    const { result } = renderHook(() => useUpdateOnboardingStep(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate({ step: 2, action: "skipped" });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(capturedBody).toEqual({ action: "skipped" });
  });

  it("handles server error gracefully", async () => {
    server.use(
      http.put(`${API_BASE}/onboarding/step/:step`, () => {
        return HttpResponse.json(
          { error: "step 1 cannot be skipped" },
          { status: 400 }
        );
      })
    );

    const { result } = renderHook(() => useUpdateOnboardingStep(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate({ step: 1, action: "skipped" });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ---------------------------------------------------------------------------
// useCompleteOnboarding
// ---------------------------------------------------------------------------
describe("useCompleteOnboarding", () => {
  it("sends POST request to /v1/onboarding/complete", async () => {
    let wasCalled = false;

    server.use(
      http.post(`${API_BASE}/onboarding/complete`, () => {
        wasCalled = true;
        return HttpResponse.json({ message: "onboarding completed" });
      })
    );

    const { result } = renderHook(() => useCompleteOnboarding(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate();
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(wasCalled).toBe(true);
  });

  it("handles error when step 1 not completed", async () => {
    server.use(
      http.post(`${API_BASE}/onboarding/complete`, () => {
        return HttpResponse.json(
          { error: "step 1 must be completed" },
          { status: 400 }
        );
      })
    );

    const { result } = renderHook(() => useCompleteOnboarding(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate();
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
