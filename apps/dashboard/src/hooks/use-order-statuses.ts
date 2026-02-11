import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { OrderStatusConfig } from "@/types/api";

export const COLOR_PRESETS: Record<string, string> = {
  blue: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  indigo: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200",
  yellow: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  orange: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
  purple: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
  teal: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200",
  green: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  "green-dark": "bg-green-200 text-green-900 dark:bg-green-900 dark:text-green-200",
  gray: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
  red: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
  "red-dark": "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200",
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
