import { describe, it, expect, vi, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import {
  render,
  screen,
  userEvent,
  renderWithProviders,
} from "@/test/test-utils";
import { OperationsBucketSummary } from "@/components/dashboard/operations-bucket-summary";
import { FulfillmentExceptionsFeed } from "@/components/dashboard/fulfillment-exceptions-feed";
import { FulfillmentDetailPanel } from "@/components/fulfillment/fulfillment-detail-panel";
import { useResolveBlocker, useRetryStep } from "@/hooks/use-fulfillment";
import type {
  FulfillmentExceptionItem,
  FulfillmentProcessDetail,
  OperatorBucket,
} from "@/types/fulfillment";

// next-intl is globally mocked (vitest.setup.ts) to echo translation keys, so
// assertions target the i18n key strings rather than localized copy.

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function hookWrapper() {
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

const ZERO_BUCKETS: Record<OperatorBucket, number> = {
  ready: 0,
  processing: 0,
  stuck: 0,
  blocked: 0,
  provider_issue: 0,
  missing_data: 0,
};

function exceptionItem(
  overrides: Partial<FulfillmentExceptionItem> = {},
): FulfillmentExceptionItem {
  return {
    process: {
      id: "proc-1",
      tenant_id: "tenant-1",
      order_id: "order-99",
      aggregate_status: "blocked",
      health_status: "action_required",
      created_at: "2026-06-01T10:00:00Z",
      updated_at: "2026-06-01T12:00:00Z",
    },
    bucket: "missing_data",
    top_blocker: {
      id: "blk-1",
      tenant_id: "tenant-1",
      process_id: "proc-1",
      code: "external_status_unmapped",
      category: "mapping",
      status: "open",
      description: "Brak mapowania usługi kuriera",
      created_at: "2026-06-01T11:00:00Z",
      updated_at: "2026-06-01T11:00:00Z",
    },
    ...overrides,
  };
}

describe("OperationsBucketSummary", () => {
  it("renders all six operator buckets with their counts", () => {
    render(
      <OperationsBucketSummary
        buckets={{ ...ZERO_BUCKETS, blocked: 3, provider_issue: 2 }}
        total={5}
        isLoading={false}
      />,
    );

    // All six bucket labels are present.
    for (const bucket of [
      "ready",
      "processing",
      "stuck",
      "blocked",
      "provider_issue",
      "missing_data",
    ] as OperatorBucket[]) {
      expect(
        screen.getByText(`fulfillment.buckets.${bucket}.label`),
      ).toBeInTheDocument();
    }
    // Counts render (blocked=3, provider_issue=2).
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("renders an intentional all-zero state (the prod default with the flag off)", () => {
    render(
      <OperationsBucketSummary buckets={ZERO_BUCKETS} total={0} isLoading={false} />,
    );
    // Six zero cells, not an error/crash.
    expect(screen.getAllByText("0")).toHaveLength(6);
    expect(
      screen.queryByText("fulfillment.controlTower.loadError"),
    ).not.toBeInTheDocument();
  });

  it("renders skeletons while loading", () => {
    const { container } = render(
      <OperationsBucketSummary isLoading={true} />,
    );
    expect(container.querySelectorAll('[data-slot="skeleton"]').length).toBeGreaterThan(0);
  });

  it("shows an error state with retry", async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();
    render(
      <OperationsBucketSummary isLoading={false} isError onRetry={onRetry} />,
    );
    expect(
      screen.getByText("fulfillment.controlTower.loadError"),
    ).toBeInTheDocument();
    await user.click(screen.getByText("fulfillment.retry"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});

describe("FulfillmentExceptionsFeed", () => {
  it("renders an exception with its top-blocker reason and calls onSelect (drilldown target) with the item", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    const item = exceptionItem();

    render(
      <FulfillmentExceptionsFeed
        items={[item]}
        isLoading={false}
        onSelect={onSelect}
      />,
    );

    // The reason comes from the canonical blocker code (not provider internals).
    expect(
      screen.getByText("fulfillment.blockerCode.external_status_unmapped"),
    ).toBeInTheDocument();

    const row = screen.getByTestId("fulfillment-exception-item");
    // Drilldown target metadata is exposed for the panel/spec.
    expect(row).toHaveAttribute("data-process-id", "proc-1");
    expect(row).toHaveAttribute("data-order-id", "order-99");

    await user.click(row);
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledWith(item);
  });

  it("renders the empty state when there are no exceptions", () => {
    render(
      <FulfillmentExceptionsFeed items={[]} isLoading={false} onSelect={vi.fn()} />,
    );
    expect(
      screen.getByText("fulfillment.exceptions.emptyTitle"),
    ).toBeInTheDocument();
  });

  it("qualifies a capped preview badge against the true population (shown of total)", () => {
    const items = ["proc-1", "proc-2", "proc-3"].map((id) =>
      exceptionItem({ process: { ...exceptionItem().process, id } }),
    );

    render(
      <FulfillmentExceptionsFeed
        items={items}
        isLoading={false}
        onSelect={vi.fn()}
        maxItems={3}
        totalCount={12}
      />,
    );

    // The badge must NOT present the capped preview size as the total.
    expect(
      screen.getByText("fulfillment.exceptions.shownOfTotal"),
    ).toBeInTheDocument();
    expect(screen.queryByText("3")).not.toBeInTheDocument();
  });

  it("shows the bare count when the capped preview covers the whole population", () => {
    const item = exceptionItem();

    render(
      <FulfillmentExceptionsFeed
        items={[item]}
        isLoading={false}
        onSelect={vi.fn()}
        maxItems={3}
        totalCount={1}
      />,
    );

    expect(screen.getByText("1")).toBeInTheDocument();
    expect(
      screen.queryByText("fulfillment.exceptions.shownOfTotal"),
    ).not.toBeInTheDocument();
  });

  it("marks a capped badge as a preview when the population is unknown", () => {
    render(
      <FulfillmentExceptionsFeed
        items={[exceptionItem()]}
        isLoading={false}
        onSelect={vi.fn()}
        maxItems={3}
      />,
    );

    expect(
      screen.getByText("fulfillment.exceptions.topPreview"),
    ).toBeInTheDocument();
    expect(screen.queryByText("1")).not.toBeInTheDocument();
  });

  it("keeps the bare count when the list is not capped", () => {
    render(
      <FulfillmentExceptionsFeed
        items={[exceptionItem()]}
        isLoading={false}
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("humanizes a blocker code missing from the catalogs instead of leaking the raw key", () => {
    const item = exceptionItem({
      top_blocker: {
        ...exceptionItem().top_blocker!,
        code: "brand_new_backend_code",
      },
    });

    render(
      <FulfillmentExceptionsFeed
        items={[item]}
        isLoading={false}
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByText("Brand new backend code")).toBeInTheDocument();
    expect(
      screen.queryByText("fulfillment.blockerCode.brand_new_backend_code"),
    ).not.toBeInTheDocument();
  });
});

describe("FulfillmentDetailPanel actions", () => {
  const detail: FulfillmentProcessDetail = {
    process: {
      id: "proc-1",
      tenant_id: "tenant-1",
      order_id: "order-99",
      aggregate_status: "blocked",
      health_status: "action_required",
      created_at: "2026-06-01T10:00:00Z",
      updated_at: "2026-06-01T12:00:00Z",
    },
    blockers: [
      {
        id: "blk-1",
        tenant_id: "tenant-1",
        process_id: "proc-1",
        code: "external_status_unmapped",
        category: "mapping",
        status: "open",
        description: "Brak mapowania",
        created_at: "2026-06-01T11:00:00Z",
        updated_at: "2026-06-01T11:00:00Z",
      },
    ],
    units: [
      {
        unit: {
          id: "unit-1",
          tenant_id: "tenant-1",
          process_id: "proc-1",
          unit_type: "warehouse",
          status: "blocked",
          created_at: "2026-06-01T10:00:00Z",
          updated_at: "2026-06-01T12:00:00Z",
        },
        steps: [
          {
            id: "step-1",
            tenant_id: "tenant-1",
            unit_id: "unit-1",
            step_key: "generate_label",
            status: "failed",
            attempts: 3,
            created_at: "2026-06-01T10:00:00Z",
            updated_at: "2026-06-01T12:00:00Z",
          },
        ],
      },
    ],
    provider_attempts: [],
  };

  it("wires Resolve to POST the blocker resolve endpoint after confirm", async () => {
    const user = userEvent.setup();
    let resolveHit = "";
    server.use(
      http.get("*/v1/fulfillment/processes/proc-1", () =>
        HttpResponse.json(detail),
      ),
      http.post("*/v1/fulfillment/blockers/:id/resolve", ({ params }) => {
        resolveHit = String(params.id);
        return HttpResponse.json({ ...detail.blockers[0], status: "resolved" });
      }),
    );

    renderWithProviders(
      <FulfillmentDetailPanel processId="proc-1" open onOpenChange={vi.fn()} />,
    );

    // Open the blocker's Resolve action -> confirm dialog -> confirm.
    const resolveBtn = await screen.findByTestId("resolve-blocker-button");
    await user.click(resolveBtn);
    // Confirm dialog confirm button uses the same action key label.
    const confirmButtons = await screen.findAllByText("fulfillment.actions.resolve");
    await user.click(confirmButtons[confirmButtons.length - 1]);

    await waitFor(() => expect(resolveHit).toBe("blk-1"));
  });

  it("wires Retry to POST the step retry endpoint after confirm", async () => {
    const user = userEvent.setup();
    let retryHit = "";
    server.use(
      http.get("*/v1/fulfillment/processes/proc-1", () =>
        HttpResponse.json(detail),
      ),
      http.post("*/v1/fulfillment/steps/:id/retry", ({ params }) => {
        retryHit = String(params.id);
        return HttpResponse.json({ ...detail.units[0].steps[0], status: "pending" });
      }),
    );

    renderWithProviders(
      <FulfillmentDetailPanel processId="proc-1" open onOpenChange={vi.fn()} />,
    );

    const retryBtn = await screen.findByTestId("retry-step-button");
    await user.click(retryBtn);
    const confirmButtons = await screen.findAllByText("fulfillment.actions.retry");
    await user.click(confirmButtons[confirmButtons.length - 1]);

    await waitFor(() => expect(retryHit).toBe("step-1"));
  });

  it("labels known provider operations via i18n and humanizes unknown blocker codes / operations", async () => {
    const withAttempts: FulfillmentProcessDetail = {
      ...detail,
      blockers: [
        {
          ...detail.blockers[0],
          id: "blk-unknown",
          code: "vendor_specific_surprise",
        },
      ],
      provider_attempts: [
        {
          id: "att-1",
          tenant_id: "tenant-1",
          process_id: "proc-1",
          provider: "allegro",
          operation: "sync_tracking_to_marketplace",
          status: "succeeded",
          created_at: "2026-06-01T12:00:00Z",
        },
        {
          id: "att-2",
          tenant_id: "tenant-1",
          process_id: "proc-1",
          provider: "inpost",
          operation: "emit_carrier_manifest",
          status: "failed",
          created_at: "2026-06-01T12:05:00Z",
        },
      ],
    };
    server.use(
      http.get("*/v1/fulfillment/processes/proc-1", () =>
        HttpResponse.json(withAttempts),
      ),
    );

    renderWithProviders(
      <FulfillmentDetailPanel processId="proc-1" open onOpenChange={vi.fn()} />,
    );

    // Known operation -> translated (the mock echoes the existing key).
    expect(
      await screen.findByText(
        "fulfillment.providerOp.sync_tracking_to_marketplace",
      ),
    ).toBeInTheDocument();
    // Unknown operation/blocker code -> humanized text, never the raw key.
    expect(screen.getByText("Emit carrier manifest")).toBeInTheDocument();
    expect(screen.getByText("Vendor specific surprise")).toBeInTheDocument();
    expect(
      screen.queryByText("fulfillment.providerOp.emit_carrier_manifest"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("fulfillment.blockerCode.vendor_specific_surprise"),
    ).not.toBeInTheDocument();
  });
});

describe("fulfillment action hooks", () => {
  it("useResolveBlocker POSTs to the resolve endpoint", async () => {
    let hit = "";
    server.use(
      http.post("*/v1/fulfillment/blockers/:id/resolve", ({ params }) => {
        hit = String(params.id);
        return HttpResponse.json({ id: params.id, status: "resolved" });
      }),
    );

    const { result } = renderHook(() => useResolveBlocker(), {
      wrapper: hookWrapper(),
    });
    result.current.mutate("blk-42");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(hit).toBe("blk-42");
  });

  it("useRetryStep surfaces a 400 (non-retryable) as an error", async () => {
    server.use(
      http.post("*/v1/fulfillment/steps/:id/retry", () =>
        HttpResponse.json({ error: "step not retryable" }, { status: 400 }),
      ),
    );

    const { result } = renderHook(() => useRetryStep(), {
      wrapper: hookWrapper(),
    });
    result.current.mutate("step-7");
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
