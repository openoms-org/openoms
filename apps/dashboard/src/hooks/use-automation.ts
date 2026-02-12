import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  AutomationRule,
  AutomationRuleLog,
  DelayedAction,
  ListResponse,
  AutomationRuleListParams,
  CreateAutomationRuleRequest,
  UpdateAutomationRuleRequest,
  TestAutomationRuleRequest,
  TestAutomationRuleResponse,
} from "@/types/api";

export function useAutomationRules(params: AutomationRuleListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.trigger_event) query.set("trigger_event", params.trigger_event);
  if (params.enabled != null) query.set("enabled", String(params.enabled));
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["automation-rules", params],
    queryFn: () =>
      apiClient<ListResponse<AutomationRule>>(
        `/v1/automation/rules${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useAutomationRule(id: string) {
  return useQuery({
    queryKey: ["automation-rules", id],
    queryFn: () => apiClient<AutomationRule>(`/v1/automation/rules/${id}`),
    enabled: !!id,
  });
}

export function useAutomationRuleLogs(ruleId: string) {
  return useQuery({
    queryKey: ["automation-rules", ruleId, "logs"],
    queryFn: () =>
      apiClient<ListResponse<AutomationRuleLog>>(
        `/v1/automation/rules/${ruleId}/logs`
      ),
    enabled: !!ruleId,
  });
}

export function useCreateAutomationRule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateAutomationRuleRequest) =>
      apiClient<AutomationRule>("/v1/automation/rules", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["automation-rules"] });
    },
  });
}

export function useUpdateAutomationRule(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateAutomationRuleRequest) =>
      apiClient<AutomationRule>(`/v1/automation/rules/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["automation-rules"] });
    },
  });
}

export function useDeleteAutomationRule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/automation/rules/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["automation-rules"] });
    },
  });
}

export function useTestAutomationRule(id: string) {
  return useMutation({
    mutationFn: (data: TestAutomationRuleRequest) =>
      apiClient<TestAutomationRuleResponse>(
        `/v1/automation/rules/${id}/test`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
  });
}

export function useDelayedActions() {
  return useQuery({
    queryKey: ["automation-delayed"],
    queryFn: () => apiClient<DelayedAction[]>("/v1/automation/delayed"),
  });
}
