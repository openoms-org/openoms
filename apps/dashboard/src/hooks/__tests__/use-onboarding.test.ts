import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { useOnboarding } from "@/hooks/use-onboarding";
import { useAuthStore } from "@/lib/auth";

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
beforeEach(() => {
  // Set authenticated state so queries with enabled:isAuthenticated fire
  useAuthStore.setState({
    token: "test-token",
    isAuthenticated: true,
    isLoading: false,
    user: { id: "test", email: "test@test.com", role: "admin", role_id: "test", name: "Test User" },
    tenant: { id: "test", slug: "test", name: "Test", plan: "pro" },
  });
});
afterEach(() => {
  useAuthStore.setState({ token: null, user: null, tenant: null, isAuthenticated: false, isLoading: true });
  server.resetHandlers();
});
afterAll(() => server.close());

describe("useOnboarding", () => {
  it("does not fetch admin-only setup resources for non-admin users", async () => {
    let integrationsRequested = false;
    server.use(
      http.get(`${API_BASE}/integrations`, () => {
        integrationsRequested = true;
        return HttpResponse.json([]);
      }),
    );
    useAuthStore.setState({
      user: {
        id: "member",
        email: "member@test.com",
        role: "member",
        role_id: "member",
        name: "Member User",
      },
    });

    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.steps).toHaveLength(0));
    expect(result.current.isVisible).toBe(false);
    expect(integrationsRequested).toBe(false);
  });

  it("returns 4 steps", async () => {
    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.steps).toHaveLength(4));
  });

  it("marks company step as completed when company_name and NIP exist", async () => {
    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      const companyStep = result.current.steps.find(
        (s) => s.key === "company"
      );
      expect(companyStep?.completed).toBe(true);
    });
  });

  it("marks company step as incomplete when company_name is missing", async () => {
    server.use(
      http.get(`${API_BASE}/settings/company`, () => {
        return HttpResponse.json({ nip: "1234567890" });
      })
    );

    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      const companyStep = result.current.steps.find(
        (s) => s.key === "company"
      );
      expect(companyStep?.completed).toBe(false);
    });
  });

  it("marks integration step as incomplete when no integrations", async () => {
    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      const integrationStep = result.current.steps.find(
        (s) => s.key === "integration"
      );
      expect(integrationStep?.completed).toBe(false);
    });
  });

  it("links the Allegro onboarding step to the client-ready marketplace setup", async () => {
    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      const integrationStep = result.current.steps.find(
        (s) => s.key === "integration"
      );
      expect(integrationStep?.href).toBe("/marketplaces/new");
    });
  });

  it("marks integration step as completed when active Allegro exists", async () => {
    server.use(
      http.get(`${API_BASE}/integrations`, () => {
        return HttpResponse.json([
          { provider: "allegro", status: "active" },
        ]);
      })
    );

    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      const integrationStep = result.current.steps.find(
        (s) => s.key === "integration"
      );
      expect(integrationStep?.completed).toBe(true);
    });
  });

  it("is not visible when dismissed", async () => {
    server.use(
      http.get(`${API_BASE}/settings/onboarding`, () => {
        return HttpResponse.json({ dismissed: true });
      })
    );

    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isVisible).toBe(false));
  });

  it("marks order step as completed when dashboard stats show orders", async () => {
    // Default mockDashboardStats has order_counts.total = 150
    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      const orderStep = result.current.steps.find((s) => s.key === "order");
      expect(orderStep?.completed).toBe(true);
    });
  });

  it("reports correct completedCount", async () => {
    // Default handlers: company=done (name+nip), integration=incomplete (empty array),
    // product=done (stats.order_counts.total=150), order=done (stats.order_counts.total=150)
    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.completedCount).toBe(3);
    });
  });

  it("is not visible when all steps are completed", async () => {
    server.use(
      http.get(`${API_BASE}/integrations`, () => {
        return HttpResponse.json([
          { provider: "allegro", status: "active" },
        ]);
      })
    );

    const { result } = renderHook(() => useOnboarding(), {
      wrapper: createWrapper(),
    });

    // company=done, integration=done, product=done, order=done => allCompleted=true => isVisible=false
    await waitFor(() => {
      expect(result.current.allCompleted).toBe(true);
      expect(result.current.isVisible).toBe(false);
    });
  });
});
