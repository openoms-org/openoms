# OPE-242 Operations Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current sales-style dashboard home with a compact OpenOMS operations control tower that shows order-flow health, a small exception queue, integration health, and recent operational activity without exposing unavailable actions.

**Architecture:** First PR is frontend-only and uses existing APIs: `/v1/stats/dashboard`, `/v1/integrations`, `/v1/orders`, and `/v1/shipments`. A pure view-model layer converts existing data into dashboard sections so components stay simple and testable. No new action buttons are introduced unless they link to an existing route or working flow.

**Tech Stack:** Next.js 16 App Router, React 19, TypeScript, TanStack Query, Tailwind v4, shadcn/ui, lucide-react, Vitest + Testing Library.

---

## Scope

This plan covers the first implementation PR for `OPE-242`.

Implementation amendment after code-quality review:

- Dashboard must not surface an integration exception unless the provider also has a reachable client-ready fix surface.
- Ready provider errors such as Allegro/InPost may be visible.
- Controlled, beta, or blocked provider errors such as OLX/Shopify stay hidden until their destination screen is available in client-ready mode.
- Operations view-model items expose translation keys and interpolation values, not baked English/Polish user-facing copy.

Included:

- Replace `/` dashboard content with an operations cockpit.
- Keep dashboard free of fake interactions.
- Use existing data only.
- Add tests for view-model rules and visible UI.
- Update docs/context.

Not included in this PR:

- New backend aggregation endpoint.
- Retry/repair buttons for failed jobs.
- Full workflow/event-stream backend.
- Showing automation/webhook exceptions that do not have a client-ready destination.
- Enabling unverified providers.

## File Structure

Create:

- `apps/dashboard/src/lib/operations-dashboard.ts`
  Pure functions and types for dashboard stages, exceptions, integration health, and activity rows.

- `apps/dashboard/src/lib/__tests__/operations-dashboard.test.ts`
  Unit tests for stage calculation, exception limits, readiness filtering, and no-fake-action guarantees.

- `apps/dashboard/src/hooks/use-operations-dashboard.ts`
  Composes existing hooks and returns a single view model for the dashboard page.

- `apps/dashboard/src/components/dashboard/orchestration-map.tsx`
  Compact flow summary. Links only to existing routes.

- `apps/dashboard/src/components/dashboard/operational-exceptions.tsx`
  Shows up to 7 actionable or informative exceptions. No fake buttons and no hidden-provider repair links.

- `apps/dashboard/src/components/dashboard/integration-health-panel.tsx`
  Shows health of ready/visible integrations and links only to existing reachable integration areas.

- `apps/dashboard/src/components/dashboard/operations-activity.tsx`
  Shows recent order activity based on existing recent order summaries.

- `apps/dashboard/src/components/dashboard/__tests__/operations-dashboard-components.test.tsx`
  Component smoke tests for the new dashboard sections.

Modify:

- `apps/dashboard/src/app/(dashboard)/page.tsx`
  Replace sales dashboard layout with operations cockpit layout.

- `apps/dashboard/src/lib/nav-items.ts`
  Adjust only naming/navigation labels needed by the cockpit, without adding new routes.

- `apps/dashboard/messages/pl/common.json`
  Polish navigation and dashboard copy.

- `apps/dashboard/messages/en/common.json`
  English fallback copy.

- `apps/dashboard/messages/pl/dashboard.json`
  Dashboard cockpit section labels.

- `apps/dashboard/messages/en/dashboard.json`
  English fallback labels.

- `public/.claude/context/PROJECT_STATE.md`
  Record the dashboard direction after implementation.

Do not modify:

- API server in this PR.
- Public Helm chart.
- Enterprise repo.

---

### Task 1: Add Operations Dashboard View Model

**Files:**

- Create: `apps/dashboard/src/lib/operations-dashboard.ts`
- Create: `apps/dashboard/src/lib/__tests__/operations-dashboard.test.ts`

- [ ] **Step 1: Write failing tests for stage calculation**

Create `apps/dashboard/src/lib/__tests__/operations-dashboard.test.ts` with these cases:

