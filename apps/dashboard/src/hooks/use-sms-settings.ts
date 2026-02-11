"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { SMSSettings } from "@/types/api";

export function useSMSSettings() {
  return useQuery({
    queryKey: ["settings", "sms"],
    queryFn: () => apiClient<SMSSettings>("/v1/settings/sms"),
  });
}

export function useUpdateSMSSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: SMSSettings) =>
      apiClient<SMSSettings>("/v1/settings/sms", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "sms"] });
    },
  });
}

export function useSendTestSMS() {
  return useMutation({
    mutationFn: (phone: string) =>
      apiClient<{ message: string }>("/v1/settings/sms/test", {
        method: "POST",
        body: JSON.stringify({ phone }),
      }),
  });
}
