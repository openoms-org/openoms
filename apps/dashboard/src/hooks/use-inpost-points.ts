"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { InPostPointSearchResponse } from "@/types/api";

export function useInPostPointSearch(query: string) {
  return useQuery({
    queryKey: ["inpost-points", query],
    queryFn: () =>
      apiClient<InPostPointSearchResponse>(
        `/v1/inpost/points?query=${encodeURIComponent(query)}`
      ),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    placeholderData: (previousData: InPostPointSearchResponse | undefined) =>
      previousData,
  });
}
