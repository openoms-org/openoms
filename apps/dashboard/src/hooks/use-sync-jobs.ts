"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { SyncJob } from "@/types/api";

export function useSyncJobs(integrationId: string) {
  return useQuery({
    queryKey: ["sync-jobs", integrationId],
    queryFn: () =>
      apiClient<SyncJob[]>(
        `/v1/sync-jobs?integration_id=${encodeURIComponent(integrationId)}`
      ),
    enabled: !!integrationId,
  });
}
