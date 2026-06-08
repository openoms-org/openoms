"use client";

import { useState } from "react";
import { Activity, AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslations } from "next-intl";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { OperationsBucketSummary } from "@/components/dashboard/operations-bucket-summary";
import { FulfillmentExceptionsFeed } from "@/components/dashboard/fulfillment-exceptions-feed";
import { ParityReadinessIndicator } from "@/components/dashboard/parity-readiness-indicator";
import { FulfillmentDetailPanel } from "@/components/fulfillment/fulfillment-detail-panel";
import {
  useFulfillmentExceptions,
  useIntegrationCapabilitySummary,
  useOperationsSummary,
} from "@/hooks/use-fulfillment";
import type { FulfillmentExceptionItem } from "@/types/fulfillment";

/**
 * Dedicated process-backed operations control tower (OPE-419). This is the full
 * surface complementing the additive home-page panel: the 6 operator buckets,
 * integration automation-coverage metrics, and the complete actionable
 * exceptions feed with drilldown into the per-process fulfillment detail.
 *
 * While FULFILLMENT_PROCESS_ENABLED is off the backend returns empty/zero, so
 * every section renders an intentional empty/zero state rather than an error.
 */
export default function FulfillmentOperationsPage() {
  const t = useTranslations("dashboard");

  const summary = useOperationsSummary();
  const capability = useIntegrationCapabilitySummary();
  const exceptions = useFulfillmentExceptions();

  const [selected, setSelected] = useState<FulfillmentExceptionItem | null>(null);

  return (
    <AdminGuard>
      <div className="-m-6 min-h-full bg-muted/30 p-4 sm:p-6 lg:p-7">
        <div className="mx-auto max-w-[1600px] space-y-5">
          <div className="px-1">
            <h1 className="text-2xl font-semibold tracking-normal md:text-3xl">
              {t("fulfillment.controlTower.pageTitle")}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground md:text-base">
              {t("fulfillment.controlTower.pageSubtitle")}
            </p>
          </div>

          <OperationsBucketSummary
            buckets={summary.data?.buckets}
            total={summary.data?.total}
            isLoading={summary.isLoading}
            isError={summary.isError}
            onRetry={() => summary.refetch()}
          />

          <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_400px]">
            <CapabilitySummaryCard
              isLoading={capability.isLoading}
              isError={capability.isError}
              onRetry={() => capability.refetch()}
              data={capability.data}
            />
            <FulfillmentExceptionsFeed
              items={exceptions.data?.items}
              isLoading={exceptions.isLoading}
              isError={exceptions.isError}
              onRetry={() => exceptions.refetch()}
              onSelect={setSelected}
            />
          </div>

          {/* OPE-423c: cutover parity-readiness indicator. Informational only —
              shows whether the process-backed model has reached parity with the
              legacy heuristic so an operator can judge when it is safe to flip
              the NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD flag to process-backed. */}
          <ParityReadinessIndicator />

          <FulfillmentDetailPanel
            processId={selected?.process.id}
            open={!!selected}
            onOpenChange={(o) => !o && setSelected(null)}
          />
        </div>
      </div>
    </AdminGuard>
  );
}

interface CapabilitySummaryData {
  automation_coverage: number;
  manual_intervention_processes: number;
  unsupported_capability_processes: number;
  stale_data_processes: number;
  missing_mapping_processes: number;
  failed_provider_attempts: number;
  total_processes: number;
}

function CapabilitySummaryCard({
  isLoading,
  isError,
  onRetry,
  data,
}: {
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  data?: CapabilitySummaryData;
}) {
  const t = useTranslations("dashboard");

  const coveragePct =
    data && data.total_processes > 0
      ? Math.round(data.automation_coverage * 100)
      : 0;

  const metrics: { key: string; value: number; attention: boolean }[] = data
    ? [
        {
          key: "manualIntervention",
          value: data.manual_intervention_processes,
          attention: data.manual_intervention_processes > 0,
        },
        {
          key: "unsupportedCapability",
          value: data.unsupported_capability_processes,
          attention: data.unsupported_capability_processes > 0,
        },
        {
          key: "missingMapping",
          value: data.missing_mapping_processes,
          attention: data.missing_mapping_processes > 0,
        },
        {
          key: "staleData",
          value: data.stale_data_processes,
          attention: data.stale_data_processes > 0,
        },
        {
          key: "failedAttempts",
          value: data.failed_provider_attempts,
          attention: data.failed_provider_attempts > 0,
        },
      ]
    : [];

  return (
    <Card className="overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Activity className="h-4 w-4 text-info" aria-hidden="true" />
          {t("fulfillment.capability.title")}
        </CardTitle>
      </CardHeader>
      <CardContent className="p-4">
        {isError ? (
          <div className="flex min-h-32 flex-col items-center justify-center gap-2 text-center">
            <p className="text-sm text-destructive">
              {t("fulfillment.capability.loadError")}
            </p>
            <button
              type="button"
              onClick={onRetry}
              className="inline-flex items-center gap-1 text-sm font-medium text-info underline-offset-2 hover:underline"
            >
              <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
              {t("fulfillment.retry")}
            </button>
          </div>
        ) : isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-16 w-full rounded-lg" />
            <Skeleton className="h-24 w-full rounded-lg" />
          </div>
        ) : (
          <div className="space-y-4">
            <div className="rounded-lg border bg-background p-4">
              <p className="text-sm font-medium text-muted-foreground">
                {t("fulfillment.capability.automationCoverage")}
              </p>
              <p className="mt-1 text-3xl font-semibold tabular-nums">
                {coveragePct}%
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("fulfillment.capability.coverageHint", {
                  count: data?.total_processes ?? 0,
                })}
              </p>
            </div>

            <dl className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {metrics.map((metric) => (
                <div
                  key={metric.key}
                  className="flex items-center justify-between gap-3 rounded-md border bg-background px-3 py-2"
                >
                  <dt className="flex items-center gap-2 text-sm text-muted-foreground">
                    {metric.attention && (
                      <AlertTriangle
                        className="h-3.5 w-3.5 text-warning"
                        aria-hidden="true"
                      />
                    )}
                    {t(`fulfillment.capability.${metric.key}`)}
                  </dt>
                  <dd className="text-sm font-semibold tabular-nums">
                    {metric.value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
