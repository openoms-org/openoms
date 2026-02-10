"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { EmailSettings, CompanySettings } from "@/types/api";

export function useEmailSettings() {
  return useQuery({
    queryKey: ["settings", "email"],
    queryFn: () => apiClient<EmailSettings>("/v1/settings/email"),
  });
}

export function useUpdateEmailSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: EmailSettings) =>
      apiClient<EmailSettings>("/v1/settings/email", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "email"] });
    },
  });
}

export function useSendTestEmail() {
  return useMutation({
    mutationFn: (toEmail: string) =>
      apiClient<{ message: string }>("/v1/settings/email/test", {
        method: "POST",
        body: JSON.stringify({ to_email: toEmail }),
      }),
  });
}

export function useCompanySettings() {
  return useQuery({
    queryKey: ["settings", "company"],
    queryFn: () => apiClient<CompanySettings>("/v1/settings/company"),
  });
}

export function useUpdateCompanySettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CompanySettings) =>
      apiClient<CompanySettings>("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "company"] });
    },
  });
}
