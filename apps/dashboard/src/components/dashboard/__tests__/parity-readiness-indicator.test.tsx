import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@/test/test-utils";
import { ParityReadinessIndicator } from "@/components/dashboard/parity-readiness-indicator";
import { useAuthStore } from "@/lib/auth";
import type { FulfillmentParityReport } from "@/types/fulfillment";

// next-intl is globally mocked (vitest.setup.ts) to echo translation keys, so
// assertions target the i18n key strings rather than localized copy.

const useFulfillmentParityMock = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/use-fulfillment", () => ({
  useFulfillmentParity: useFulfillmentParityMock,
}));

function setAuthRole(role: string | null) {
  useAuthStore.setState({
    token: "token",
    tenant: null,
    user: role ? ({ id: "u1", role, name: "Op" } as never) : null,
    isAuthenticated: true,
    isLoading: false,
    locale: "pl",
  });
}

function parityState(
  overrides: Partial<{
    data?: FulfillmentParityReport;
    isLoading: boolean;
    isError: boolean;
  }> = {},
) {
  return {
    data: overrides.data,
    isLoading: overrides.isLoading ?? false,
    isError: overrides.isError ?? false,
    refetch: vi.fn(),
  };
}

function report(
  overrides: Partial<FulfillmentParityReport> = {},
): FulfillmentParityReport {
  return {
    non_terminal_orders: 100,
    fulfillment_processes: 100,
    orders_missing_process: 0,
    process_coverage: 1,
    legacy_problem_orders: 2,
    process_backed_exceptions: 3,
    coverage_threshold: 0.99,
    process_coverage_met: true,
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  setAuthRole("admin");
});

describe("ParityReadinessIndicator", () => {
  it("renders nothing for non-operator users", () => {
    setAuthRole("member");
    useFulfillmentParityMock.mockReturnValue(parityState({ data: report() }));

    const { container } = render(<ParityReadinessIndicator />);
    expect(container).toBeEmptyDOMElement();
    // Operator-only: the parity hook must not even be relied upon for rendering.
    expect(
      screen.queryByTestId("parity-readiness-indicator"),
    ).not.toBeInTheDocument();
  });

  it("renders the loading state", () => {
    useFulfillmentParityMock.mockReturnValue(parityState({ isLoading: true }));

    render(<ParityReadinessIndicator />);
    expect(screen.getByTestId("parity-readiness-indicator")).toBeInTheDocument();
    expect(
      screen.queryByTestId("parity-verdict"),
    ).not.toBeInTheDocument();
  });

  it("renders the error state with a retry affordance", () => {
    useFulfillmentParityMock.mockReturnValue(parityState({ isError: true }));

    render(<ParityReadinessIndicator />);
    expect(
      screen.getByText("fulfillment.parity.loadError"),
    ).toBeInTheDocument();
    expect(screen.getByText("fulfillment.retry")).toBeInTheDocument();
  });

  it("renders the empty state when there is nothing to compare", () => {
    useFulfillmentParityMock.mockReturnValue(
      parityState({
        data: report({
          non_terminal_orders: 0,
          fulfillment_processes: 0,
          process_coverage: 1,
          process_coverage_met: true,
        }),
      }),
    );

    render(<ParityReadinessIndicator />);
    expect(
      screen.getByText("fulfillment.parity.emptyTitle"),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("parity-verdict")).not.toBeInTheDocument();
  });

  it("renders the ready verdict and coverage when parity is met", () => {
    useFulfillmentParityMock.mockReturnValue(
      parityState({
        data: report({
          process_coverage: 0.995,
          orders_missing_process: 0,
          process_coverage_met: true,
        }),
      }),
    );

    render(<ParityReadinessIndicator />);
    const verdict = screen.getByTestId("parity-verdict");
    expect(verdict).toHaveAttribute("data-met", "true");
    expect(screen.getByText("fulfillment.parity.ready")).toBeInTheDocument();
    // Coverage rounded to a whole percent.
    expect(screen.getByText("100%")).toBeInTheDocument();
  });

  it("renders the not-ready verdict and the missing-process gap when below threshold", () => {
    useFulfillmentParityMock.mockReturnValue(
      parityState({
        data: report({
          non_terminal_orders: 100,
          fulfillment_processes: 80,
          orders_missing_process: 20,
          process_coverage: 0.8,
          process_coverage_met: false,
        }),
      }),
    );

    render(<ParityReadinessIndicator />);
    const verdict = screen.getByTestId("parity-verdict");
    expect(verdict).toHaveAttribute("data-met", "false");
    expect(screen.getByText("fulfillment.parity.notReady")).toBeInTheDocument();
    expect(screen.getByText("80%")).toBeInTheDocument();
    // The missing-process gap is shown with its value.
    expect(
      screen.getByText("fulfillment.parity.missingProcesses"),
    ).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
  });
});
