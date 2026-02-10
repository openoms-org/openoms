import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { OrderStatusConfig } from "@/types/api";

export const COLOR_PRESETS: Record<string, string> = {
  blue: "bg-blue-100 text-blue-800",
  indigo: "bg-indigo-100 text-indigo-800",
  yellow: "bg-yellow-100 text-yellow-800",
  orange: "bg-orange-100 text-orange-800",
  purple: "bg-purple-100 text-purple-800",
  teal: "bg-teal-100 text-teal-800",
  green: "bg-green-100 text-green-800",
  "green-dark": "bg-green-200 text-green-900",
  gray: "bg-gray-100 text-gray-800",
  red: "bg-red-100 text-red-800",
  "red-dark": "bg-red-200 text-red-900",
};

export function useOrderStatuses() {
  return useQuery({
    queryKey: ["order-statuses"],
    queryFn: () => apiClient<OrderStatusConfig>("/v1/order-statuses"),
    staleTime: 5 * 60 * 1000,
  });
}

export function statusesToMap(
  config: OrderStatusConfig
): Record<string, { label: string; color: string }> {
  const map: Record<string, { label: string; color: string }> = {};
  for (const s of config.statuses) {
    map[s.key] = {
      label: s.label,
      color: COLOR_PRESETS[s.color] || COLOR_PRESETS.gray,
    };
  }
  return map;
}