```ts
import {
  buildOrchestrationStages,
  buildOperationalExceptions,
  buildIntegrationHealth,
} from "@/lib/operations-dashboard";
import type { DashboardStats, Integration, ListResponse, Order, Shipment } from "@/types/api";

function stats(overrides: Partial<DashboardStats> = {}): DashboardStats {
  return {
    order_counts: {
      total: 18,
      by_status: {
        new: 3,
        confirmed: 2,
        processing: 4,
        ready_to_ship: 3,
        shipped: 2,
        in_transit: 1,
        delivered: 2,
        completed: 1,
        on_hold: 2,
      },
      by_source: { allegro: 12, manual: 6 },
    },
    revenue: { total: 0, currency: "PLN", daily: [] },
    recent_orders: [],
    ...overrides,
  };
}

describe("buildOrchestrationStages", () => {
  it("groups existing order statuses into a compact operational pipeline", () => {
    const stages = buildOrchestrationStages(stats());

    expect(stages.map((stage) => stage.key)).toEqual([
      "intake",
      "fulfillment",
      "shipping",
      "completed",
    ]);
    expect(stages.find((stage) => stage.key === "intake")?.count).toBe(5);
    expect(stages.find((stage) => stage.key === "fulfillment")?.count).toBe(7);
    expect(stages.find((stage) => stage.key === "shipping")?.count).toBe(3);
    expect(stages.find((stage) => stage.key === "completed")?.count).toBe(3);
  });

  it("marks the pipeline as warning when on-hold orders exist", () => {
    const stages = buildOrchestrationStages(stats());

    expect(stages.find((stage) => stage.key === "fulfillment")?.health).toBe("warning");
  });
});
```

- [ ] **Step 2: Write failing tests for no-fake-action exceptions**

Append:

```ts
function listResponse<T>(items: T[]): ListResponse<T> {
  return { items, total: items.length, limit: items.length, offset: 0 };
}

const baseOrder = {
  id: "order-1",
  tenant_id: "tenant-1",
  source: "allegro",
  status: "on_hold",
  customer_name: "Jan Kowalski",
  total_amount: 100,
  currency: "PLN",
  payment_status: "paid",
  tags: [],
  created_at: "2026-05-11T10:00:00Z",
  updated_at: "2026-05-11T10:00:00Z",
} satisfies Order;

const baseShipment = {
  id: "shipment-1",
  tenant_id: "tenant-1",
  order_id: "order-1",
  provider: "inpost",
  status: "failed",
  package_number: 1,
  notes: "",
  created_at: "2026-05-11T10:00:00Z",
  updated_at: "2026-05-11T10:00:00Z",
} satisfies Shipment;

describe("buildOperationalExceptions", () => {
  it("returns only existing-route links and no speculative action labels", () => {
    const exceptions = buildOperationalExceptions({
      onHoldOrders: listResponse([baseOrder]),
      failedShipments: listResponse([baseShipment]),
      integrations: [],
      limit: 7,
    });

    expect(exceptions).toHaveLength(2);
    expect(exceptions[0]?.primaryHref).toMatch(/^\\/(orders|shipments)/);
    expect(exceptions.every((item) => item.primaryActionLabel === undefined)).toBe(true);
  });

  it("limits dashboard exceptions to prevent clutter", () => {
    const orders = Array.from({ length: 10 }, (_, index) => ({
      ...baseOrder,
      id: `order-${index}`,
    }));

    const exceptions = buildOperationalExceptions({
      onHoldOrders: listResponse(orders),
      failedShipments: listResponse([]),
      integrations: [],
      limit: 7,
    });

    expect(exceptions).toHaveLength(7);
  });
});
```

- [ ] **Step 3: Write failing tests for integration visibility**

Append:

```ts
function integration(overrides: Partial<Integration>): Integration {
  return {
    id: "integration-1",
    tenant_id: "tenant-1",
    provider: "allegro",
    status: "active",
    has_credentials: true,
    created_at: "2026-05-11T10:00:00Z",
    updated_at: "2026-05-11T10:00:00Z",
    ...overrides,
  };
}

describe("buildIntegrationHealth", () => {
  it("shows ready providers and error providers, but hides blocked providers", () => {
    const health = buildIntegrationHealth([
      integration({ id: "allegro", provider: "allegro", status: "active" }),
      integration({ id: "inpost", provider: "inpost", status: "active" }),
      integration({ id: "shopify", provider: "shopify", status: "inactive" }),
    ]);

    expect(health.map((item) => item.provider)).toEqual(["allegro", "inpost"]);
  });

  it("keeps an error integration visible when the user must fix it", () => {
    const health = buildIntegrationHealth([
      integration({
        id: "olx-error",
        provider: "olx",
        status: "error",
        error_message: "Reconnect required",
      }),
    ]);

    expect(health).toHaveLength(1);
    expect(health[0]?.health).toBe("problem");
  });
});
```

- [ ] **Step 4: Run tests and verify RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/operations-dashboard.test.ts --reporter=dot
```

Expected: fail because `@/lib/operations-dashboard` does not exist.

- [ ] **Step 5: Implement view-model functions**

Create `apps/dashboard/src/lib/operations-dashboard.ts`:

```ts
import type { DashboardStats, Integration, ListResponse, Order, Shipment } from "@/types/api";
import { getProviderReadiness, isReadinessVisible } from "@/lib/readiness";
import { PROVIDER_CATEGORIES } from "@/lib/constants";

