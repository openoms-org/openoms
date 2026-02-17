import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  RepricingRule,
  RepricingLog,
  RepricingSummary,
  SimulationResult,
  ApplyResult,
  ListResponse,
  RepricingRuleListParams,
  CreateRepricingRuleRequest,
  UpdateRepricingRuleRequest,
} from "@/types/api";

export function useRepricingRules(params: RepricingRuleListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.strategy) query.set("strategy", params.strategy);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["repricing-rules", params],
    queryFn: () =>
      apiClient<ListResponse<RepricingRule>>(
        `/v1/repricing/rules${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useRepricingRule(id: string) {
  return useQuery({
    queryKey: ["repricing-rules", id],
    queryFn: () => apiClient<RepricingRule>(`/v1/repricing/rules/${id}`),
    enabled: !!id,
  });
}

export function useCreateRepricingRule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateRepricingRuleRequest) =>
      apiClient<RepricingRule>("/v1/repricing/rules", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repricing-rules"] });
      queryClient.invalidateQueries({ queryKey: ["repricing-summary"] });
    },
  });
}

export function useUpdateRepricingRule(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateRepricingRuleRequest) =>
      apiClient<RepricingRule>(`/v1/repricing/rules/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repricing-rules"] });
      queryClient.invalidateQueries({ queryKey: ["repricing-rules", id] });
      queryClient.invalidateQueries({ queryKey: ["repricing-summary"] });
    },
  });
}

export function useDeleteRepricingRule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/repricing/rules/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repricing-rules"] });
      queryClient.invalidateQueries({ queryKey: ["repricing-summary"] });
    },
  });
}

export function useSimulateRepricingRule(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      apiClient<SimulationResult[]>(`/v1/repricing/rules/${id}/simulate`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["repricing-simulation", id],
      });
    },
  });
}

export function useApplyRepricingRules() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      apiClient<ApplyResult>("/v1/repricing/apply", {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repricing-rules"] });
      queryClient.invalidateQueries({ queryKey: ["repricing-summary"] });
      queryClient.invalidateQueries({ queryKey: ["repricing-log"] });
      queryClient.invalidateQueries({ queryKey: ["products"] });
    },
  });
}

export function useRepricingLog(params: {
  limit?: number;
  offset?: number;
  rule_id?: string;
  product_id?: string;
} = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.rule_id) query.set("rule_id", params.rule_id);
  if (params.product_id) query.set("product_id", params.product_id);

  const qs = query.toString();

  return useQuery({
    queryKey: ["repricing-log", params],
    queryFn: () =>
      apiClient<ListResponse<RepricingLog>>(
        `/v1/repricing/log${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useRepricingSummary() {
  return useQuery({
    queryKey: ["repricing-summary"],
    queryFn: () => apiClient<RepricingSummary>("/v1/repricing/summary"),
  });
}
