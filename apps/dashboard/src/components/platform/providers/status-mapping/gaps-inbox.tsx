"use client";

import { AlertOctagon, CheckCircle2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ApiClientError } from "@/lib/api-client";
import {
  useProviderGaps,
  useUpdateProviderGap,
} from "@/hooks/use-platform-provider-config";
import type { GapSeverity, ProviderIntegrationGap } from "@/types/platform";

interface GapsInboxProps {
  providerId: string;
  versionId: string;
  readOnly: boolean;
}

const SEVERITY_TONE: Record<GapSeverity, "info" | "warning" | "destructive"> = {
  info: "info",
  warning: "warning",
  action_required: "destructive",
  system_error: "destructive",
};

/**
 * Unknown-gaps inbox shown at the top of the status workbench (design spec
 * §370). Surfaces open/acknowledged integration gaps — chiefly unmapped
 * statuses — so they are handled rather than silently mapped to success.
 */
export function GapsInbox({ providerId, versionId, readOnly }: GapsInboxProps) {
  const t = useTranslations("platform");
  const { data: gaps = [] } = useProviderGaps(providerId, versionId);
  const updateGap = useUpdateProviderGap(providerId, versionId);

  const openGaps = gaps.filter((g) => g.status !== "resolved");

  if (openGaps.length === 0) {
    return (
      <div className="flex items-center gap-2 rounded-md border border-success/40 bg-success/10 p-3 text-sm text-success">
        <CheckCircle2 className="h-4 w-4" />
        {t("statusMapping.gaps.none")}
      </div>
    );
  }

  const resolve = (gap: ProviderIntegrationGap) => {
    updateGap.mutate(
      { gapId: gap.id, status: "resolved" },
      {
        onSuccess: () => toast.success(t("statusMapping.gaps.resolved")),
        onError: (err) =>
          toast.error(err instanceof ApiClientError ? err.message : t("saveError")),
      },
    );
  };

  const acknowledge = (gap: ProviderIntegrationGap) => {
    updateGap.mutate(
      { gapId: gap.id, status: "acknowledged" },
      {
        onError: (err) =>
          toast.error(err instanceof ApiClientError ? err.message : t("saveError")),
      },
    );
  };

  return (
    <div className="space-y-2 rounded-md border border-destructive/40 bg-destructive/5 p-3">
      <div className="flex items-center gap-2 text-sm font-medium text-destructive">
        <AlertOctagon className="h-4 w-4" />
        {t("statusMapping.gaps.title", { count: openGaps.length })}
      </div>
      <ul className="space-y-2">
        {openGaps.map((gap) => (
          <li
            key={gap.id}
            className="flex flex-wrap items-center gap-2 rounded-md border bg-background p-2.5"
          >
            <Badge variant={SEVERITY_TONE[gap.severity]}>
              {t(`statusMapping.gaps.severity.${gap.severity}`)}
            </Badge>
            <Badge variant="outline">{t(`statusMapping.gaps.type.${gap.gap_type}`)}</Badge>
            <span className="min-w-0 flex-1 text-sm">
              {gap.description || t("statusMapping.gaps.noDescription")}
            </span>
            <Badge variant="secondary">
              {t(`statusMapping.gaps.status.${gap.status}`)}
            </Badge>
            {!readOnly && (
              <span className="flex gap-1">
                {gap.status === "open" && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => acknowledge(gap)}
                    disabled={updateGap.isPending}
                  >
                    {t("statusMapping.gaps.acknowledge")}
                  </Button>
                )}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => resolve(gap)}
                  disabled={updateGap.isPending}
                >
                  {t("statusMapping.gaps.resolve")}
                </Button>
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