export type OperationsHealth = "ok" | "warning" | "problem";

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
  title: string;
  description: string;
  primaryHref: string;
  primaryActionLabel?: never;
  createdAt?: string;
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
  title: string;
  description: string;
  href: string;
  createdAt: string;
  source: string;
}

interface BuildOperationalExceptionsInput {
  onHoldOrders?: ListResponse<Order>;
  failedShipments?: ListResponse<Shipment>;
  integrations?: Integration[];
  limit?: number;
}

const count = (stats: DashboardStats | undefined, statuses: string[]) =>
  statuses.reduce((sum, status) => sum + (stats?.order_counts.by_status[status] ?? 0), 0);

export function buildOrchestrationStages(stats?: DashboardStats): OrchestrationStage[] {
  const onHoldCount = count(stats, ["on_hold"]);
  const failedLikeCount = count(stats, ["cancelled", "refunded"]);

  return [
    {
      key: "intake",
      labelKey: "operations.stages.intake",
      count: count(stats, ["new", "confirmed"]),
      exceptionCount: 0,
      health: "ok",
      href: "/orders",
    },
    {
      key: "fulfillment",
      labelKey: "operations.stages.fulfillment",
      count: count(stats, ["processing", "ready_to_ship"]),
      exceptionCount: onHoldCount,
      health: onHoldCount > 0 ? "warning" : "ok",
      href: "/orders",
    },
    {
      key: "shipping",
      labelKey: "operations.stages.shipping",
      count: count(stats, ["shipped", "in_transit", "out_for_delivery"]),
      exceptionCount: 0,
      health: "ok",
      href: "/shipments",
    },
    {
      key: "completed",
      labelKey: "operations.stages.completed",
      count: count(stats, ["delivered", "completed"]),
      exceptionCount: failedLikeCount,
      health: failedLikeCount > 0 ? "warning" : "ok",
      href: "/orders",
    },
  ];
}

export function buildOperationalExceptions({
  onHoldOrders,
  failedShipments,
  integrations = [],
  limit = 7,
}: BuildOperationalExceptionsInput): OperationalException[] {
  const integrationExceptions = integrations
    .filter((integration) => integration.status === "error")
    .map((integration) => ({
      id: `integration-${integration.id}`,
      kind: "integration" as const,
      severity: "problem" as const,
      title: `${providerLabel(integration.provider)} wymaga uwagi`,
      description: integration.error_message || "Integracja ma status bledu.",
      primaryHref: integrationHref(integration.provider),
      createdAt: integration.updated_at,
    }));

  const shipmentExceptions = (failedShipments?.items ?? []).map((shipment) => ({
    id: `shipment-${shipment.id}`,
    kind: "shipment" as const,
    severity: "problem" as const,
    title: `Przesylka ${providerLabel(shipment.provider)} nie przeszla`,
    description: "Sprawdz szczegoly przesylki i dane przewoznika.",
    primaryHref: "/shipments",
    createdAt: shipment.updated_at,
  }));

  const orderExceptions = (onHoldOrders?.items ?? []).map((order) => ({
    id: `order-${order.id}`,
    kind: "order" as const,
    severity: "warning" as const,
    title: `Zamowienie wstrzymane: ${order.customer_name}`,
    description: "Zamowienie wymaga decyzji operatora.",
    primaryHref: `/orders/${order.id}`,
    createdAt: order.updated_at,
  }));

  return [...integrationExceptions, ...shipmentExceptions, ...orderExceptions]
    .sort((a, b) => Date.parse(b.createdAt ?? "") - Date.parse(a.createdAt ?? ""))
    .slice(0, limit);
}

export function buildIntegrationHealth(integrations: Integration[] = []): IntegrationHealthItem[] {
  return integrations
    .filter((integration) => {
      const readiness = getProviderReadiness(integration.provider);
      return integration.status === "error" || isReadinessVisible(readiness);
    })
    .map((integration) => ({
      id: integration.id,
      provider: integration.provider,
      label: integration.label || providerLabel(integration.provider),
      health: integration.status === "error" ? "problem" : integration.status === "active" ? "ok" : "warning",
      status: integration.status,
      href: integrationHref(integration.provider),
      errorMessage: integration.error_message,
      lastSyncAt: integration.last_sync_at,
    }));
}

export function buildOperationsActivity(stats?: DashboardStats): OperationsActivityItem[] {
  return (stats?.recent_orders ?? []).slice(0, 5).map((order) => ({
    id: order.id,
    title: `Zamowienie ${order.customer_name}`,
    description: `${sourceLabel(order.source)} -> ${order.status}`,
    href: `/orders/${order.id}`,
    createdAt: order.created_at,
    source: order.source,
  }));
}

