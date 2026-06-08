"use client";

import {
  CircleCheck,
  CircleHelp,
  Info,
  RefreshCw,
  TriangleAlert,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/lib/auth";
import { useFulfillmentParity } from "@/hooks/use-fulfillment";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { FulfillmentParityReport } from "@/types/fulfillment";

/**
 * Cutover parity-readiness indicator (OPE-423c). Consumes the read-only
 * GET /v1/operations/parity report and tells an operator whether the
 * process-backed model has reached parity with the legacy heuristic, so the
 * operations dashboard can be safely cut over (the flag flip is operational, not
 * code — this component never blocks anything; it is purely informational).
 *
 * Operator-guarded (admin/owner) like the rest of the operations surface. The
 * /v1/operations/parity endpoint is always registered server-side and returns
 * vacuous full coverage (1.0, met) on a quiet/empty tenant, so the default
 * rendering is an intentional "nothing to compare yet" state, not an error.
 */
export function ParityReadinessIndicator() {
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";
  const t = useTranslations("dashboard");

  const parity = useFulfillmentParity({ enabled: isAdmin });

  if (!isAdmin) {
    return null;
  }

  return (
    <Card
      className="overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm"
      data-testid="parity-readiness-indicator"
    >
      <CardHeader className="border-b bg-background/70 px-5 py-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <CircleHelp className="h-4 w-4 text-info" aria-hidden="true" />
          {t("fulfillment.parity.title")}
        </CardTitle>
        <p className="text-sm text-muted-foreground">
          {t("fulfillment.parity.subtitle")}
        </p>
      </CardHeader>
      <CardContent className="p-4">
        {parity.isError ? (
          <div className="flex min-h-32 flex-col items-center justify-center gap-2 text-center">
            <p className="text-sm text-destructive">
              {t("fulfillment.parity.loadError")}
            </p>
            <button
              type="button"
              onClick={() => parity.refetch()}
              className="inline-flex items-center gap-1 text-sm font-medium text-info underline-offset-2 hover:underline"
            >
              <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
              {t("fulfillment.retry")}
            </button>
          </div>
        ) : parity.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-20 w-full rounded-lg" />
            <Skeleton className="h-24 w-full rounded-lg" />
          </div>
        ) : parity.data ? (
          <ParityReadinessBody report={parity.data} />
        ) : null}
      </CardContent>
    </Card>
  );
}

function ParityReadinessBody({ report }: { report: FulfillmentParityReport }) {
  const t = useTranslations("dashboard");

  const coveragePct = Math.round(report.process_coverage * 100);
  const thresholdPct = Math.round(report.coverage_threshold * 100);
  const met = report.process_coverage_met;

  // A genuinely empty/quiet tenant: no active orders to compare. Coverage is
  // vacuously met (1.0) but there is nothing meaningful to show, so render the
  // intentional empty state instead of a misleading "ready" verdict.
  if (report.non_terminal_orders === 0 && report.fulfillment_processes === 0) {
    return (
      <div className="flex min-h-32 flex-col items-center justify-center gap-1 text-center">
        <Info className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <p className="text-sm font-medium">
          {t("fulfillment.parity.emptyTitle")}
        </p>
        <p className="text-xs text-muted-foreground">
          {t("fulfillment.parity.emptyDescription")}
        </p>
      </div>
    );
  }

  const VerdictIcon = met ? CircleCheck : TriangleAlert;

  return (
    <div className="space-y-4">
      <div
        className={cn(
          "flex items-start gap-3 rounded-lg border p-3",
          met
            ? "border-success/30 bg-success/10"
            : "border-warning/40 bg-warning/15",
        )}
        data-testid="parity-verdict"
        data-met={met ? "true" : "false"}
      >
        <VerdictIcon
          className={cn(
            "mt-0.5 h-5 w-5 shrink-0",
            met ? "text-success" : "text-warning",
          )}
          aria-hidden="true"
        />
        <div>
          <p
            className={cn(
              "text-sm font-semibold",
              met ? "text-success" : "text-warning",
            )}
          >
            {met
              ? t("fulfillment.parity.ready")
              : t("fulfillment.parity.notReady")}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {met
              ? t("fulfillment.parity.readyHint")
              : t("fulfillment.parity.notReadyHint")}
          </p>
        </div>
      </div>

      <div className="rounded-lg border bg-background p-4">
        <div className="flex items-baseline justify-between gap-2">
          <p className="text-sm font-medium text-muted-foreground">
            {t("fulfillment.parity.coverageLabel")}
          </p>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex cursor-help items-center gap-1 text-xs text-muted-foreground">
                  <Info className="h-3.5 w-3.5" aria-hidden="true" />
                  {t("fulfillment.parity.thresholdHint", {
                    threshold: thresholdPct,
                  })}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                {t("fulfillment.parity.processCount", {
                  count: report.fulfillment_processes,
                })}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        <p className="mt-1 text-3xl font-semibold tabular-nums">
          {coveragePct}%
        </p>
        <Progress
          value={coveragePct}
          className="mt-3"
          aria-label={t("fulfillment.parity.coverageLabel")}
        />
      </div>

      <dl className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <ParityMetric
          label={t("fulfillment.parity.missingProcesses")}
          value={report.orders_missing_process}
          attention={report.orders_missing_process > 0}
          hint={
            report.orders_missing_process > 0
              ? t("fulfillment.parity.missingProcessesHint")
              : undefined
          }
        />
        <ParityMetric
          label={t("fulfillment.parity.nonTerminalOrders")}
          value={report.non_terminal_orders}
        />
        <ParityMetric
          label={t("fulfillment.parity.legacyProblemOrders")}
          value={report.legacy_problem_orders}
        />
        <ParityMetric
          label={t("fulfillment.parity.processBackedExceptions")}
          value={report.process_backed_exceptions}
        />
      </dl>
    </div>
  );
}

function ParityMetric({
  label,
  value,
  attention = false,
  hint,
}: {
  label: string;
  value: number;
  attention?: boolean;
  hint?: string;
}) {
  return (
    <div className="rounded-md border bg-background px-3 py-2">
      <div className="flex items-center justify-between gap-3">
        <dt className="flex items-center gap-2 text-sm text-muted-foreground">
          {attention && (
            <TriangleAlert
              className="h-3.5 w-3.5 text-warning"
              aria-hidden="true"
            />
          )}
          {label}
        </dt>
        <dd className="text-sm font-semibold tabular-nums">{value}</dd>
      </div>
      {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}
