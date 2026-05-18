"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import {
  buildIntegrationHealth,
  buildOperationalExceptions,
  buildOperationsActivity,
  buildOrchestrationStages,
} from "@/lib/operations-dashboard";
import type { Integration, OrderListParams, ShipmentListParams } from "@/types/api";
import { integrationQueryKeys } from "./integration-query-keys";
import { useDashboardStats } from "./use-dashboard-stats";
import { useOrders } from "./use-orders";
import { useShipments } from "./use-shipments";

const EXCEPTION_LIMIT = 7;

const ON_HOLD_ORDER_PARAMS = {
  status: "on_hold",
  limit: EXCEPTION_LIMIT,
  offset: 0,
  sort_by: "created_at",
  sort_order: "desc",
} satisfies OrderListParams;

const FAILED_SHIPMENT_PARAMS = {
  status: "failed",
  limit: EXCEPTION_LIMIT,
  offset: 0,
  sort_by: "created_at",
  sort_order: "desc",
} satisfies ShipmentListParams;

export function useOperationsDashboard() {
  const user = useAuthStore((state) => state.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";

  const statsQuery = useDashboardStats();
  const onHoldOrdersQuery = useOrders(ON_HOLD_ORDER_PARAMS, { enabled: isAdmin });
  const failedShipmentsQuery = useShipments(FAILED_SHIPMENT_PARAMS, {
    enabled: isAdmin,
  });
  const integrationsQuery = useQuery<Integration[]>({
    queryKey: integrationQueryKeys.all,
    queryFn: () => apiClient<Integration[]>("/v1/integrations"),
    enabled: isAdmin,
  });

  const model = useMemo(() => {
    const integrations = isAdmin ? integrationsQuery.data ?? [] : [];

    return {
      stages: buildOrchestrationStages(statsQuery.data),
      exceptions: buildOperationalExceptions({
        onHoldOrders: onHoldOrdersQuery.data,
        failedShipments: failedShipmentsQuery.data,
        integrations,
        limit: EXCEPTION_LIMIT,
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
      (isAdmin && onHoldOrdersQuery.isLoading) ||
      (isAdmin && failedShipmentsQuery.isLoading) ||
      (isAdmin && integrationsQuery.isLoading),
    isError:
      statsQuery.isError ||
      (isAdmin && onHoldOrdersQuery.isError) ||
      (isAdmin && failedShipmentsQuery.isError) ||
      (isAdmin && integrationsQuery.isError),
    refetch: async () => {
      const refetches: Promise<unknown>[] = [statsQuery.refetch()];

      if (isAdmin) {
        refetches.push(
          onHoldOrdersQuery.refetch(),
          failedShipmentsQuery.refetch(),
          integrationsQuery.refetch()
        );
      }

      await Promise.all(refetches);
    },
  };
}
