import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import type {
  OnboardingStatus,
  UpdateOnboardingStepRequest,
} from "@/types/api";

export function useOnboardingStatus() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["onboarding", "status"],
    queryFn: () => apiClient<OnboardingStatus>("/v1/onboarding/status"),
    staleTime: 5 * 60 * 1000,
    enabled: isAuthenticated,
  });
}

export function useUpdateOnboardingStep() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ step, action }: { step: number; action: "completed" | "skipped" }) => {
      const body: UpdateOnboardingStepRequest = { action };
      return apiClient<{ message: string }>(`/v1/onboarding/step/${step}`, {
        method: "PUT",
        body: JSON.stringify(body),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["onboarding"] });
    },
  });
}

export function useCompleteOnboarding() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ message: string }>("/v1/onboarding/complete", {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["onboarding"] });
    },
  });
}

export function useDismissOnboarding() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<OnboardingStatus>("/v1/settings/onboarding", {
        method: "PUT",
        body: JSON.stringify({ dismissed: true }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["onboarding"] });
    },
  });
}
