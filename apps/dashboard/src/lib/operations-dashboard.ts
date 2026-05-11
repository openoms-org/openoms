import { PROVIDER_CATEGORIES } from "@/lib/constants";
import { getProviderDisplayName } from "@/lib/provider-info";
import { getProviderReadiness, isReadinessVisible } from "@/lib/readiness";
import type {
  DashboardStats,
  Integration,
  ListResponse,
  Order,
  Shipment,
} from "@/types/api";

export type OperationsHealth = "ok" | "warning" | "problem";
export type OperationsViewModelValues = Record<string, string | number>;

export interface OrchestrationStage {
  key: "intake" | "fulfillment" | "shipping" | "completed";
  labelKey: string;
  count: number;
  exceptionCount: number;
  health: OperationsHealth;
  href: string;
}

export interface OperationalException {
  id: string;
  kind: "order" | "shipment" | "integration";
  severity: "warning" | "problem";
  titleKey: string;
  descriptionKey: string;
  values?: OperationsViewModelValues;
  primaryHref: string;
  createdAt?: string;
  primaryActionLabel?: never;
}

export interface IntegrationHealthItem {
  id: string;
  provider: string;
  label: string;
  health: OperationsHealth;
  status: Integration["status"];
  href: string;
  errorMessage?: string;
  lastSyncAt?: string;
}

export interface OperationsActivityItem {
  id: string;
  titleKey: string;
  descriptionKey: string;
  values?: OperationsViewModelValues;
  href: string;
  createdAt: string;
  source: string;
}

export interface BuildOperationalExceptionsInput {
  onHoldOrders?: ListResponse<Order>;
  failedShipments?: ListResponse<Shipment>;
  integrations?: Integration[];
  limit?: number;
}

const DEFAULT_EXCEPTION_LIMIT = 7;
const MAX_ACTIVITY_ITEMS = 5;

export function buildOrchestrationStages(stats?: DashboardStats): OrchestrationStage[] {
  const byStatus = stats?.order_counts.by_status ?? {};
  const onHoldCount = statusCount(byStatus, "on_hold");
  const completedExceptionCount =
    statusCount(byStatus, "cancelled") + statusCount(byStatus, "refunded");

  return [
    {
      key: "intake",
      labelKey: "operations.stages.intake",
      count: statusCount(byStatus, "new") + statusCount(byStatus, "confirmed"),
      exceptionCount: 0,
      health: "ok",
      href: "/orders",
    },
    {
      key: "fulfillment",
      labelKey: "operations.stages.fulfillment",
      count: statusCount(byStatus, "processing") + statusCount(byStatus, "ready_to_ship"),
      exceptionCount: onHoldCount,
      health: onHoldCount > 0 ? "warning" : "ok",
      href: "/orders",
    },
    {
      key: "shipping",
      labelKey: "operations.stages.shipping",
      count:
        statusCount(byStatus, "shipped") +
        statusCount(byStatus, "in_transit") +
        statusCount(byStatus, "out_for_delivery"),
      exceptionCount: 0,
      health: "ok",
      href: "/shipments",
    },
    {
      key: "completed",
      labelKey: "operations.stages.completed",
      count: statusCount(byStatus, "delivered") + statusCount(byStatus, "completed"),
      exceptionCount: completedExceptionCount,
      health: completedExceptionCount > 0 ? "warning" : "ok",
      href: "/orders",
    },
  ];
}

export function buildOperationalExceptions(
  input: BuildOperationalExceptionsInput = {},
): OperationalException[] {
  const limit = Math.max(0, input.limit ?? DEFAULT_EXCEPTION_LIMIT);
  const integrationExceptions = (input.integrations ?? [])
    .filter(
      (integration) =>
        integration.status === "error" && isClientReadyProviderVisible(integration.provider),
    )
    .map(buildIntegrationException);
  const shipmentExceptions = (input.failedShipments?.items ?? [])
    .filter((shipment) => shipment.status === "failed")
    .map(buildShipmentException);
  const orderExceptions = (input.onHoldOrders?.items ?? [])
    .filter((order) => order.status === "on_hold")
    .map(buildOrderException);

  return [...integrationExceptions, ...shipmentExceptions, ...orderExceptions]
    .sort((a, b) => compareCreatedAtDesc(a.createdAt, b.createdAt))
    .slice(0, limit);
}

