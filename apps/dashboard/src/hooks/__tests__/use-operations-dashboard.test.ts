import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { integrationQueryKeys } from "@/hooks/integration-query-keys";
import { useOperationsDashboard } from "@/hooks/use-operations-dashboard";
import { useAuthStore } from "@/lib/auth";

const apiClientMock = vi.hoisted(() => vi.fn());
const statsRefetchMock = vi.hoisted(() => vi.fn());
const onHoldOrdersRefetchMock = vi.hoisted(() => vi.fn());
const failedShipmentsRefetchMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api-client", () => ({
  apiClient: apiClientMock,
}));

vi.mock("@/hooks/use-dashboard-stats", () => ({
  useDashboardStats: () => ({
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: statsRefetchMock,
  }),
}));

vi.mock("@/hooks/use-orders", () => ({
  useOrders: () => ({
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: onHoldOrdersRefetchMock,
  }),
}));

vi.mock("@/hooks/use-shipments", () => ({
  useShipments: () => ({
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: failedShipmentsRefetchMock,
  }),
}));

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

beforeEach(() => {
  vi.clearAllMocks();
  statsRefetchMock.mockResolvedValue({ data: undefined });
  onHoldOrdersRefetchMock.mockResolvedValue({ data: undefined });
  failedShipmentsRefetchMock.mockResolvedValue({ data: undefined });
  useAuthStore.setState({
    token: "test-token",
    isAuthenticated: true,
    isLoading: false,
    user: {
      id: "member",
      email: "member@test.com",
      role: "member",
      role_id: "member",
      name: "Member User",
    },
    tenant: { id: "tenant", slug: "tenant", name: "Tenant", plan: "pro" },
  });
});

describe("useOperationsDashboard", () => {
  it("uses the canonical integrations query key for admin integration health", async () => {
    useAuthStore.setState({
      user: {
        id: "admin",
        email: "admin@test.com",
        role: "admin",
        role_id: "admin",
        name: "Admin User",
      },
    });
    apiClientMock.mockResolvedValue([]);
    const queryClient = createTestQueryClient();

    renderHook(() => useOperationsDashboard(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith("/v1/integrations");
    });

    expect(
      queryClient.getQueryCache().find({
        queryKey: integrationQueryKeys.all,
        exact: true,
      })
    ).toBeDefined();
    expect(
      queryClient.getQueryCache().find({
        queryKey: [...integrationQueryKeys.all, "operations-dashboard"],
        exact: true,
      })
    ).toBeUndefined();
  });

  it("does not fetch or refetch admin-only integrations for non-admin users", async () => {
    const { result } = renderHook(() => useOperationsDashboard(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(false);
    expect(apiClientMock).not.toHaveBeenCalled();

    await act(async () => {
      await result.current.refetch();
    });

    expect(statsRefetchMock).toHaveBeenCalledTimes(1);
    expect(onHoldOrdersRefetchMock).toHaveBeenCalledTimes(1);
    expect(failedShipmentsRefetchMock).toHaveBeenCalledTimes(1);
    expect(apiClientMock).not.toHaveBeenCalled();
  });
});
