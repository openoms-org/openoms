"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { AlertTriangle, CheckCircle2, CircleAlert } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { OperationalException } from "@/lib/operations-dashboard";

interface OperationalExceptionsProps {
  exceptions: OperationalException[];
  isLoading: boolean;
}

export function OperationalExceptions({
  exceptions,
  isLoading,
}: OperationalExceptionsProps) {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const dateLocale = locale === "pl" ? pl : enUS;

  return (
    <Card className="h-full overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <CardTitle className="text-base">{t("operations.exceptionsTitle")}</CardTitle>
          </div>
          {!isLoading && (
            <Badge variant={exceptions.length > 0 ? "warning" : "success"}>
              {exceptions.length}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {isLoading ? (
          <div className="space-y-3 p-5">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-20 w-full rounded-lg" />
            ))}
          </div>
        ) : exceptions.length === 0 ? (
          <div className="flex min-h-72 flex-col items-center justify-center px-8 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-lg bg-success/10">
              <CheckCircle2 className="h-6 w-6 text-success" aria-hidden="true" />
            </span>
            <p className="mt-3 font-medium">{t("operations.noWorkTitle")}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("operations.noWorkDescription")}
            </p>
          </div>
        ) : (
          <div className="divide-y">
            {exceptions.map((item) => (
              <Link
                key={item.id}
                href={item.primaryHref}
                className="flex gap-3 bg-card p-4 transition-colors hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <ExceptionIcon
                  severity={item.severity}
                  className={cn(
                    "mt-0.5 h-5 w-5 shrink-0",
                    item.severity === "problem" ? "text-destructive" : "text-warning",
                  )}
                />
                <span className="min-w-0 flex-1">
                  <span className="flex items-start justify-between gap-3">
                    <span className="block text-sm font-semibold">
                      {t(item.titleKey, item.values)}
                    </span>
                    <Badge
                      variant={item.severity === "problem" ? "destructive" : "warning"}
                      className="shrink-0"
                    >
                      {t(`operations.health.${item.severity}`)}
                    </Badge>
                  </span>
                  <span className="mt-1 block text-sm text-muted-foreground">
                    {t(item.descriptionKey, item.values)}
                  </span>
                  {item.createdAt && (
                    <span className="mt-2 block text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(item.createdAt), {
                        addSuffix: true,
                        locale: dateLocale,
                      })}
                    </span>
                  )}
                </span>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ExceptionIcon({
  severity,
  className,
}: {
  severity: OperationalException["severity"];
  className?: string;
}) {
  if (severity === "problem") {
    return <CircleAlert aria-hidden="true" className={className} />;
  }

  return <AlertTriangle aria-hidden="true" className={className} />;
}
