"use client";

import type { LucideIcon } from "lucide-react";
import {
  AlertOctagon,
  CircleSlash,
  Hourglass,
  HelpCircle,
  PackageCheck,
  PlayCircle,
} from "lucide-react";
import { useTranslations } from "next-intl";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { OPERATOR_BUCKETS, type OperatorBucket } from "@/types/fulfillment";

interface OperationsBucketSummaryProps {
  buckets?: Record<OperatorBucket, number>;
  total?: number;
  isLoading: boolean;
  isError?: boolean;
  onRetry?: () => void;
  /** Optional click handler — receives the bucket key for filtering/drilldown. */
  onSelectBucket?: (bucket: OperatorBucket) => void;
}

const BUCKET_ICONS: Record<OperatorBucket, LucideIcon> = {
  ready: PackageCheck,
  processing: PlayCircle,
  stuck: AlertOctagon,
  blocked: CircleSlash,
  provider_issue: Hourglass,
  missing_data: HelpCircle,
};

// Tone classes for the accent. "ready"/"processing" are healthy; the rest are
// attention buckets.
const BUCKET_TONE: Record<OperatorBucket, string> = {
  ready: "text-info",
  processing: "text-info",
  stuck: "text-destructive",
  blocked: "text-destructive",
  provider_issue: "text-warning",
  missing_data: "text-warning",
};

export function OperationsBucketSummary({
  buckets,
  total,
  isLoading,
  isError,
  onRetry,
  onSelectBucket,
}: OperationsBucketSummaryProps) {
  const t = useTranslations("dashboard");

  return (
    <Card className="overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-3">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="text-base">
            {t("fulfillment.controlTower.title")}
          </CardTitle>
          {!isLoading && !isError && typeof total === "number" && (
            <span className="text-xs text-muted-foreground">
              {t("fulfillment.controlTower.totalProcesses", { count: total })}
            </span>
          )}
        </div>
      </CardHeader>
      <CardContent className="p-4">
        {isError ? (
          <div className="flex min-h-32 flex-col items-center justify-center gap-2 text-center">
            <p className="text-sm text-destructive">
              {t("fulfillment.controlTower.loadError")}
            </p>
            {onRetry && (
              <button
                type="button"
                onClick={onRetry}
                className="text-sm font-medium text-info underline-offset-2 hover:underline"
              >
                {t("fulfillment.retry")}
              </button>
            )}
          </div>
        ) : isLoading ? (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-24 w-full rounded-lg" />
            ))}
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {OPERATOR_BUCKETS.map((bucket) => {
              const Icon = BUCKET_ICONS[bucket];
              const count = buckets?.[bucket] ?? 0;
              const interactive = !!onSelectBucket;

              const inner = (
                <>
                  <span className="flex items-start justify-between gap-2">
                    <span className="text-sm font-medium text-muted-foreground">
                      {t(`fulfillment.buckets.${bucket}.label`)}
                    </span>
                    <Icon
                      className={cn("h-4 w-4 shrink-0", BUCKET_TONE[bucket])}
                      aria-hidden="true"
                    />
                  </span>
                  <span className="mt-2 block text-3xl font-semibold tabular-nums">
                    {count}
                  </span>
                  <span className="mt-1 block text-xs text-muted-foreground">
                    {t(`fulfillment.buckets.${bucket}.hint`)}
                  </span>
                </>
              );

              const baseClass =
                "flex flex-col rounded-lg border bg-background p-4 text-left shadow-[0_18px_50px_-38px_rgba(15,23,42,0.55)]";

              if (interactive) {
                return (
                  <button
                    key={bucket}
                    type="button"
                    onClick={() => onSelectBucket(bucket)}
                    className={cn(
                      baseClass,
                      "transition-colors hover:border-primary/40 hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                    )}
                  >
                    {inner}
                  </button>
                );
              }

              return (
                <div key={bucket} className={baseClass}>
                  {inner}
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
