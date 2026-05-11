"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { useLocale, useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { OperationsActivityItem } from "@/lib/operations-dashboard";

interface OperationsActivityProps {
  items: OperationsActivityItem[];
  isLoading: boolean;
}

export function OperationsActivity({ items, isLoading }: OperationsActivityProps) {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const dateLocale = locale === "pl" ? pl : enUS;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("operations.activityTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className="h-14 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("operations.noActivity")}
          </p>
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                className="block rounded-lg border bg-background p-3 transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <span className="flex items-start justify-between gap-3">
                  <span className="min-w-0">
                    <span className="block text-sm font-medium">
                      {t(item.titleKey, item.values)}
                    </span>
                    <span className="mt-1 block text-sm text-muted-foreground">
                      {t(item.descriptionKey, item.values)}
                    </span>
                  </span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {formatDistanceToNow(new Date(item.createdAt), {
                      addSuffix: true,
                      locale: dateLocale,
                    })}
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
