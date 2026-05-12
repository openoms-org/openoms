"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type StatusTone = "ok" | "warning" | "problem" | "inactive" | "draft" | "pending" | "blocked";

const toneClasses: Record<StatusTone, string> = {
  ok: "bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200",
  warning: "bg-amber-50 text-amber-800 ring-1 ring-amber-200",
  problem: "bg-red-50 text-red-800 ring-1 ring-red-200",
  inactive: "bg-slate-100 text-slate-700 ring-1 ring-slate-200",
  draft: "bg-slate-100 text-slate-700 ring-1 ring-slate-200",
  pending: "bg-blue-50 text-blue-800 ring-1 ring-blue-200",
  blocked: "bg-red-50 text-red-800 ring-1 ring-red-200",
};

interface StatusBadgeProps {
  status: string;
  statusMap?: Record<string, { label: string; color: string }>;
  /** Translation prefix under "statuses" namespace, e.g. "order", "shipment", "return" */
  translationPrefix?: string;
  label?: string;
  tone?: StatusTone;
}

export function StatusBadge({
  status,
  statusMap,
  translationPrefix,
  label: explicitLabel,
  tone,
}: StatusBadgeProps) {
  const t = useTranslations("statuses");
  const config = statusMap?.[status];

  let label = explicitLabel ?? config?.label ?? status;
  if (!explicitLabel && translationPrefix) {
    try {
      label = t(`${translationPrefix}.${status}`);
    } catch {
      label = config?.label ?? status;
    }
  }

  if (tone) {
    return (
      <span
        className={cn(
          "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
          toneClasses[tone]
        )}
      >
        {label}
      </span>
    );
  }

  if (config) {
    return (
      <span
        className={cn(
          "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
          config.color
        )}
      >
        {label}
      </span>
    );
  }

  return <Badge variant="outline">{label}</Badge>;
}
