"use client";

import { formatDistanceToNow } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { CheckCircle2, ChevronRight } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  BucketBadge,
  blockerReasonLabel,
} from "@/components/fulfillment/fulfillment-status-badge";
import type { FulfillmentExceptionItem } from "@/types/fulfillment";

export interface FulfillmentExceptionsFeedProps {
  items?: FulfillmentExceptionItem[];
  isLoading: boolean;
  isError?: boolean;
  onRetry?: () => void;
  /** Drilldown — receives the selected exception's process. */
  onSelect: (item: FulfillmentExceptionItem) => void;
  /** Cap rendered items (e.g. a home-page preview). */
  maxItems?: number;
  /**
   * True size of the exception population (derived from the operations
   * summary). When the list is capped, the badge reads "shown of total"
   * instead of presenting the capped preview count as the total; when
   * undefined (summary unavailable) the badge is qualified as a preview.
   */
  totalCount?: number;
  titleKey?: string;
}

export function FulfillmentExceptionsFeed({
  items,
  isLoading,
  isError,
  onRetry,
  onSelect,
  maxItems,
  totalCount,
  titleKey = "fulfillment.exceptions.title",
}: FulfillmentExceptionsFeedProps) {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const dateLocale = locale === "pl" ? pl : enUS;

  const isCapped = typeof maxItems === "number";
  const visible = isCapped ? (items ?? []).slice(0, maxItems) : items ?? [];
  const shownCount = visible.length;

  // Capped previews must not present the preview size as the total (OPE-529):
  // qualify it against the true population, or mark it as a preview when the
  // population is unknown.
  let badgeText: string;
  if (!isCapped) {
    badgeText = String(items?.length ?? 0);
  } else if (typeof totalCount === "number") {
    badgeText =
      totalCount > shownCount
        ? t("fulfillment.exceptions.shownOfTotal", {
            shown: shownCount,
            total: totalCount,
          })
        : String(shownCount);
  } else {
    badgeText =
      shownCount > 0
        ? t("fulfillment.exceptions.topPreview", { count: shownCount })
        : "0";
  }
  const hasExceptions = shownCount > 0 || (totalCount ?? 0) > 0;

  return (
    <Card className="h-full overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-4">
        <div className="flex items-start justify-between gap-4">
          <CardTitle className="text-base">{t(titleKey)}</CardTitle>
          {!isLoading && !isError && (
            <Badge variant={hasExceptions ? "warning" : "success"}>
              {badgeText}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {isError ? (
          <div className="flex min-h-48 flex-col items-center justify-center gap-2 px-8 text-center">
            <p className="text-sm text-destructive">
              {t("fulfillment.exceptions.loadError")}
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
          <div className="space-y-3 p-5">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-20 w-full rounded-lg" />
            ))}
          </div>
        ) : visible.length === 0 ? (
          <div className="flex min-h-48 flex-col items-center justify-center px-8 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-lg bg-success/10">
              <CheckCircle2 className="h-6 w-6 text-success" aria-hidden="true" />
            </span>
            <p className="mt-3 font-medium">
              {t("fulfillment.exceptions.emptyTitle")}
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("fulfillment.exceptions.emptyDescription")}
            </p>
          </div>
        ) : (
          <ul className="divide-y">
            {visible.map((item) => {
              const blocker = item.top_blocker;
              const reason = blocker
                ? blockerReasonLabel(t, blocker.code)
                : t("fulfillment.exceptions.noBlocker");

              return (
                <li key={item.process.id}>
                  <button
                    type="button"
                    onClick={() => onSelect(item)}
                    className="flex w-full items-start gap-3 bg-card p-4 text-left transition-colors hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    data-testid="fulfillment-exception-item"
                    data-process-id={item.process.id}
                    data-order-id={item.process.order_id}
                  >
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center justify-between gap-3">
                        <BucketBadge bucket={item.bucket} />
                        <span className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(item.process.updated_at), {
                            addSuffix: true,
                            locale: dateLocale,
                          })}
                        </span>
                      </span>
                      <span className="mt-2 block text-sm font-medium">
                        {reason}
                      </span>
                      {blocker?.description && (
                        <span className="mt-1 block text-sm text-muted-foreground">
                          {blocker.description}
                        </span>
                      )}
                      <span className="mt-1 block text-xs text-muted-foreground">
                        {t("fulfillment.exceptions.orderRef", {
                          orderId: item.process.order_id,
                        })}
                      </span>
                    </span>
                    <ChevronRight
                      className="mt-1 h-4 w-4 shrink-0 text-muted-foreground"
                      aria-hidden="true"
                    />
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
