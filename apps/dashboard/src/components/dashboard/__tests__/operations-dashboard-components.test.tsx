import { describe, expect, it } from "vitest";
import { render, screen } from "@/test/test-utils";
import { IntegrationHealthPanel } from "@/components/dashboard/integration-health-panel";
import { OperationalExceptions } from "@/components/dashboard/operational-exceptions";
import { OperationsActivity } from "@/components/dashboard/operations-activity";
import { OrchestrationMap } from "@/components/dashboard/orchestration-map";
import type {
  IntegrationHealthItem,
  OperationsActivityItem,
  OperationalException,
  OrchestrationStage,
} from "@/lib/operations-dashboard";

const stages: OrchestrationStage[] = [
  {
    key: "intake",
    labelKey: "operations.stages.intake",
    count: 4,
    exceptionCount: 0,
    health: "ok",
    href: "/orders",
  },
  {
    key: "fulfillment",
    labelKey: "operations.stages.fulfillment",
    count: 3,
    exceptionCount: 1,
    health: "warning",
    href: "/orders",
  },
  {
    key: "shipping",
    labelKey: "operations.stages.shipping",
    count: 2,
    exceptionCount: 0,
    health: "ok",
    href: "/shipments",
  },
  {
    key: "completed",
    labelKey: "operations.stages.completed",
    count: 8,
    exceptionCount: 0,
    health: "ok",
    href: "/orders",
  },
];

describe("operations dashboard components", () => {
  it("renders orchestration stages as links to existing workflow pages", () => {
    render(<OrchestrationMap stages={stages} isLoading={false} />);

    expect(screen.getByText("operations.stages.intake")).toBeInTheDocument();

    const links = screen.getAllByRole("link");
    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      "/orders",
      "/orders",
      "/shipments",
      "/orders",
    ]);
  });

  it("renders operational exceptions through translation keys without action buttons", () => {
    const exceptions: OperationalException[] = [
      {
        id: "exception-1",
        kind: "order",
        severity: "warning",
        titleKey: "operations.exceptions.orderOnHold.title",
        descriptionKey: "operations.exceptions.orderOnHold.description",
        values: {
          orderId: "order-1",
          customerName: "Jan Kowalski",
        },
        primaryHref: "/orders/order-1",
      },
    ];

    render(<OperationalExceptions exceptions={exceptions} isLoading={false} />);

    expect(screen.getByText("operations.exceptions.orderOnHold.title")).toBeInTheDocument();
    expect(screen.getByText("operations.exceptions.orderOnHold.description")).toBeInTheDocument();
    expect(
      screen.getByRole("link", {
        name: /operations\.exceptions\.orderOnHold\.title/i,
      }),
    ).toHaveAttribute("href", "/orders/order-1");
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("does not require stale title or description fields for exceptions", () => {
    render(
      <OperationalExceptions
        exceptions={[
          {
            id: "exception-title-key-only",
            kind: "shipment",
            severity: "problem",
            titleKey: "operations.exceptions.shipment.title",
            descriptionKey: "operations.exceptions.shipment.description",
            values: {
              orderId: "order-1",
            },
            primaryHref: "/shipments",
          },
        ]}
        isLoading={false}
      />,
    );

    expect(screen.getByText("operations.exceptions.shipment.title")).toBeInTheDocument();
    expect(screen.getByText("operations.exceptions.shipment.description")).toBeInTheDocument();
  });

  it("renders integration health items as links to existing integration pages", () => {
    const items: IntegrationHealthItem[] = [
      {
        id: "carrier-inpost",
        provider: "inpost",
        label: "InPost",
        health: "ok",
        status: "active",
        href: "/carriers",
      },
    ];

    render(<IntegrationHealthPanel items={items} isLoading={false} />);

    expect(screen.getByRole("link", { name: /InPost/i })).toHaveAttribute(
      "href",
      "/carriers",
    );
  });

  it("renders activity items through translation keys as links to order details", () => {
    const items: OperationsActivityItem[] = [
      {
        id: "activity-order-1",
        titleKey: "operations.activity.order.title",
        descriptionKey: "operations.activity.order.description",
        values: {
          customerName: "Jan Kowalski",
          source: "manual",
          status: "new",
        },
        href: "/orders/order-1",
        createdAt: "2026-05-01T08:00:00Z",
        source: "manual",
      },
    ];

    render(<OperationsActivity items={items} isLoading={false} />);

    expect(screen.getByText("operations.activity.order.title")).toBeInTheDocument();
    expect(screen.getByText("operations.activity.order.description")).toBeInTheDocument();
    expect(
      screen.getByRole("link", {
        name: /operations\.activity\.order\.title/i,
      }),
    ).toHaveAttribute("href", "/orders/order-1");
  });
});
