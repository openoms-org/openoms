"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, apiFetch } from "@/lib/api-client";
import { downloadBlob } from "@/lib/download";
import type {
  VATRateSet,
  VATCalculation,
  VATCalculateRequest,
  OSSConfig,
  OSSReport,
  ThresholdStatus,
} from "@/types/api";

export function useVATRates() {
  return useQuery({
    queryKey: ["vat-oss", "rates"],
    queryFn: () =>
      apiClient<Record<string, VATRateSet>>("/v1/vat-oss/rates"),
  });
}

export function useVATCountryRates(country: string) {
  return useQuery({
    queryKey: ["vat-oss", "rates", country],
    queryFn: () =>
      apiClient<VATRateSet>(`/v1/vat-oss/rates/${country}`),
    enabled: !!country,
  });
}

export function useCalculateVAT() {
  return useMutation({
    mutationFn: (data: VATCalculateRequest) =>
      apiClient<VATCalculation>("/v1/vat-oss/calculate", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}

export function useOSSConfig() {
  return useQuery({
    queryKey: ["vat-oss", "config"],
    queryFn: () => apiClient<OSSConfig>("/v1/vat-oss/config"),
  });
}

export function useUpdateOSSConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: OSSConfig) =>
      apiClient<OSSConfig>("/v1/vat-oss/config", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vat-oss", "config"] });
    },
  });
}

export function useOSSReport(quarter: number, year: number) {
  return useQuery({
    queryKey: ["vat-oss", "report", quarter, year],
    queryFn: () =>
      apiClient<OSSReport>(
        `/v1/vat-oss/report?quarter=${quarter}&year=${year}`
      ),
  });
}

export function useOSSThreshold(year: number) {
  return useQuery({
    queryKey: ["vat-oss", "threshold", year],
    queryFn: () =>
      apiClient<ThresholdStatus>(`/v1/vat-oss/threshold?year=${year}`),
  });
}

export function useDownloadOSSReportCSV() {
  return async (quarter: number, year: number) => {
    const res = await apiFetch(
      `/v1/vat-oss/report/csv?quarter=${quarter}&year=${year}`
    );
    const blob = await res.blob();
    downloadBlob(blob, `vat-oss-report-Q${quarter}-${year}.csv`);
  };
}
