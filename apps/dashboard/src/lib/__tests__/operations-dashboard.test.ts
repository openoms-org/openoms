import { describe, expect, it } from "vitest";
import {
  buildIntegrationHealth,
  buildOperationalExceptions,
  buildOperationsActivity,
  buildOrchestrationStages,
} from "@/lib/operations-dashboard";
import type {
  DashboardStats,
  Integration,
  ListResponse,
  Order,
  Shipment,
} from "@/types/api";

function listResponse<T>(items: T[]): ListResponse<T> {
  return {
    items,
    total: items.length,
    limit: items.length,
    offset: 0,
  };
}

function dashboardStats(byStatus: Record<string, number>): DashboardStats {
  return {
    order_counts: {
      total: Object.values(byStatus).reduce((sum, count) => sum + count, 0),
      by_status: byStatus,
      by_source: {},
    },
    revenue: {
      total: 0,
      currency: "PLN",
      daily: [],
    },
    recent_orders: [],
  };
}

function order(overrides: Partial<Order> = {}): Order {
  return {
    id: "order-1",
    tenant_id: "tenant-1",
    source: "manual",
    status: "new",
    customer_name: "Jan Kowalski",
    total_amount: 199,
    currency: "PLN",
    tags: [],
    payment_status: "paid",
    created_at: "2026-05-01T08:00:00Z",
    updated_at: "2026-05-01T08:00:00Z",
    ...overrides,
  };
}

function shipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    id: "shipment-1",
    tenant_id: "tenant-1",
    order_id: "order-1",
    provider: "inpost",
    status: "failed",
    package_number: 1,
    notes: "",
    created_at: "2026-05-01T09:00:00Z",
    updated_at: "2026-05-01T09:00:00Z",
    ...overrides,
  };
}

function integration(overrides: Partial<Integration> = {}): Integration {
  return {
    id: "integration-1",
    tenant_id: "tenant-1",
    provider: "allegro",
    label: "Allegro main",
    status: "active",
    has_credentials: true,
    created_at: "2026-05-01T10:00:00Z",
    updated_at: "2026-05-01T10:00:00Z",
    ...overrides,
  };
}

describe("buildOrchestrationStages", () => {
  it("groups statuses into intake, fulfillment, shipping, and completed stages", () => {
    const stages = buildOrchestrationStages(
      dashboardStats({
        new: 3,
        confirmed: 2,
        processing: 4,
        ready_to_ship: 5,
        shipped: 6,
        in_transit: 7,
        out_for_delivery: 8,
        delivered: 9,
        completed: 10,
        cancelled: 1,
        refunded: 2,
      }),
    );

    expect(
      stages.map(({ key, count, exceptionCount, health, href }) => ({
        key,
        count,
        exceptionCount,
        health,
        href,
      })),
    ).toEqual([
      { key: "intake", count: 5, exceptionCount: 0, health: "ok", href: "/orders" },
      { key: "fulfillment", count: 9, exceptionCount: 0, health: "ok", href: "/orders" },
      { key: "shipping", count: 21, exceptionCount: 0, health: "ok", href: "/shipments" },
      { key: "completed", count: 19, exceptionCount: 3, health: "warning", href: "/orders" },
    ]);
  });

  it("uses dashboard operations stage translation keys", () => {
    const stages = buildOrchestrationStages();

    expect(stages.map((stage) => stage.labelKey)).toEqual([
      "operations.stages.intake",
      "operations.stages.fulfillment",
      "operations.stages.shipping",
      "operations.stages.completed",
    ]);
  });

  it("marks fulfillment as warning when on_hold orders exist", () => {
    const stages = buildOrchestrationStages(
      dashboardStats({
        processing: 4,
        ready_to_ship: 5,
        on_hold: 2,
      }),
    );

    expect(stages.find((stage) => stage.key === "fulfillment")).toMatchObject({
      exceptionCount: 2,
      health: "warning",
    });
  });
});

