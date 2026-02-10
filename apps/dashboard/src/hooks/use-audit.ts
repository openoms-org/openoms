"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { AuditLogEntry, AuditListParams, ListResponse } from "@/types/api";

export function useAuditLog(params: AuditListParams) {
  return useQuery({
    queryKey: ["audit", params],
    queryFn: () => {
      const query = new URLSearchParams();
      if (params.limit) query.set("limit", params.limit.toString());
      if (params.offset) query.set("offset", params.offset.toString());
      if (params.entity_type) query.set("entity_type", params.entity_type);
      if (params.action) query.set("action", params.action);
      if (params.user_id) query.set("user_id", params.user_id);
      return apiClient<ListResponse<AuditLogEntry>>(`/v1/audit?${query.toString()}`);
    },
  });
}