function integrationHref(provider: string): string {
  if (PROVIDER_CATEGORIES.carrier.providers.includes(provider)) return "/carriers";
  if (PROVIDER_CATEGORIES.marketplace.providers.includes(provider)) return "/marketplaces";
  return "/settings/company";
}

function providerLabel(provider: string): string {
  const labels: Record<string, string> = {
    allegro: "Allegro",
    inpost: "InPost",
    fakturownia: "Fakturownia",
    olx: "OLX",
  };
  return labels[provider] ?? provider;
}

function sourceLabel(source: string): string {
  const labels: Record<string, string> = {
    allegro: "Allegro",
    manual: "Manual",
  };
  return labels[source] ?? source;
}
```

- [ ] **Step 6: Run tests and verify GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/operations-dashboard.test.ts --reporter=dot
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/dashboard/src/lib/operations-dashboard.ts apps/dashboard/src/lib/__tests__/operations-dashboard.test.ts
git commit -m "OPE-242: add operations dashboard view model"
```

---

### Task 2: Compose Existing Data With a Dashboard Hook

**Files:**

- Create: `apps/dashboard/src/hooks/use-operations-dashboard.ts`
- Modify: `apps/dashboard/src/types/api.ts`

- [ ] **Step 1: Add `order_ids` support to shipment params**

Modify `ShipmentListParams` in `apps/dashboard/src/types/api.ts`:

```ts
export interface ShipmentListParams extends PaginationParams {
  status?: string;
  provider?: string;
  order_id?: string;
  order_ids?: string;
}
```

- [ ] **Step 2: Create the hook**

Create `apps/dashboard/src/hooks/use-operations-dashboard.ts`:

```ts
"use client";

import { useMemo } from "react";
import { useAuthStore } from "@/lib/auth";
import { useDashboardStats } from "@/hooks/use-dashboard-stats";
import { useIntegrations } from "@/hooks/use-integrations";
import { useOrders } from "@/hooks/use-orders";
import { useShipments } from "@/hooks/use-shipments";
import {
  buildIntegrationHealth,
  buildOperationalExceptions,
  buildOperationsActivity,
  buildOrchestrationStages,
} from "@/lib/operations-dashboard";

export function useOperationsDashboard() {
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";

  const statsQuery = useDashboardStats();
  const integrationsQuery = useIntegrations();
  const onHoldOrdersQuery = useOrders({ status: "on_hold", limit: 7, offset: 0, sort_by: "updated_at", sort_order: "desc" });
  const failedShipmentsQuery = useShipments({ status: "failed", limit: 7, offset: 0, sort_by: "created_at", sort_order: "desc" });

  const model = useMemo(() => {
    const integrations = isAdmin ? integrationsQuery.data ?? [] : [];
    return {
      stages: buildOrchestrationStages(statsQuery.data),
      exceptions: buildOperationalExceptions({
        onHoldOrders: onHoldOrdersQuery.data,
        failedShipments: failedShipmentsQuery.data,
        integrations,
        limit: 7,
      }),
      integrationHealth: isAdmin ? buildIntegrationHealth(integrations) : [],
      activity: buildOperationsActivity(statsQuery.data),
    };
  }, [
    failedShipmentsQuery.data,
    integrationsQuery.data,
    isAdmin,
    onHoldOrdersQuery.data,
    statsQuery.data,
  ]);

  return {
    ...model,
    isLoading:
      statsQuery.isLoading ||
      onHoldOrdersQuery.isLoading ||
      failedShipmentsQuery.isLoading ||
      (isAdmin && integrationsQuery.isLoading),
    isError:
      statsQuery.isError ||
      onHoldOrdersQuery.isError ||
      failedShipmentsQuery.isError ||
      (isAdmin && integrationsQuery.isError),
    refetch: () => {
      void statsQuery.refetch();
      void onHoldOrdersQuery.refetch();
      void failedShipmentsQuery.refetch();
      if (isAdmin) void integrationsQuery.refetch();
    },
  };
}
```

- [ ] **Step 3: Run TypeScript check for touched files**

Run:

```bash
cd apps/dashboard
npx tsc --noEmit --pretty false
```

Expected: no TypeScript errors related to the new hook or `ShipmentListParams`.

- [ ] **Step 4: Commit**

```bash
git add apps/dashboard/src/hooks/use-operations-dashboard.ts apps/dashboard/src/types/api.ts
git commit -m "OPE-242: compose operations dashboard data"
```

---

### Task 3: Build Dashboard Components

