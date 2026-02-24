import { useQuery, useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
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

const automationHooks = createCrudHooks<
  AutomationRule,
  CreateAutomationRuleRequest,
  UpdateAutomationRuleRequest,
  AutomationRuleListParams
>({
  resourceKey: "automation-rules",
  basePath: "/v1/automation/rules",
});

export const useAutomationRules = automationHooks.useList;
export const useAutomationRule = automationHooks.useGet;
export const useCreateAutomationRule = automationHooks.useCreate;
export const useUpdateAutomationRule = automationHooks.useUpdate;
export const useDeleteAutomationRule = automationHooks.useDelete;

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
