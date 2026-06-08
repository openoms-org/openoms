import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useFulfillmentParity } from "@/hooks/use-fulfillment";
import type { FulfillmentParityReport } from "@/types/fulfillment";

const apiClientMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api-client", () => ({
  apiClient: apiClientMock,
}));

function wrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

const REPORT: FulfillmentParityReport = {
  non_terminal_orders: 100,
  fulfillment_processes: 100,
  orders_missing_process: 0,
  process_coverage: 1,
  legacy_problem_orders: 2,
  process_backed_exceptions: 3,
  coverage_threshold: 0.99,
  process_coverage_met: true,
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useFulfillmentParity", () => {
  it("calls GET /v1/operations/parity and returns the report", async () => {
    apiClientMock.mockResolvedValue(REPORT);

    const { result } = renderHook(() => useFulfillmentParity(), {
      wrapper: wrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(apiClientMock).toHaveBeenCalledWith("/v1/operations/parity");
    expect(result.current.data).toEqual(REPORT);
  });

  it("does not fetch when disabled", () => {
    apiClientMock.mockResolvedValue(REPORT);

    renderHook(() => useFulfillmentParity({ enabled: false }), {
      wrapper: wrapper(),
    });

    expect(apiClientMock).not.toHaveBeenCalled();
  });

  it("surfaces an error state when the request fails", async () => {
    apiClientMock.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useFulfillmentParity(), {
      wrapper: wrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