**Files:**

- Create: `apps/dashboard/src/components/dashboard/orchestration-map.tsx`
- Create: `apps/dashboard/src/components/dashboard/operational-exceptions.tsx`
- Create: `apps/dashboard/src/components/dashboard/integration-health-panel.tsx`
- Create: `apps/dashboard/src/components/dashboard/operations-activity.tsx`
- Create: `apps/dashboard/src/components/dashboard/__tests__/operations-dashboard-components.test.tsx`

Implementation amendment from Task 1 review:

- `OperationalException` items expose `titleKey`, `descriptionKey`, and `values`.
- `OperationsActivityItem` items expose `titleKey`, `descriptionKey`, and `values`.
- Components must render translated copy with `t(item.titleKey, item.values)` and `t(item.descriptionKey, item.values)`.
- Do not read stale `item.title` or `item.description` fields.

- [ ] **Step 1: Create `OrchestrationMap`**

Create `apps/dashboard/src/components/dashboard/orchestration-map.tsx`:

```tsx
"use client";

import Link from "next/link";
import { ArrowRight, CheckCircle2, CircleAlert, CircleDot } from "lucide-react";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { OrchestrationStage } from "@/lib/operations-dashboard";

interface OrchestrationMapProps {
  stages: OrchestrationStage[];
  isLoading: boolean;
}

export function OrchestrationMap({ stages, isLoading }: OrchestrationMapProps) {
  const t = useTranslations("dashboard");

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("operations.flowTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="grid gap-3 md:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-24 w-full" />
            ))}
          </div>
        ) : (
          <div className="grid gap-3 md:grid-cols-4">
            {stages.map((stage, index) => (
              <Link
                key={stage.key}
                href={stage.href}
                className="group rounded-md border bg-card p-4 transition-colors hover:bg-muted/40"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-sm font-medium">{t(stage.labelKey)}</p>
                    <p className="mt-2 text-2xl font-semibold tabular-nums">{stage.count}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {stage.exceptionCount > 0
                        ? t("operations.exceptionsCount", { count: stage.exceptionCount })
                        : t("operations.noExceptions")}
                    </p>
                  </div>
                  <StageIcon health={stage.health} />
                </div>
                {index < stages.length - 1 && (
                  <ArrowRight className="mt-3 h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
                )}
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function StageIcon({ health }: { health: OrchestrationStage["health"] }) {
  const className = cn(
    "h-5 w-5",
    health === "ok" && "text-emerald-600",
    health === "warning" && "text-amber-600",
    health === "problem" && "text-red-600",
  );

  if (health === "ok") return <CheckCircle2 className={className} />;
  if (health === "warning") return <CircleAlert className={className} />;
  return <CircleDot className={className} />;
}
```

- [ ] **Step 2: Create `OperationalExceptions`**

Create `apps/dashboard/src/components/dashboard/operational-exceptions.tsx`:

```tsx
"use client";

import Link from "next/link";
import { AlertTriangle, CheckCircle2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { OperationalException } from "@/lib/operations-dashboard";

interface OperationalExceptionsProps {
  exceptions: OperationalException[];
  isLoading: boolean;
}

export function OperationalExceptions({ exceptions, isLoading }: OperationalExceptionsProps) {
  const t = useTranslations("dashboard");

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("operations.exceptionsTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-14 w-full" />
            ))}
          </div>
        ) : exceptions.length === 0 ? (
          <div className="flex min-h-32 flex-col items-center justify-center rounded-md border border-dashed text-center">
            <CheckCircle2 className="h-8 w-8 text-emerald-600" />
            <p className="mt-3 text-sm font-medium">{t("operations.noWorkTitle")}</p>
            <p className="mt-1 text-sm text-muted-foreground">{t("operations.noWorkDescription")}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {exceptions.map((item) => (
              <Link
                key={item.id}
                href={item.primaryHref}
                className="flex gap-3 rounded-md border p-3 transition-colors hover:bg-muted/40"
              >
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium">{item.title}</span>
                  <span className="block truncate text-xs text-muted-foreground">{item.description}</span>
                </span>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 3: Create `IntegrationHealthPanel`**

Create `apps/dashboard/src/components/dashboard/integration-health-panel.tsx`:

```tsx
"use client";

import Link from "next/link";
import { CircleAlert, CircleCheck, CircleMinus } from "lucide-react";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import type { IntegrationHealthItem } from "@/lib/operations-dashboard";

interface IntegrationHealthPanelProps {
  items: IntegrationHealthItem[];
  isLoading: boolean;
}

