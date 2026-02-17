"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import type { CarbonStats } from "@/types/api";

export function useCarbonStats(dateFrom?: string, dateTo?: string) {
  const params = new URLSearchParams();
  if (dateFrom) params.set("date_from", dateFrom);
  if (dateTo) params.set("date_to", dateTo);
  const qs = params.toString();

  return useQuery({
    queryKey: ["carbon", "stats", dateFrom, dateTo],
    queryFn: () =>
      apiClient<CarbonStats>(`/v1/carbon/stats${qs ? `?${qs}` : ""}`),
  });
}

export function useDownloadCarbonReport() {
  return async (dateFrom?: string, dateTo?: string) => {
    const params = new URLSearchParams();
    if (dateFrom) params.set("date_from", dateFrom);
    if (dateTo) params.set("date_to", dateTo);
    const qs = params.toString();

    const res = await apiFetch(`/v1/carbon/report${qs ? `?${qs}` : ""}`);
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "carbon-report.csv";
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  };
}
