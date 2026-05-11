"use client";

import type { LucideIcon } from "lucide-react";
import { Activity, AlertTriangle, GitBranch, RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type {
  IntegrationHealthItem,
  OperationalException,
  OperationsHealth,
  OrchestrationStage,
} from "@/lib/operations-dashboard";

interface OperationsSummaryStripProps {
  stages: OrchestrationStage[];
  exceptions: OperationalException[];
  integrationHealth: IntegrationHealthItem[];
  isLoading: boolean;
}

type SummaryTone = OperationsHealth | "info";

interface SummaryMetric {
  key: string;
  label: string;
  value: string;
  hint: string;
  tone: SummaryTone;
  icon: LucideIcon;
}

const toneStyles: Record<SummaryTone, { shell: string; icon: string; value: string }> = {
  ok: {
    shell: "bg-success/10 text-success",
    icon: "text-success",
    value: "text-success",
  },
  warning: {
    shell: "bg-warning/15 text-warning",
    icon: "text-warning",
    value: "text-warning",
  },
  problem: {
    shell: "bg-destructive/10 text-destructive",
    icon: "text-destructive",
    value: "text-destructive",
  },
  info: {
    shell: "bg-info/10 text-info",
    icon: "text-info",
    value: "text-foreground",
  },
};

export function OperationsSummaryStrip({
  stages,
  exceptions,
  integrationHealth,
  isLoading,
}: OperationsSummaryStripProps) {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const numberLocale = locale === "pl" ? "pl-PL" : "en-US";
  const queueCount = stages
    .filter((stage) => stage.key !== "completed")
    .reduce((sum, stage) => sum + stage.count, 0);
  const activeConnections = integrationHealth.filter((item) => item.health === "ok").length;
  const flowHealth = getFlowHealth(stages, exceptions);

  const metrics: SummaryMetric[] = [
    {
      key: "queue",
      label: t("operations.summary.queue"),
      value: queueCount.toLocaleString(numberLocale),
      hint: t("operations.summary.queueHint"),
      tone: "info",
      icon: GitBranch,
    },
    {
      key: "exceptions",
      label: t("operations.summary.exceptions"),
      value: exceptions.length.toLocaleString(numberLocale),
      hint: t("operations.summary.exceptionsHint"),
      tone: exceptions.length > 0 ? "warning" : "ok",
      icon: AlertTriangle,
    },
    {
      key: "connections",
      label: t("operations.summary.connections"),
      value: `${activeConnections}/${integrationHealth.length}`,
      hint: t("operations.summary.connectionsHint"),
      tone: integrationHealth.some((item) => item.health === "problem") ? "problem" : "ok",
      icon: RefreshCw,
    },
    {
      key: "flow",
      label: t("operations.summary.flow"),
      value: t(`operations.health.${flowHealth}`),
      hint: t(
        flowHealth === "ok"
          ? "operations.summary.flowOk"
          : "operations.summary.flowNeedsAttention",
      ),
      tone: flowHealth,
      icon: Activity,
    },
  ];

  return (
    <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {metrics.map((metric) => {
        const Icon = metric.icon;
        const style = toneStyles[metric.tone];

        return (
          <Card
            key={metric.key}
            className="overflow-hidden rounded-lg border-border/80 bg-card/95 py-0 shadow-sm"
          >
            <CardContent className="flex min-h-24 items-center gap-4 p-4">
              {isLoading ? (
                <>
                  <Skeleton className="h-10 w-10 rounded-lg" />
                  <div className="min-w-0 flex-1 space-y-2">
                    <Skeleton className="h-4 w-28" />
                    <Skeleton className="h-7 w-20" />
                    <Skeleton className="h-3 w-24" />
                  </div>
                </>
              ) : (
                <>
                  <span
                    className={cn(
                      "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg",
                      style.shell,
                    )}
                  >
                    <Icon className={cn("h-4 w-4", style.icon)} aria-hidden="true" />
                  </span>
                  <span className="min-w-0">
                    <span className="block text-sm font-medium text-muted-foreground">
                      {metric.label}
                    </span>
                    <span className={cn("mt-1 block text-2xl font-semibold", style.value)}>
                      {metric.value}
                    </span>
                    <span className="mt-1 block text-xs text-muted-foreground">
                      {metric.hint}
                    </span>
                  </span>
                </>
              )}
            </CardContent>
          </Card>
        );
      })}
    </section>
  );
}

function getFlowHealth(
  stages: OrchestrationStage[],
  exceptions: OperationalException[],
): OperationsHealth {
  if (exceptions.some((item) => item.severity === "problem")) {
    return "problem";
  }

  if (exceptions.length > 0 || stages.some((stage) => stage.health === "warning")) {
    return "warning";
  }

  return "ok";
}