describe("buildOperationalExceptions", () => {
  it("returns existing-route links and translation keys without action labels", () => {
    const exceptions = buildOperationalExceptions({
      onHoldOrders: listResponse([
        order({
          id: "order-hold",
          status: "on_hold",
          updated_at: "2026-05-01T08:00:00Z",
        }),
      ]),
      failedShipments: listResponse([
        shipment({
          id: "shipment-failed",
          status: "failed",
          updated_at: "2026-05-01T09:00:00Z",
        }),
      ]),
      integrations: [
        integration({
          id: "integration-error",
          provider: "allegro",
          status: "error",
          error_message: "Token expired",
          updated_at: "2026-05-01T10:00:00Z",
        }),
      ],
    });

    expect(exceptions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          kind: "integration",
          severity: "problem",
          titleKey: "operations.exceptions.integration.title",
          descriptionKey: "operations.exceptions.integration.description",
          primaryHref: "/marketplaces",
          values: expect.objectContaining({
            provider: "Allegro",
          }),
        }),
        expect.objectContaining({
          kind: "shipment",
          severity: "problem",
          titleKey: "operations.exceptions.shipment.title",
          descriptionKey: "operations.exceptions.shipment.description",
          primaryHref: "/shipments",
        }),
        expect.objectContaining({
          kind: "order",
          severity: "warning",
          titleKey: "operations.exceptions.orderOnHold.title",
          descriptionKey: "operations.exceptions.orderOnHold.description",
          primaryHref: "/orders/order-hold",
        }),
      ]),
    );

    for (const exception of exceptions) {
      expect(exception).not.toHaveProperty("primaryActionLabel");
      expect(exception).not.toHaveProperty("title");
      expect(exception).not.toHaveProperty("description");
      expect(Object.keys(exception).some((key) => key.toLowerCase().includes("action"))).toBe(false);
    }
  });

  it("hides error integrations without a reachable client-ready fix surface", () => {
    const exceptions = buildOperationalExceptions({
      integrations: [
        integration({
          id: "olx",
          provider: "olx",
          label: "OLX",
          status: "error",
          error_message: "Reconnect required",
        }),
      ],
    });

    expect(exceptions).toEqual([]);
  });

  it("keeps ready provider errors visible as problems", () => {
    const exceptions = buildOperationalExceptions({
      integrations: [
        integration({
          id: "allegro",
          provider: "allegro",
          label: "Allegro",
          status: "error",
          error_message: "Token expired",
        }),
      ],
    });

    expect(exceptions).toEqual([
      expect.objectContaining({
        kind: "integration",
        severity: "problem",
        primaryHref: "/marketplaces",
        values: expect.objectContaining({
          provider: "Allegro",
          errorMessage: "Token expired",
        }),
      }),
    ]);
  });

  it("limits results to 7 by default", () => {
    const exceptions = buildOperationalExceptions({
      onHoldOrders: listResponse(
        Array.from({ length: 9 }, (_, index) =>
          order({
            id: `order-${index}`,
            status: "on_hold",
            updated_at: `2026-05-01T0${index}:00:00Z`,
          }),
        ),
      ),
    });

    expect(exceptions).toHaveLength(7);
  });
});

describe("buildIntegrationHealth", () => {
  it("shows ready allegro and inpost providers while hiding blocked shopify", () => {
    const health = buildIntegrationHealth([
      integration({ id: "allegro", provider: "allegro", label: "Allegro", status: "active" }),
      integration({ id: "inpost", provider: "inpost", label: "InPost", status: "active" }),
      integration({ id: "shopify", provider: "shopify", label: "Shopify", status: "active" }),
    ]);

    expect(health.map((item) => item.provider)).toEqual(["allegro", "inpost"]);
    expect(health.map((item) => item.health)).toEqual(["ok", "ok"]);
  });

  it("hides controlled provider errors in client-ready mode when no fix surface is reachable", () => {
    const health = buildIntegrationHealth([
      integration({
        id: "olx",
        provider: "olx",
        label: "OLX",
        status: "error",
        error_message: "Reconnect required",
      }),
    ]);

    expect(health).toEqual([]);
  });

  it("shows ready provider errors as problems", () => {
    const health = buildIntegrationHealth([
      integration({
        id: "allegro",
        provider: "allegro",
        label: "Allegro",
        status: "error",
        error_message: "Token expired",
      }),
    ]);

    expect(health).toEqual([
      expect.objectContaining({
        id: "allegro",
        provider: "allegro",
        health: "problem",
        errorMessage: "Token expired",
        href: "/marketplaces",
      }),
    ]);
  });
});

describe("buildOperationsActivity", () => {
  it("maps recent orders into operational activity links", () => {
    const activity = buildOperationsActivity({
      ...dashboardStats({ new: 1 }),
      recent_orders: [
        {
          id: "order-1",
          customer_name: "Jan Kowalski",
          status: "new",
          source: "allegro",
          total_amount: 199,
          currency: "PLN",
          created_at: "2026-05-01T08:00:00Z",
        },
      ],
    });

    expect(activity).toEqual([
      expect.objectContaining({
        id: "order-1",
        titleKey: "operations.activity.order.title",
        descriptionKey: "operations.activity.order.description",
        values: {
          customerName: "Jan Kowalski",
          source: "allegro",
          status: "new",
        },
        href: "/orders/order-1",
        source: "allegro",
        createdAt: "2026-05-01T08:00:00Z",
      }),
    ]);

    expect(activity[0]).not.toHaveProperty("title");
    expect(activity[0]).not.toHaveProperty("description");
  });
});