export function IntegrationHealthPanel({ items, isLoading }: IntegrationHealthPanelProps) {
  const t = useTranslations("dashboard");

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("operations.integrationHealthTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, index) => (
              <Skeleton key={index} className="h-11 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("operations.noIntegrationHealth")}</p>
        ) : (
          <div className="space-y-2">
            {items.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                className="flex items-center justify-between gap-3 rounded-md border px-3 py-2 transition-colors hover:bg-muted/40"
              >
                <span className="flex min-w-0 items-center gap-2">
                  <HealthIcon health={item.health} />
                  <span className="truncate text-sm font-medium">{item.label}</span>
                </span>
                <Badge variant={item.health === "problem" ? "destructive" : item.health === "warning" ? "secondary" : "success"}>
                  {t(`operations.health.${item.health}`)}
                </Badge>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function HealthIcon({ health }: { health: IntegrationHealthItem["health"] }) {
  if (health === "ok") return <CircleCheck className="h-4 w-4 text-emerald-600" />;
  if (health === "warning") return <CircleMinus className="h-4 w-4 text-amber-600" />;
  return <CircleAlert className="h-4 w-4 text-red-600" />;
}
```

- [ ] **Step 4: Create `OperationsActivity`**

Create `apps/dashboard/src/components/dashboard/operations-activity.tsx`:

```tsx
"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { useLocale, useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { OperationsActivityItem } from "@/lib/operations-dashboard";

interface OperationsActivityProps {
  items: OperationsActivityItem[];
  isLoading: boolean;
}

export function OperationsActivity({ items, isLoading }: OperationsActivityProps) {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const dateLocale = locale === "pl" ? pl : enUS;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("operations.activityTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className="h-10 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("operations.noActivity")}</p>
        ) : (
          <div className="space-y-1">
            {items.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                className="grid grid-cols-[1fr_auto] gap-3 rounded-md px-2 py-2 transition-colors hover:bg-muted/40"
              >
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium">{item.title}</span>
                  <span className="block truncate text-xs text-muted-foreground">{item.description}</span>
                </span>
                <span className="whitespace-nowrap text-xs text-muted-foreground">
                  {formatDistanceToNow(new Date(item.createdAt), { addSuffix: true, locale: dateLocale })}
                </span>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 5: Add component smoke tests**

Create `apps/dashboard/src/components/dashboard/__tests__/operations-dashboard-components.test.tsx`:

```tsx
import { render, screen } from "@/test/utils";
import { OrchestrationMap } from "@/components/dashboard/orchestration-map";
import { OperationalExceptions } from "@/components/dashboard/operational-exceptions";

describe("operations dashboard components", () => {
  it("renders orchestration stages as links to existing routes", () => {
    render(
      <OrchestrationMap
        isLoading={false}
        stages={[
          { key: "intake", labelKey: "operations.stages.intake", count: 3, exceptionCount: 0, health: "ok", href: "/orders" },
          { key: "fulfillment", labelKey: "operations.stages.fulfillment", count: 2, exceptionCount: 1, health: "warning", href: "/orders" },
          { key: "shipping", labelKey: "operations.stages.shipping", count: 1, exceptionCount: 0, health: "ok", href: "/shipments" },
          { key: "completed", labelKey: "operations.stages.completed", count: 4, exceptionCount: 0, health: "ok", href: "/orders" },
        ]}
      />,
    );

    expect(screen.getByText("Przyjecie")).toBeInTheDocument();
    expect(screen.getAllByRole("link").map((link) => link.getAttribute("href"))).toEqual([
      "/orders",
      "/orders",
      "/shipments",
      "/orders",
    ]);
  });

  it("does not render unavailable action buttons for exceptions", () => {
    render(
      <OperationalExceptions
        isLoading={false}
        exceptions={[
          {
            id: "order-1",
            kind: "order",
            severity: "warning",
            title: "Zamowienie wstrzymane",
            description: "Wymaga decyzji operatora.",
            primaryHref: "/orders/order-1",
          },
        ]}
      />,
    );

    expect(screen.getByRole("link", { name: /zamowienie wstrzymane/i })).toHaveAttribute("href", "/orders/order-1");
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 6: Run component tests**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/dashboard/__tests__/operations-dashboard-components.test.tsx --reporter=dot
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/dashboard/src/components/dashboard/orchestration-map.tsx apps/dashboard/src/components/dashboard/operational-exceptions.tsx apps/dashboard/src/components/dashboard/integration-health-panel.tsx apps/dashboard/src/components/dashboard/operations-activity.tsx apps/dashboard/src/components/dashboard/__tests__/operations-dashboard-components.test.tsx
git commit -m "OPE-242: build operations dashboard sections"
```

---

### Task 4: Replace Dashboard Home

**Files:**

- Modify: `apps/dashboard/src/app/(dashboard)/page.tsx`
- Modify: `apps/dashboard/messages/pl/dashboard.json`
- Modify: `apps/dashboard/messages/en/dashboard.json`
- Modify: `apps/dashboard/messages/pl/common.json`
- Modify: `apps/dashboard/messages/en/common.json`

- [ ] **Step 1: Add dashboard translations**

Add these keys under `dashboard` in both Polish and English message files.

Polish values:

```json
{
  "operations": {
    "title": "Centrum orkiestracji",
    "subtitle": "Stan przeplywu zamowien przez kanaly, realizacje i wysylke.",
    "flowTitle": "Przeplyw operacyjny",
    "exceptionsTitle": "Do obsluzenia teraz",
    "integrationHealthTitle": "Stan polaczen",
    "activityTitle": "Ostatnia aktywnosc",
    "noExceptions": "Bez wyjatkow",
    "exceptionsCount": "{count} wyj.",
    "noWorkTitle": "Brak pilnych spraw",
    "noWorkDescription": "Najwazniejsze przeplywy wygladaja zdrowo.",
    "noIntegrationHealth": "Brak widocznych polaczen do sprawdzenia.",
    "noActivity": "Brak ostatniej aktywnosci.",
    "stages": {
      "intake": "Przyjecie",
      "fulfillment": "Realizacja",
      "shipping": "Wysylka",
      "completed": "Zakonczone"
    },
    "health": {
      "ok": "OK",
      "warning": "Uwaga",
      "problem": "Problem"
    }
  }
}
```

English values:

```json
{
  "operations": {
    "title": "Orchestration center",
    "subtitle": "Order-flow health across channels, fulfillment, and shipping.",
    "flowTitle": "Operational flow",
    "exceptionsTitle": "Needs attention",
    "integrationHealthTitle": "Connection health",
    "activityTitle": "Recent activity",
    "noExceptions": "No exceptions",
    "exceptionsCount": "{count} exc.",
    "noWorkTitle": "No urgent work",
    "noWorkDescription": "The most important flows look healthy.",
    "noIntegrationHealth": "No visible connections to check.",
    "noActivity": "No recent activity.",
    "stages": {
      "intake": "Intake",
      "fulfillment": "Fulfillment",
      "shipping": "Shipping",
      "completed": "Completed"
    },
    "health": {
      "ok": "OK",
      "warning": "Warning",
      "problem": "Problem"
    }
  }
}
```

- [ ] **Step 2: Replace dashboard page composition**

Modify `apps/dashboard/src/app/(dashboard)/page.tsx` so it uses the new hook and components. Keep `OnboardingWizard` and `QuickStartCard`; remove revenue/status/source charts from the home dashboard.

Expected structure:

```tsx
export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const { stages, exceptions, integrationHealth, activity, isLoading, isError, refetch } = useOperationsDashboard();
  const user = useAuthStore((s) => s.user);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal">{t("operations.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("operations.subtitle")}</p>
        </div>
        {user?.name && (
          <p className="text-sm text-muted-foreground">Operator: {user.name}</p>
        )}
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{t("loadError")}</p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
            {t("retry")}
          </Button>
        </div>
      )}

      <OnboardingWizard />
      <QuickStartCard />

      <OrchestrationMap stages={stages} isLoading={isLoading} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.8fr)]">
        <OperationalExceptions exceptions={exceptions} isLoading={isLoading} />
        <IntegrationHealthPanel items={integrationHealth} isLoading={isLoading} />
      </div>

      <OperationsActivity items={activity} isLoading={isLoading} />
    </div>
  );
}
```

- [ ] **Step 3: Remove unused imports**

Remove dashboard imports that are no longer used:

```ts
import { useDashboardStats } from "@/hooks/use-dashboard-stats";
import { StatCards } from "@/components/dashboard/stat-cards";
import { RevenueChart } from "@/components/dashboard/revenue-chart";
import { OrderStatusChart } from "@/components/dashboard/order-status-chart";
import { OrderSourceChart } from "@/components/dashboard/order-source-chart";
import { RecentOrdersTable } from "@/components/dashboard/recent-orders-table";
```

Add imports:

```ts
import { useOperationsDashboard } from "@/hooks/use-operations-dashboard";
import { OrchestrationMap } from "@/components/dashboard/orchestration-map";
import { OperationalExceptions } from "@/components/dashboard/operational-exceptions";
import { IntegrationHealthPanel } from "@/components/dashboard/integration-health-panel";
import { OperationsActivity } from "@/components/dashboard/operations-activity";
```

- [ ] **Step 4: Run dashboard tests and lint**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/operations-dashboard.test.ts src/components/dashboard/__tests__/operations-dashboard-components.test.tsx --reporter=dot
npx eslint --quiet src/app/\\(dashboard\\)/page.tsx src/components/dashboard src/hooks/use-operations-dashboard.ts src/lib/operations-dashboard.ts
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add apps/dashboard/src/app/\\(dashboard\\)/page.tsx apps/dashboard/messages/pl/dashboard.json apps/dashboard/messages/en/dashboard.json apps/dashboard/messages/pl/common.json apps/dashboard/messages/en/common.json
git commit -m "OPE-242: replace home with operations cockpit"
```

---

### Task 5: Navigation Naming Pass

**Files:**

- Modify: `apps/dashboard/src/lib/nav-items.ts`
- Modify: `apps/dashboard/messages/pl/common.json`
- Modify: `apps/dashboard/messages/en/common.json`
- Test: existing nav/readiness tests if present, otherwise targeted eslint/typecheck.

- [ ] **Step 1: Rename only labels, not routes**

Keep existing routes. Do not introduce unavailable destinations.

Recommended label changes:

```ts
{ href: "/", label: "operationsDashboard", icon: LayoutDashboard },
{ href: "/marketplaces", label: "salesChannelConnections", icon: Store, adminOnly: true, group: "salesChannels" },
```

Add translations:

```json
{
  "navigation": {
    "operationsDashboard": "Pulpit operacyjny",
    "salesChannelConnections": "Polaczenia"
  }
}
```

English:

```json
{
  "navigation": {
    "operationsDashboard": "Operations dashboard",
    "salesChannelConnections": "Connections"
  }
}
```

- [ ] **Step 2: Keep client-ready filtering unchanged**

Do not change readiness statuses in `apps/dashboard/src/lib/readiness.ts` in this task.

- [ ] **Step 3: Run targeted lint**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/lib/nav-items.ts
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add apps/dashboard/src/lib/nav-items.ts apps/dashboard/messages/pl/common.json apps/dashboard/messages/en/common.json
git commit -m "OPE-242: clarify operations navigation labels"
```

---

### Task 6: Visual QA and Documentation

**Files:**

- Modify: `public/.claude/context/PROJECT_STATE.md`
- Modify: optional screenshot or notes under `docs/superpowers/specs/`

- [ ] **Step 1: Run local validation**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/operations-dashboard.test.ts src/components/dashboard/__tests__/operations-dashboard-components.test.tsx --reporter=dot
npx eslint --quiet src/app/\\(dashboard\\)/page.tsx src/components/dashboard src/hooks/use-operations-dashboard.ts src/lib/operations-dashboard.ts src/lib/nav-items.ts
```

Expected: pass.

- [ ] **Step 2: Run full public validation before push**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: pass.

- [ ] **Step 3: Browser pass**

Start the dashboard locally if needed:

```bash
cd apps/dashboard
npm run dev
```

Check:

- Desktop `/` shows no fake buttons in exceptions.
- Desktop `/` does not show revenue chart as primary dashboard content.
- Desktop `/` uses compact operation stages and no visual clutter.
- Mobile `/` has no horizontal scroll.
- Links from stages go only to existing routes: `/orders`, `/shipments`.
- Integration health links go only to existing routes: `/marketplaces`, `/carriers`, or `/settings/company`.

- [ ] **Step 4: Update project state**

Append a factual note to `public/.claude/context/PROJECT_STATE.md`:

```md
- 2026-05-11: OPE-242 — dashboard home shifted from sales-style reporting to an operations control tower. The new surface uses existing data only: compact order-flow stages, a limited exception queue, reachable ready-provider integration health, and recent order activity. It intentionally avoids unavailable action buttons and hidden-provider repair links.
```

- [ ] **Step 5: Commit**

```bash
git add public/.claude/context/PROJECT_STATE.md
git commit -m "OPE-242: document operations dashboard rollout"
```

---

## Follow-up Issues To Create After PR

Create separate Linear issues instead of expanding this PR:

- Backend `operations summary` endpoint if dashboard data becomes too chatty or needs automation/webhook counts.
- Real retry actions for failed shipments, webhook deliveries, or delayed automation actions.
- Dedicated operational event stream after backend source-of-truth is defined.
- Full shell visual system pass for tables/forms after `OPE-242` lands.

## Self-Review

- Spec coverage: The plan implements operations control tower, compact map, limited exception queue, integration health, no fake interactions, and navigation copy.
- Placeholder scan: No placeholder markers or unspecified action buttons are part of the implementation.
- Type consistency: View-model types are defined in `operations-dashboard.ts` and consumed by hook/components.
- Scope check: Backend aggregation and real retry actions are split into follow-up issues to avoid showing unavailable interactions.
