"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type { ListResponse } from "@/types/api";
import type {
  FulfillmentBlocker,
  FulfillmentExceptionsResponse,
  FulfillmentProcess,
  FulfillmentProcessDetail,
  FulfillmentProcessListParams,
  FulfillmentStep,
  IntegrationCapabilitySummary,
  OperationsSummary,
} from "@/types/fulfillment";

/**
 * React Query hooks for the OPE-419 operations control tower + fulfillment
 * read/action API (#540). All endpoints are tenant-scoped (RLS) and return
 * empty/zero results while no fulfillment data is recorded — every consumer
 * must render an intentional empty state, not an error.
 */

export const fulfillmentKeys = {
  all: ["fulfillment"] as const,
  processes: (params: FulfillmentProcessListParams) =>
    ["fulfillment", "processes", params] as const,
  process: (id: string) => ["fulfillment", "process", id] as const,
  order: (orderId: string) => ["fulfillment", "order", orderId] as const,
  summary: () => ["operations", "summary"] as const,
  exceptions: (limit?: number) => ["operations", "exceptions", limit ?? 0] as const,
  capability: () => ["operations", "integration-capability-summary"] as const,
};

// === Reads ===

export function useFulfillmentProcesses(
  params: FulfillmentProcessListParams = {},
  options: { enabled?: boolean } = {},
) {
  const qs = buildSearchParams(params).toString();
  return useQuery({
    queryKey: fulfillmentKeys.processes(params),
    queryFn: () =>
      apiClient<ListResponse<FulfillmentProcess>>(
        `/v1/fulfillment/processes${qs ? `?${qs}` : ""}`,
      ),
    ...options,
  });
}

export function useFulfillmentProcess(
  id: string | undefined,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: fulfillmentKeys.process(id ?? ""),
    queryFn: () =>
      apiClient<FulfillmentProcessDetail>(`/v1/fulfillment/processes/${id}`),
    enabled: !!id && (options.enabled ?? true),
  });
}

export function useOrderFulfillment(
  orderId: string | undefined,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: fulfillmentKeys.order(orderId ?? ""),
    queryFn: () =>
      apiClient<FulfillmentProcessDetail>(`/v1/fulfillment/orders/${orderId}`),
    enabled: !!orderId && (options.enabled ?? true),
  });
}

export function useOperationsSummary(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: fulfillmentKeys.summary(),
    queryFn: () => apiClient<OperationsSummary>("/v1/operations/summary"),
    ...options,
  });
}

export function useFulfillmentExceptions(
  limit?: number,
  options: { enabled?: boolean } = {},
) {
  const qs = buildSearchParams(limit ? { limit } : {}).toString();
  return useQuery({
    queryKey: fulfillmentKeys.exceptions(limit),
    queryFn: () =>
      apiClient<FulfillmentExceptionsResponse>(
        `/v1/operations/exceptions${qs ? `?${qs}` : ""}`,
      ),
    ...options,
  });
}

export function useIntegrationCapabilitySummary(
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: fulfillmentKeys.capability(),
    queryFn: () =>
      apiClient<IntegrationCapabilitySummary>(
        "/v1/operations/integration-capability-summary",
      ),
    ...options,
  });
}

// === Actions ===

/**
 * Resolve a fulfillment blocker. On success invalidates every operations/
 * fulfillment query so summary, exceptions and any open detail panel refetch.
 */
export function useResolveBlocker() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (blockerId: string) =>
      apiClient<FulfillmentBlocker>(
        `/v1/fulfillment/blockers/${blockerId}/resolve`,
        { method: "POST" },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fulfillmentKeys.all });
      queryClient.invalidateQueries({ queryKey: ["operations"] });
    },
  });
}

/**
 * Retry a failed/blocked fulfillment step. Returns 400 when the step is not in
 * a retryable state — callers surface that as a toast.
 */
export function useRetryStep() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (stepId: string) =>
      apiClient<FulfillmentStep>(`/v1/fulfillment/steps/${stepId}/retry`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fulfillmentKeys.all });
      queryClient.invalidateQueries({ queryKey: ["operations"] });
    },
  });
}
