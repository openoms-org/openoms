import { useQuery, useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  WorkflowTemplate,
  WorkflowDefinition,
  ValidateWorkflowResponse,
  ConvertWorkflowResponse,
} from "@/types/api";

export function useWorkflowTemplates() {
  return useQuery({
    queryKey: ["workflow-templates"],
    queryFn: () => apiClient<WorkflowTemplate[]>("/v1/workflows/templates"),
  });
}

export function useValidateWorkflow() {
  return useMutation({
    mutationFn: (definition: WorkflowDefinition) =>
      apiClient<ValidateWorkflowResponse>("/v1/workflows/validate", {
        method: "POST",
        body: JSON.stringify({ definition }),
      }),
  });
}

export function useConvertWorkflow() {
  return useMutation({
    mutationFn: (data: { definition: WorkflowDefinition; name: string; enabled?: boolean; priority?: number }) =>
      apiClient<ConvertWorkflowResponse>("/v1/workflows/convert", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}

export function useWorkflowForRule(ruleId: string) {
  return useQuery({
    queryKey: ["workflow-for-rule", ruleId],
    queryFn: () => apiClient<WorkflowDefinition>(`/v1/workflows/rules/${ruleId}/workflow`),
    enabled: !!ruleId,
  });
}
