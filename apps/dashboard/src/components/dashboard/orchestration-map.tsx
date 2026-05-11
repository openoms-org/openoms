"use client";

import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  CircleAlert,
  ClipboardList,
  PackageCheck,
  ShieldCheck,
  Truck,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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

const healthBadgeVariant: Record<OperationsHealth, "success" | "warning" | "destructive"> = {
  ok: "success",
  warning: "warning",
  problem: "destructive",
};

const stageIcons: Record<OrchestrationStage["key"], LucideIcon> = {
  intake: ClipboardList,
  fulfillment: ShieldCheck,
  shipping: Truck,
  completed: PackageCheck,
};

export function OrchestrationMap({ stages, isLoading }: OrchestrationMapProps) {
  const t = useTranslations("dashboard");

  return (
    <Card className="overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-3">
        <CardTitle className="text-base">{t("operations.flowTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="p-4">
        {isLoading ? (
          <div className="grid gap-4 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-36 w-full rounded-lg" />
            ))}
          </div>
        ) : (
          <div className="grid gap-4 lg:grid-cols-4">
            {stages.map((stage, index) => {
              const Icon = stageIcons[stage.key];

              return (
                <div key={stage.key} className="relative">
                  {index < stages.length - 1 && (
                    <span
                      aria-hidden="true"
                      className="absolute -right-4 top-[4.75rem] hidden w-4 items-center justify-center text-muted-foreground lg:flex"
                    >
                      <span className="absolute h-px w-full bg-border" />
                      <ChevronRight className="relative h-4 w-4 rounded-full bg-card" />
                    </span>
                  )}
                  <Link
                    href={stage.href}
                    className={cn(
                      "group flex min-h-36 flex-col justify-between rounded-lg border bg-background p-4 shadow-[0_18px_50px_-38px_rgba(15,23,42,0.55)] transition-colors hover:border-primary/40 hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                      stage.health === "problem" && "border-destructive/40",
                      stage.health === "warning" && "border-warning/50",
                    )}
                  >
                    <span className="flex items-start justify-between gap-3">
                      <span className="flex min-w-0 items-center gap-3">
                        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border bg-card text-info">
                          <Icon className="h-5 w-5" aria-hidden="true" />
                        </span>
                        <span className="min-w-0">
                          <span className="block text-sm font-semibold">
                            {t(stage.labelKey)}
                          </span>
                          <span className="mt-1 block text-xs text-muted-foreground">
                            {t("operations.stageHint")}
                          </span>
                        </span>
                      </span>
                      <StageStatusIcon
                        health={stage.health}
                        className={cn(
                          "h-4 w-4 shrink-0",
                          healthIconClassName[stage.health],
                        )}
                      />
                    </span>

                    <span>
                      <span className="block text-3xl font-semibold tabular-nums">
                        {stage.count}
                      </span>
                      <span className="mt-3 flex flex-wrap items-center gap-2">
                        <Badge variant={healthBadgeVariant[stage.health]}>
                          {t(`operations.health.${stage.health}`)}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {stage.exceptionCount > 0
                            ? t("operations.exceptionCount", { count: stage.exceptionCount })
                            : t("operations.noExceptions")}
                        </span>
                      </span>
                    </span>
                  </Link>
                </div>
              );
            })}
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
