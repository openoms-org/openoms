"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: string;
  statusMap: Record<string, { label: string; color: string }>;
  /** Translation prefix under "statuses" namespace, e.g. "order", "shipment", "return" */
  translationPrefix?: string;
}

export function StatusBadge({ status, statusMap, translationPrefix }: StatusBadgeProps) {
  const t = useTranslations("statuses");
  const config = statusMap[status];
  if (!config) {
    return <Badge variant="outline">{status}</Badge>;
  }

  let label: string;
  if (translationPrefix) {
    try {
      label = t(`${translationPrefix}.${status}`);
    } catch {
      label = config.label;
    }
  } else {
    // If the label is a translation key (same as status key), try to translate it
    // Otherwise fall back to the raw label
    label = config.label;
  }

  return (
    <span className={cn("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium", config.color)}>
      {label}
    </span>
  );
}