export function buildIntegrationHealth(integrations: Integration[] = []): IntegrationHealthItem[] {
  return integrations.flatMap((integration) => {
    if (!isClientReadyProviderVisible(integration.provider)) {
      return [];
    }

    return [
      {
        id: integration.id,
        provider: integration.provider,
        label: integration.label?.trim() || getProviderDisplayName(integration.provider),
        health: integrationHealth(integration.status),
        status: integration.status,
        href: providerCategoryHref(integration.provider),
        ...(integration.error_message ? { errorMessage: integration.error_message } : {}),
        ...(integration.last_sync_at ? { lastSyncAt: integration.last_sync_at } : {}),
      },
    ];
  });
}

export function buildOperationsActivity(stats?: DashboardStats): OperationsActivityItem[] {
  return (stats?.recent_orders ?? []).slice(0, MAX_ACTIVITY_ITEMS).map((order) => ({
    id: order.id,
    titleKey: "operations.activity.order.title",
    descriptionKey: "operations.activity.order.description",
    values: {
      customerName: order.customer_name,
      source: order.source,
      status: order.status,
    },
    href: `/orders/${order.id}`,
    createdAt: order.created_at,
    source: order.source,
  }));
}

function buildIntegrationException(integration: Integration): OperationalException {
  const createdAt = integration.updated_at || integration.last_sync_at || integration.created_at;
  const label = integration.label?.trim() || getProviderDisplayName(integration.provider);

  return {
    id: `integration-${integration.id}`,
    kind: "integration",
    severity: "problem",
    titleKey: "operations.exceptions.integration.title",
    descriptionKey: "operations.exceptions.integration.description",
    values: {
      provider: getProviderDisplayName(integration.provider),
      label,
      ...(integration.error_message ? { errorMessage: integration.error_message } : {}),
    },
    primaryHref: providerCategoryHref(integration.provider),
    ...(createdAt ? { createdAt } : {}),
  };
}

function buildShipmentException(shipment: Shipment): OperationalException {
  const createdAt = shipment.updated_at || shipment.created_at;

  return {
    id: `shipment-${shipment.id}`,
    kind: "shipment",
    severity: "problem",
    titleKey: "operations.exceptions.shipment.title",
    descriptionKey: "operations.exceptions.shipment.description",
    values: {
      provider: getProviderDisplayName(shipment.provider),
      orderId: shipment.order_id,
      status: shipment.status,
      ...(shipment.tracking_number ? { trackingNumber: shipment.tracking_number } : {}),
    },
    primaryHref: "/shipments",
    ...(createdAt ? { createdAt } : {}),
  };
}

function buildOrderException(order: Order): OperationalException {
  const createdAt = order.updated_at || order.created_at;

  return {
    id: `order-${order.id}`,
    kind: "order",
    severity: "warning",
    titleKey: "operations.exceptions.orderOnHold.title",
    descriptionKey: "operations.exceptions.orderOnHold.description",
    values: {
      orderId: order.id,
      customerName: order.customer_name,
      source: order.source,
      status: order.status,
    },
    primaryHref: `/orders/${order.id}`,
    ...(createdAt ? { createdAt } : {}),
  };
}

function statusCount(byStatus: Record<string, number>, status: string): number {
  return byStatus[status] ?? 0;
}

function integrationHealth(status: Integration["status"]): OperationsHealth {
  if (status === "error") {
    return "problem";
  }

  return status === "active" ? "ok" : "warning";
}

function isClientReadyProviderVisible(provider: string): boolean {
  return isReadinessVisible(getProviderReadiness(provider), {
    mode: "client-ready",
  });
}

function providerCategoryHref(provider: string): string {
  const category = Object.entries(PROVIDER_CATEGORIES).find(([, config]) =>
    config.providers.includes(provider),
  )?.[0];

  if (category === "carrier") {
    return "/carriers";
  }

  if (category === "marketplace") {
    return "/marketplaces";
  }

  return "/settings/company";
}

function compareCreatedAtDesc(a?: string, b?: string): number {
  return parseDate(b) - parseDate(a);
}

function parseDate(value?: string): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}
