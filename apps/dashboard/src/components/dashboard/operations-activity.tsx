"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { ArrowUpRight, Clock3 } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
    <Card className="overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-3">
        <CardTitle className="text-base">{t("operations.activityTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {isLoading ? (
          <div className="space-y-3 p-5">
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className="h-12 w-full rounded-md" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="flex min-h-48 items-center justify-center px-6 text-center">
            <p className="text-sm text-muted-foreground">
              {t("operations.noActivity")}
            </p>
          </div>
        ) : (
          <div className="divide-y">
            <div className="hidden grid-cols-[104px_120px_minmax(0,1fr)_32px] gap-4 px-5 py-2 text-xs font-medium uppercase text-muted-foreground md:grid">
              <span>{t("operations.activityColumns.time")}</span>
              <span>{t("operations.activityColumns.source")}</span>
              <span>{t("operations.activityColumns.event")}</span>
              <span aria-hidden="true" />
            </div>
            {items.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                className="grid gap-2 bg-card px-5 py-2 transition-colors hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:grid-cols-[104px_120px_minmax(0,1fr)_32px] md:items-center md:gap-4"
              >
                <span className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Clock3 className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>
                    {formatDistanceToNow(new Date(item.createdAt), {
                      addSuffix: true,
                      locale: dateLocale,
                    })}
                  </span>
                </span>
                <span className="truncate text-sm text-muted-foreground">{item.source}</span>
                <span className="min-w-0 text-sm">
                  <span className="font-semibold">
                    {t(item.titleKey, item.values)}
                  </span>
                  <span className="text-muted-foreground">
                    {" · "}
                    {t(item.descriptionKey, item.values)}
                  </span>
                </span>
                <ArrowUpRight
                  className="hidden h-4 w-4 justify-self-end text-muted-foreground md:block"
                  aria-hidden="true"
                />
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
