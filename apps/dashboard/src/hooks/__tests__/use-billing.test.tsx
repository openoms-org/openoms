import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSubscription } from "@/hooks/use-billing";
import { useAuthStore } from "@/lib/auth";

const apiClientMock = vi.hoisted(() => vi.fn());
const usePublicConfigMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api-client", () => ({
  apiClient: apiClientMock,
}));

vi.mock("@/hooks/use-public-config", () => ({
  usePublicConfig: usePublicConfigMock,
}));

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

function setPublicConfig(billingEnabled: boolean, isLoading = false) {
  usePublicConfigMock.mockReturnValue({
    registration_mode: "invite",
    license_enabled: false,
    billing_enabled: billingEnabled,
    isLoading,
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  apiClientMock.mockResolvedValue({ plan: "plus", status: "active" });
  setPublicConfig(false);
  useAuthStore.setState({
    token: "test-token",
    isAuthenticated: true,
    isLoading: false,
    user: {
      id: "user-1",
      tenant_id: "tenant-1",
      email: "admin@example.com",
      role: "owner",
      role_id: "role-1",
      name: "Admin",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
    tenant: {
      id: "tenant-1",
      slug: "dev",
      name: "Dev",
      plan: "plus",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  });
});

describe("useSubscription", () => {
  it("does not fetch subscription data when billing is disabled", async () => {
    renderHook(() => useSubscription(), { wrapper: createWrapper() });

    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(apiClientMock).not.toHaveBeenCalled();
  });

  it("fetches subscription data only when billing is enabled", async () => {
    setPublicConfig(true);

    renderHook(() => useSubscription(), { wrapper: createWrapper() });

    await waitFor(() => expect(apiClientMock).toHaveBeenCalledTimes(1));
    expect(apiClientMock).toHaveBeenCalledWith("/v1/billing/subscription");
  });
});
