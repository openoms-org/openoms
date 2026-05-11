"use client";

import Link from "next/link";
import { AlertTriangle, CheckCircle2, CircleAlert } from "lucide-react";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("operations.exceptionsTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-16 w-full" />
            ))}
          </div>
        ) : exceptions.length === 0 ? (
          <div className="flex min-h-40 flex-col items-center justify-center text-center">
            <CheckCircle2 className="h-8 w-8 text-success" aria-hidden="true" />
            <p className="mt-3 font-medium">{t("operations.noWorkTitle")}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("operations.noWorkDescription")}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {exceptions.map((item) => (
              <Link
                key={item.id}
                href={item.primaryHref}
                className="flex gap-3 rounded-lg border bg-background p-3 transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <ExceptionIcon
                  severity={item.severity}
                  className={cn(
                    "mt-0.5 h-4 w-4 shrink-0",
                    item.severity === "problem" ? "text-destructive" : "text-warning",
                  )}
                />
                <span className="min-w-0">
                  <span className="block text-sm font-medium">
                    {t(item.titleKey, item.values)}
                  </span>
                  <span className="mt-1 block text-sm text-muted-foreground">
                    {t(item.descriptionKey, item.values)}
                  </span>
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
