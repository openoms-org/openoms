"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/lib/auth";
import {
  useFulfillmentExceptions,
  useOperationsSummary,
} from "@/hooks/use-fulfillment";
import { OperationsBucketSummary } from "@/components/dashboard/operations-bucket-summary";
import { FulfillmentExceptionsFeed } from "@/components/dashboard/fulfillment-exceptions-feed";
import { FulfillmentDetailPanel } from "@/components/fulfillment/fulfillment-detail-panel";
import { Button } from "@/components/ui/button";
import { exceptionPopulationFromBuckets } from "@/lib/fulfillment-exceptions";
import type { FulfillmentExceptionItem } from "@/types/fulfillment";

const PREVIEW_EXCEPTION_LIMIT = 3;

/**
 * Additive home-page panel surfacing the process-backed control tower next to
 * the existing heuristic dashboard. While FULFILLMENT_PROCESS_ENABLED is off the
 * summary is all-zero and the exceptions feed shows the intentional empty state.
 * Only operators (admin/owner) load it — matching the existing dashboard guard.
 */
export function FulfillmentControlTowerPanel() {
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";
  const t = useTranslations("dashboard");

  const summary = useOperationsSummary({ enabled: isAdmin });
  const exceptions = useFulfillmentExceptions(PREVIEW_EXCEPTION_LIMIT, {
    enabled: isAdmin,
  });

  const [selected, setSelected] = useState<FulfillmentExceptionItem | null>(null);

  if (!isAdmin) {
    return null;
  }

  return (
    <section className="space-y-3" data-testid="fulfillment-control-tower-panel">
      <div className="flex items-center justify-between gap-3 px-1">
        <div>
          <h2 className="text-lg font-semibold">
            {t("fulfillment.controlTower.title")}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t("fulfillment.controlTower.subtitle")}
          </p>
        </div>
        <Button variant="outline" size="sm" asChild>
          <Link href="/operations/fulfillment">
            {t("fulfillment.controlTower.viewAll")}
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </Link>
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_400px]">
        <OperationsBucketSummary
          buckets={summary.data?.buckets}
          total={summary.data?.total}
          isLoading={summary.isLoading}
          isError={summary.isError}
          onRetry={() => summary.refetch()}
        />
        <FulfillmentExceptionsFeed
          items={exceptions.data?.items}
          isLoading={exceptions.isLoading}
          isError={exceptions.isError}
          onRetry={() => exceptions.refetch()}
          onSelect={setSelected}
          maxItems={PREVIEW_EXCEPTION_LIMIT}
          totalCount={exceptionPopulationFromBuckets(summary.data?.buckets)}
        />
      </div>

      <FulfillmentDetailPanel
        processId={selected?.process.id}
        open={!!selected}
        onOpenChange={(o) => !o && setSelected(null)}
      />
    </section>
  );
}
