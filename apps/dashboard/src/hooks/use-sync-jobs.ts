"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { SyncJob, SyncJobListParams, ListResponse } from "@/types/api";

export function useSyncJobs(params: SyncJobListParams) {
  return useQuery({
    queryKey: ["sync-jobs", params],
    queryFn: () => {
      const query = new URLSearchParams();
      if (params.limit != null) query.set("limit", params.limit.toString());
      if (params.offset != null) query.set("offset", params.offset.toString());
      if (params.integration_id) query.set("integration_id", params.integration_id);
      if (params.job_type) query.set("job_type", params.job_type);
      if (params.status) query.set("status", params.status);
      return apiClient<ListResponse<SyncJob>>(`/v1/sync-jobs?${query.toString()}`);
    },
  });
}

export function useSyncJob(id: string) {
  return useQuery({
    queryKey: ["sync-job", id],
    queryFn: () => apiClient<SyncJob>(`/v1/sync-jobs/${encodeURIComponent(id)}`),
    enabled: !!id,
  });
}
