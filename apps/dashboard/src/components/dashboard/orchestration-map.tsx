"use client";

import Link from "next/link";
import { AlertTriangle, CheckCircle2, CircleAlert } from "lucide-react";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { OperationsHealth, OrchestrationStage } from "@/lib/operations-dashboard";

interface OrchestrationMapProps {
  stages: OrchestrationStage[];
  isLoading: boolean;
}

const healthIconClassName: Record<OperationsHealth, string> = {
  ok: "text-success",
  warning: "text-warning",
  problem: "text-destructive",
};

export function OrchestrationMap({ stages, isLoading }: OrchestrationMapProps) {
  const t = useTranslations("dashboard");

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("operations.flowTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="grid gap-3 md:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-28 w-full" />
            ))}
          </div>
        ) : (
          <div className="grid gap-3 md:grid-cols-4">
            {stages.map((stage) => (
              <Link
                key={stage.key}
                href={stage.href}
                className="rounded-lg border bg-background p-4 transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <div className="flex items-start justify-between gap-3">
                  <p className="text-sm font-medium text-muted-foreground">
                    {t(stage.labelKey)}
                  </p>
                  <StageStatusIcon
                    health={stage.health}
                    className={cn("h-4 w-4 shrink-0", healthIconClassName[stage.health])}
                  />
                </div>
                <p className="mt-4 text-3xl font-bold">{stage.count}</p>
                <p className="mt-2 text-xs text-muted-foreground">
                  {stage.exceptionCount > 0
                    ? t("operations.exceptionCount", { count: stage.exceptionCount })
                    : t("operations.noExceptions")}
                </p>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function StageStatusIcon({
  health,
  className,
}: {
  health: OperationsHealth;
  className?: string;
}) {
  if (health === "problem") {
    return <CircleAlert aria-hidden="true" className={className} />;
  }

  if (health === "warning") {
    return <AlertTriangle aria-hidden="true" className={className} />;
  }

  return <CheckCircle2 aria-hidden="true" className={className} />;
}
